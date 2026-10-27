package configuration

import (
	"context"
	"errors"
	"math"
	"strconv"

	"github.com/jackc/pgx/v5"
)

const priceRecalcLockKey int64 = 90210

type priceResult struct {
	price  float64
	priced bool
}

type priceResolver struct {
	ctx      context.Context
	q        querier
	rates    map[string]float64
	boms     map[string]*Bom
	defaults map[string]string
	bomCache map[string]priceResult
	revCache map[string]priceResult
	visiting map[string]bool
}

func newPriceResolver(ctx context.Context, q querier) (*priceResolver, error) {
	p := &priceResolver{
		ctx:      ctx,
		q:        q,
		rates:    map[string]float64{},
		boms:     map[string]*Bom{},
		defaults: map[string]string{},
		bomCache: map[string]priceResult{},
		revCache: map[string]priceResult{},
		visiting: map[string]bool{},
	}

	rateRows, err := q.Query(ctx, `SELECT id, hourly_rate FROM operation_classes`)
	if err != nil {
		return nil, err
	}
	for rateRows.Next() {
		var id, rate string
		if err := rateRows.Scan(&id, &rate); err != nil {
			rateRows.Close()
			return nil, err
		}
		p.rates[id] = parseNumber(rate)
	}
	if err := rateRows.Err(); err != nil {
		rateRows.Close()
		return nil, err
	}
	rateRows.Close()

	bomRows, err := q.Query(ctx, `SELECT id, product_id, is_default, price, price_manual FROM boms`)
	if err != nil {
		return nil, err
	}
	for bomRows.Next() {
		b := &Bom{}
		if err := bomRows.Scan(&b.ID, &b.ProductID, &b.IsDefault, &b.Price, &b.PriceManual); err != nil {
			bomRows.Close()
			return nil, err
		}
		p.boms[b.ID] = b
		if b.IsDefault {
			p.defaults[b.ProductID] = b.ID
		}
	}
	if err := bomRows.Err(); err != nil {
		bomRows.Close()
		return nil, err
	}
	bomRows.Close()
	return p, nil
}

func (p *priceResolver) bomPrice(bomID string) (priceResult, error) {
	if cached, ok := p.bomCache[bomID]; ok {
		return cached, nil
	}
	bom, ok := p.boms[bomID]
	if !ok {
		return priceResult{}, ErrNotFound
	}
	if bom.PriceManual {
		res := priceResult{price: parseNumber(bom.Price), priced: true}
		p.bomCache[bomID] = res
		return res, nil
	}

	key := "bom:" + bomID
	if p.visiting[key] {
		return priceResult{}, ErrRouteCycle
	}
	p.visiting[key] = true
	defer delete(p.visiting, key)

	rows, err := listRouteRowsQ(p.ctx, p.q, bomID)
	if err != nil {
		return priceResult{}, err
	}
	total := 0.0
	for _, row := range rows {
		cost, err := p.rowCost(row)
		if err != nil {
			return priceResult{}, err
		}
		total += cost
	}
	res := priceResult{price: total, priced: len(rows) > 0}
	p.bomCache[bomID] = res
	return res, nil
}

func (p *priceResolver) revisionPrice(bomID string, revision int) (priceResult, error) {
	key := "rev:" + bomID + ":" + strconv.Itoa(revision)
	if cached, ok := p.revCache[key]; ok {
		return cached, nil
	}
	if p.visiting[key] {
		return priceResult{}, ErrRouteCycle
	}
	p.visiting[key] = true
	defer delete(p.visiting, key)

	rows, err := loadRevisionRouteRowsQ(p.ctx, p.q, bomID, revision)
	if err != nil {
		return priceResult{}, err
	}
	if len(rows) == 0 {
		bom, ok := p.boms[bomID]
		if ok && bom.PriceManual {
			res := priceResult{price: parseNumber(bom.Price), priced: true}
			p.revCache[key] = res
			return res, nil
		}
		res := priceResult{}
		p.revCache[key] = res
		return res, nil
	}
	total := 0.0
	for _, row := range rows {
		cost, err := p.rowCost(row)
		if err != nil {
			return priceResult{}, err
		}
		total += cost
	}
	res := priceResult{price: total, priced: true}
	p.revCache[key] = res
	return res, nil
}

func (p *priceResolver) componentPrice(productID string, referenceBomID *string, referenceRevision *int) (priceResult, error) {
	if referenceBomID != nil {
		if referenceRevision == nil || *referenceRevision <= 0 {
			return priceResult{}, ErrInvalidRouteRow
		}
		return p.revisionPrice(*referenceBomID, *referenceRevision)
	}
	bomID, ok := p.defaults[productID]
	if !ok {
		return priceResult{}, nil
	}
	return p.bomPrice(bomID)
}

func (p *priceResolver) rowCost(row *RouteRow) (float64, error) {
	material, err := p.componentPrice(row.ProductID, row.ReferenceBomID, row.ReferenceRevision)
	if err != nil {
		return 0, err
	}
	labor := parseNumber(row.LaborNorm) * p.rates[row.OperationClassID]
	return parseNumber(row.Quantity)*material.price + labor, nil
}

func loadRevisionRouteRowsQ(ctx context.Context, q querier, bomID string, revision int) ([]*RouteRow, error) {
	const query = `SELECT routes FROM bom_revisions WHERE bom_id = $1 AND revision = $2`
	var raw []byte
	err := q.QueryRow(ctx, query, bomID, revision).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrReferenceNotFound
	}
	if err != nil {
		return nil, err
	}
	return decodeRouteRowSnapshots(bomID, raw)
}

func recalcPricesTx(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, priceRecalcLockKey); err != nil {
		return err
	}
	resolver, err := newPriceResolver(ctx, tx)
	if err != nil {
		return err
	}
	for id, bom := range resolver.boms {
		if bom.PriceManual {
			rows, err := listRouteRowsQ(ctx, tx, id)
			if err != nil {
				return err
			}
			if len(rows) > 0 {
				return ErrManualPriceWithRoutes
			}
			continue
		}
		result, err := resolver.bomPrice(id)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE boms SET price = $2 WHERE id = $1 AND price_manual = false`, id, formatMoney(result.price)); err != nil {
			return err
		}
	}
	return nil
}

func parseNumber(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func formatMoney(f float64) string {
	return strconv.FormatFloat(math.Round(f*100)/100, 'f', 2, 64)
}
