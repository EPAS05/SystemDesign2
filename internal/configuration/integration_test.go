package configuration

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRoutingIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	const truncate = `TRUNCATE bom_revisions, bom_route_rows, bom_rows, boms, operation_classes, equipment, tools RESTART IDENTITY CASCADE;`
	if _, err := pool.Exec(ctx, truncate); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), truncate)
	})

	svc := NewService(NewRepo(pool))

	productID := "00000000-0000-0000-0000-000000000001"
	productAID := "00000000-0000-0000-0000-000000000002"
	productBID := "00000000-0000-0000-0000-000000000003"

	cut, err := svc.CreateOperationClass(ctx, "CUT", "Cut", "500")
	if err != nil {
		t.Fatal(err)
	}
	weld, err := svc.CreateOperationClass(ctx, "WELD", "Weld", "800")
	if err != nil {
		t.Fatal(err)
	}
	assy, err := svc.CreateOperationClass(ctx, "ASSY", "Assembly", "600")
	if err != nil {
		t.Fatal(err)
	}
	press, err := svc.CreateEquipment(ctx, "PRESS", "Press")
	if err != nil {
		t.Fatal(err)
	}
	die, err := svc.CreateTool(ctx, "DIE", "Die")
	if err != nil {
		t.Fatal(err)
	}

	bomDefault, err := svc.CreateBom(ctx, productID, "", true)
	if err != nil {
		t.Fatal(err)
	}
	route1, err := svc.AddRouteRow(ctx, bomDefault.ID, RouteRowInput{
		Seq:              10,
		OperationClassID: cut.ID,
		EquipmentID:      press.ID,
		ToolID:           die.ID,
		ProductID:        productAID,
		Quantity:         "10",
		LaborNorm:        "0.2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if route1.OperationClassName != "Cut" || route1.EquipmentName == nil || *route1.EquipmentName != "Press" || route1.ToolName == nil || *route1.ToolName != "Die" {
		t.Fatalf("unexpected route1: %+v", route1)
	}

	route2, err := svc.AddRouteRow(ctx, bomDefault.ID, RouteRowInput{
		Seq:              20,
		OperationClassID: weld.ID,
		ProductID:        productBID,
		Quantity:         "2",
		LaborNorm:        "0.5",
	})
	if err != nil {
		t.Fatal(err)
	}
	if route2.Seq != 20 {
		t.Fatalf("unexpected route2: %+v", route2)
	}

	bomA, err := svc.CreateBom(ctx, productAID, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddRouteRow(ctx, bomA.ID, RouteRowInput{
		Seq:              10,
		OperationClassID: cut.ID,
		ProductID:        productBID,
		Quantity:         "3",
		LaborNorm:        "0.1",
	}); err != nil {
		t.Fatal(err)
	}

	bomB, err := svc.CreateBom(ctx, productBID, "", true)
	if err != nil {
		t.Fatal(err)
	}
	bomB, err = svc.UpdateBomPrice(ctx, bomB.ID, "42.50", true)
	if err != nil {
		t.Fatal(err)
	}
	if bomB.Price != "42.50" || !bomB.PriceManual {
		t.Fatalf("unexpected manual bom: %+v", bomB)
	}

	defaultBom, err := svc.GetBom(ctx, bomDefault.ID)
	if err != nil {
		t.Fatal(err)
	}
	if defaultBom.Price != "2360.00" {
		t.Fatalf("default price = %s, want 2360.00", defaultBom.Price)
	}

	if _, err := svc.UpdateBomPrice(ctx, bomDefault.ID, "1", true); !errors.Is(err, ErrManualPriceWithRoutes) {
		t.Fatalf("got %v, want ErrManualPriceWithRoutes", err)
	}

	bomMod, err := svc.CreateBom(ctx, productID, "Mod", false)
	if err != nil {
		t.Fatal(err)
	}
	inherited, err := svc.ListRouteRows(ctx, bomMod.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(inherited) != 2 {
		t.Fatalf("inherited rows = %d, want 2", len(inherited))
	}
	for _, row := range inherited {
		if !row.Inherited {
			t.Fatalf("expected inherited row: %+v", row)
		}
	}

	if _, err := svc.AddRouteRow(ctx, bomMod.ID, RouteRowInput{
		Seq:              15,
		OperationClassID: assy.ID,
		ProductID:        productAID,
		Quantity:         "1",
		LaborNorm:        "0.3",
	}); err != nil {
		t.Fatal(err)
	}

	updated, err := svc.UpdateRouteRow(ctx, bomMod.ID, route1.ID, RouteRowInput{
		Seq:              10,
		OperationClassID: cut.ID,
		ProductID:        productAID,
		Quantity:         "10",
		LaborNorm:        "0.4",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Inherited {
		t.Fatalf("override is still inherited: %+v", updated)
	}

	if _, err := svc.UpdateRouteRow(ctx, bomMod.ID, route2.ID, RouteRowInput{
		Seq:              20,
		OperationClassID: weld.ID,
		ProductID:        productBID,
		Quantity:         "0",
		LaborNorm:        "0",
		IsExcluded:       true,
	}); err != nil {
		t.Fatal(err)
	}

	modRows, err := svc.ListRouteRows(ctx, bomMod.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(modRows) != 2 {
		t.Fatalf("modification rows = %d, want 2", len(modRows))
	}
	for _, row := range modRows {
		if row.Seq == 20 {
			t.Fatalf("excluded row present: %+v", row)
		}
	}

	defaultRows, err := svc.ListRouteRows(ctx, bomDefault.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(defaultRows) != 2 {
		t.Fatalf("default rows = %d, want 2", len(defaultRows))
	}

	if _, err := svc.UpdateOperationClass(ctx, cut.ID, "CUT", "Cut", "600"); err != nil {
		t.Fatal(err)
	}
	defaultBom, err = svc.GetBom(ctx, bomDefault.ID)
	if err != nil {
		t.Fatal(err)
	}
	if defaultBom.Price != "2480.00" {
		t.Fatalf("default price = %s, want 2480.00", defaultBom.Price)
	}

	if _, err := svc.AddRouteRow(ctx, bomA.ID, RouteRowInput{
		Seq:              20,
		OperationClassID: weld.ID,
		ProductID:        productID,
		Quantity:         "1",
		LaborNorm:        "0",
	}); !errors.Is(err, ErrRouteCycle) {
		t.Fatalf("got %v, want ErrRouteCycle", err)
	}
	bomARows, err := svc.ListRouteRows(ctx, bomA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(bomARows) != 1 {
		t.Fatalf("bomA rows = %d, want 1", len(bomARows))
	}

	revisions, err := svc.ListBomRevisions(ctx, bomDefault.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) < 3 {
		t.Fatalf("revisions = %d, want at least 3", len(revisions))
	}
	_, routeRows, _, err := svc.GetBomRevision(ctx, bomDefault.ID, revisions[len(revisions)-1].Revision)
	if err != nil {
		t.Fatal(err)
	}
	if len(routeRows) != 2 {
		t.Fatalf("revision route rows = %d, want 2", len(routeRows))
	}
	if routeRows[0].Seq != 10 || routeRows[1].Seq != 20 {
		t.Fatalf("revision seqs = %d, %d", routeRows[0].Seq, routeRows[1].Seq)
	}
	if routeRows[0].OperationClassName != "Cut" {
		t.Fatalf("operation class name = %s", routeRows[0].OperationClassName)
	}
	if routeRows[0].EquipmentName == nil || *routeRows[0].EquipmentName != "Press" {
		t.Fatalf("equipment name = %v", routeRows[0].EquipmentName)
	}
	if routeRows[0].ToolName == nil || *routeRows[0].ToolName != "Die" {
		t.Fatalf("tool name = %v", routeRows[0].ToolName)
	}
	if routeRows[0].Inherited {
		t.Fatal("revision route row is inherited")
	}
}
