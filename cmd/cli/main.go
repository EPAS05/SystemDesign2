package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Node struct {
	ID       string `json:"id"`
	ParentID string `json:"parentId"`
	Name     string `json:"name"`
}

type Product struct {
	ID     string `json:"id"`
	NodeID string `json:"nodeId"`
	Name   string `json:"name"`
}

type BomRow struct {
	ID                string `json:"id"`
	ProductID         string `json:"productId"`
	Quantity          string `json:"quantity"`
	IsBase            bool   `json:"isBase"`
	IsExcluded        bool   `json:"isExcluded"`
	ReferenceBomID    string `json:"referenceBomId,omitempty"`
	ReferenceRevision int    `json:"referenceRevision,omitempty"`
}

type Bom struct {
	ID          string   `json:"id"`
	ProductID   string   `json:"productId"`
	IsDefault   bool     `json:"isDefault"`
	Name        string   `json:"name"`
	Rows        []BomRow `json:"rows"`
	Price       string   `json:"price"`
	PriceManual bool     `json:"priceManual"`
}

type Revision struct {
	BomID     string     `json:"bomId"`
	Revision  int        `json:"revision"`
	ChangedAt string     `json:"changedAt"`
	RouteRows []RouteRow `json:"routeRows"`
}

type cli struct {
	base     string
	in       *bufio.Reader
	nodes    []Node
	products []Product
	boms     []Bom
}

func main() {
	addr := flag.String("addr", "http://localhost:8080", "адрес HTTP-шлюза")
	flag.Parse()
	c := &cli{base: strings.TrimRight(*addr, "/"), in: bufio.NewReader(os.Stdin)}
	c.run()
}

func (c *cli) run() {
	for {
		fmt.Println()
		fmt.Println("=== CLI каталога и конфигураций ===")
		fmt.Println("1. Узлы классификатора")
		fmt.Println("2. Изделия")
		fmt.Println("3. Конфигурации (BOM)")
		fmt.Println("4. Маршрутная технология")
		fmt.Println("0. Выход")
		switch c.ask("выбор> ") {
		case "1":
			c.nodesMenu()
		case "2":
			c.productsMenu()
		case "3":
			c.bomsMenu()
		case "4":
			c.routingMenu()
		case "0", "q":
			return
		}
	}
}

func (c *cli) ask(prompt string) string {
	fmt.Print(prompt)
	s, err := c.in.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

func (c *cli) askYes(prompt string, def bool) bool {
	s := strings.ToLower(c.ask(prompt))
	if s == "" {
		return def
	}
	return s == "y" || s == "yes" || s == "д" || s == "да"
}

func (c *cli) confirm(prompt string) bool {
	return c.askYes(prompt, false)
}

func (c *cli) send(method, path string, in, out any, show bool) error {
	var body io.Reader
	if in != nil {
		raw, err := json.MarshalIndent(in, "", "  ")
		if err != nil {
			return err
		}
		if show {
			fmt.Printf("→ %s %s\n%s\n", method, c.base+path, raw)
		}
		body = bytes.NewReader(raw)
	} else if show {
		fmt.Printf("→ %s %s\n", method, c.base+path)
	}
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if show {
		fmt.Printf("← %d\n", resp.StatusCode)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	if show && len(raw) > 0 {
		var buf bytes.Buffer
		if err := json.Indent(&buf, raw, "", "  "); err == nil {
			fmt.Println(buf.String())
		}
	}
	if out != nil && len(raw) > 0 {
		return json.Unmarshal(raw, out)
	}
	return nil
}

func (c *cli) fail(err error) {
	fmt.Println("!! ошибка:", err)
}

func (c *cli) upsertNodes(ns ...Node) {
	for _, n := range ns {
		found := false
		for i := range c.nodes {
			if c.nodes[i].ID == n.ID {
				c.nodes[i] = n
				found = true
				break
			}
		}
		if !found {
			c.nodes = append(c.nodes, n)
		}
	}
}

func (c *cli) upsertProducts(ps ...Product) {
	for _, p := range ps {
		found := false
		for i := range c.products {
			if c.products[i].ID == p.ID {
				c.products[i] = p
				found = true
				break
			}
		}
		if !found {
			c.products = append(c.products, p)
		}
	}
}

func (c *cli) upsertBoms(bs ...Bom) {
	for _, b := range bs {
		found := false
		for i := range c.boms {
			if c.boms[i].ID == b.ID {
				c.boms[i] = b
				found = true
				break
			}
		}
		if !found {
			c.boms = append(c.boms, b)
		}
	}
}

func (c *cli) dropNode(id string) {
	out := c.nodes[:0]
	for _, n := range c.nodes {
		if n.ID != id {
			out = append(out, n)
		}
	}
	c.nodes = out
}

func (c *cli) dropProduct(id string) {
	out := c.products[:0]
	for _, p := range c.products {
		if p.ID != id {
			out = append(out, p)
		}
	}
	c.products = out
}

func (c *cli) dropBom(id string) {
	out := c.boms[:0]
	for _, b := range c.boms {
		if b.ID != id {
			out = append(out, b)
		}
	}
	c.boms = out
}

func (c *cli) nodeName(id string) string {
	for _, n := range c.nodes {
		if n.ID == id {
			return n.Name
		}
	}
	return id
}

func (c *cli) productName(id string) string {
	for _, p := range c.products {
		if p.ID == id {
			return p.Name
		}
	}
	return id
}

func (c *cli) refreshNodes() {
	var roots struct {
		Nodes []Node `json:"nodes"`
	}
	if err := c.send("GET", "/v1/nodes", nil, &roots, false); err != nil {
		c.fail(err)
	} else {
		c.upsertNodes(roots.Nodes...)
	}
	for round := 0; round < 100; round++ {
		before := len(c.nodes)
		for _, n := range append([]Node(nil), c.nodes...) {
			var resp struct {
				Nodes []Node `json:"nodes"`
			}
			if err := c.send("GET", "/v1/nodes/"+n.ID+"/children", nil, &resp, false); err != nil {
				c.fail(err)
				continue
			}
			c.upsertNodes(resp.Nodes...)
		}
		if len(c.nodes) == before {
			return
		}
	}
}

func (c *cli) printNodes() {
	if len(c.nodes) == 0 {
		fmt.Println("(список пуст)")
	}
	for _, n := range c.nodes {
		parent := "—"
		if n.ParentID != "" {
			parent = c.nodeName(n.ParentID)
		}
		fmt.Printf("%s  %s  (parent: %s)\n", n.ID, n.Name, parent)
	}
}

func (c *cli) selectNode() string {
	for {
		fmt.Println("-- известные узлы --")
		for i, n := range c.nodes {
			fmt.Printf("%d) %-30s %s\n", i+1, n.Name, n.ID)
		}
		in := c.ask("номер или вставьте ID (r - обновить список)> ")
		switch {
		case in == "":
			continue
		case in == "r":
			c.refreshNodes()
		default:
			if i, err := strconv.Atoi(in); err == nil {
				if i >= 1 && i <= len(c.nodes) {
					return c.nodes[i-1].ID
				}
				fmt.Println("нет такого номера")
				continue
			}
			return in
		}
	}
}

func (c *cli) selectProduct() string {
	for {
		fmt.Println("-- известные изделия --")
		for i, p := range c.products {
			fmt.Printf("%d) %-30s %s\n", i+1, p.Name, p.ID)
		}
		in := c.ask("номер или ID (l - загрузить изделия узла)> ")
		switch {
		case in == "":
			continue
		case in == "l":
			nodeID := c.selectNode()
			var resp struct {
				Products []Product `json:"products"`
			}
			if err := c.send("GET", "/v1/nodes/"+nodeID+"/products", nil, &resp, false); err != nil {
				c.fail(err)
				continue
			}
			c.upsertProducts(resp.Products...)
		default:
			if i, err := strconv.Atoi(in); err == nil {
				if i >= 1 && i <= len(c.products) {
					return c.products[i-1].ID
				}
				fmt.Println("нет такого номера")
				continue
			}
			return in
		}
	}
}

func (c *cli) selectBom() string {
	for {
		fmt.Println("-- известные BOM --")
		for i, b := range c.boms {
			kind := "базовая"
			if !b.IsDefault {
				kind = "модификация " + b.Name
			}
			fmt.Printf("%d) %s [%s] %s  price=%s\n", i+1, c.productName(b.ProductID), kind, b.ID, b.Price)
		}
		in := c.ask("номер или ID (l - загрузить BOM)> ")
		switch {
		case in == "":
			continue
		case in == "l":
			productID := c.selectProduct()
			var resp struct {
				Boms []Bom `json:"boms"`
			}
			if err := c.send("GET", "/v1/products/"+productID+"/boms", nil, &resp, false); err != nil {
				c.fail(err)
				continue
			}
			c.upsertBoms(resp.Boms...)
		default:
			if i, err := strconv.Atoi(in); err == nil {
				if i >= 1 && i <= len(c.boms) {
					return c.boms[i-1].ID
				}
				fmt.Println("нет такого номера")
				continue
			}
			return in
		}
	}
}

func (c *cli) selectRow(bomID string) string {
	var resp struct {
		Bom Bom `json:"bom"`
	}
	if err := c.send("GET", "/v1/boms/"+bomID, nil, &resp, false); err != nil {
		c.fail(err)
		return ""
	}
	rows := resp.Bom.Rows
	for {
		fmt.Println("-- строки BOM --")
		for i, r := range rows {
			fmt.Printf("%d) %-30s %s  qty=%s\n", i+1, c.productName(r.ProductID), r.ID, r.Quantity)
		}
		in := c.ask("номер строки или ID> ")
		if in == "" {
			continue
		}
		if i, err := strconv.Atoi(in); err == nil {
			if i >= 1 && i <= len(rows) {
				return rows[i-1].ID
			}
			fmt.Println("нет такого номера")
			continue
		}
		return in
	}
}

func (c *cli) loadBom(id string) (Bom, error) {
	var resp struct {
		Bom Bom `json:"bom"`
	}
	if err := c.send("GET", "/v1/boms/"+id, nil, &resp, false); err != nil {
		return Bom{}, err
	}
	c.upsertBoms(resp.Bom)
	return resp.Bom, nil
}

func (c *cli) askReference(bomID, productID string, current *BomRow) (string, int, bool) {
	keepBomID := ""
	keepRevision := 0
	prompt := "ссылка на BOM (Enter — без ссылки, иначе выбрать)> "
	if current != nil && current.ReferenceBomID != "" {
		keepBomID = current.ReferenceBomID
		keepRevision = current.ReferenceRevision
		prompt = fmt.Sprintf("ссылка на BOM [%s r%d] (Enter — оставить, - — убрать, иначе выбрать)> ", keepBomID, keepRevision)
	}
	keep := func() (string, int, bool) {
		if keepBomID != "" {
			return keepBomID, keepRevision, true
		}
		return "", 0, false
	}
	in := c.ask(prompt)
	if in == "-" {
		return "", 0, false
	}
	if in == "" {
		return keep()
	}

	var resp struct {
		Boms []Bom `json:"boms"`
	}
	if err := c.send("GET", "/v1/products/"+productID+"/boms", nil, &resp, false); err != nil {
		c.fail(err)
		return keep()
	}
	c.upsertBoms(resp.Boms...)
	candidates := make([]Bom, 0, len(resp.Boms))
	for _, b := range resp.Boms {
		if b.ID != bomID {
			candidates = append(candidates, b)
		}
	}
	if len(candidates) == 0 {
		fmt.Println("нет других BOM этого изделия")
		return keep()
	}

	for {
		fmt.Println("-- BOM изделия --")
		for i, b := range candidates {
			kind := "базовая"
			if !b.IsDefault {
				kind = "модификация " + b.Name
			}
			fmt.Printf("%d) %-30s %s\n", i+1, kind, b.ID)
		}
		in := c.ask("номер или ID BOM (Enter — отмена)> ")
		if in == "" {
			return keep()
		}
		var selected *Bom
		if i, err := strconv.Atoi(in); err == nil {
			if i >= 1 && i <= len(candidates) {
				selected = &candidates[i-1]
			}
		} else {
			for i := range candidates {
				if candidates[i].ID == in {
					selected = &candidates[i]
					break
				}
			}
		}
		if selected == nil {
			fmt.Println("нет такого BOM")
			continue
		}

		var revResp struct {
			Revisions []Revision `json:"revisions"`
		}
		if err := c.send("GET", "/v1/boms/"+selected.ID+"/revisions", nil, &revResp, false); err != nil {
			c.fail(err)
			return keep()
		}
		if len(revResp.Revisions) == 0 {
			fmt.Println("у BOM нет ревизий")
			return keep()
		}
		for {
			fmt.Println("-- ревизии BOM --")
			for i, rev := range revResp.Revisions {
				fmt.Printf("%d) r%d  %s\n", i+1, rev.Revision, rev.ChangedAt)
			}
			in := c.ask("номер из списка или номер ревизии (Enter — отмена)> ")
			if in == "" {
				return keep()
			}
			n, err := strconv.Atoi(in)
			if err != nil {
				fmt.Println("нужен номер")
				continue
			}
			if n >= 1 && n <= len(revResp.Revisions) {
				return selected.ID, revResp.Revisions[n-1].Revision, true
			}
			for _, rev := range revResp.Revisions {
				if rev.Revision == n {
					return selected.ID, n, true
				}
			}
			fmt.Println("нет такой ревизии")
		}
	}
}

func (c *cli) askRow(bomID string, defaultBom bool, current *BomRow) map[string]any {
	productID := c.selectProduct()
	row := map[string]any{"productId": productID}
	excluded := false
	if defaultBom {
		isBase := true
		if current != nil {
			isBase = current.IsBase
		}
		prompt := "базовая строка? (Y/n)> "
		if !isBase {
			prompt = "базовая строка? (y/N)> "
		}
		row["isBase"] = c.askYes(prompt, isBase)
	} else {
		if current != nil {
			excluded = current.IsExcluded
		}
		prompt := "удалить унаследованную строку? (y/N)> "
		if excluded {
			prompt = "удалить унаследованную строку? (Y/n)> "
		}
		excluded = c.askYes(prompt, excluded)
		row["isExcluded"] = excluded
	}
	if excluded {
		row["quantity"] = "0"
		return row
	}

	quantity := ""
	if current != nil && !current.IsExcluded {
		quantity = current.Quantity
	}
	qtyPrompt := "количество (напр. 2)> "
	if quantity != "" {
		qtyPrompt = fmt.Sprintf("количество [%s]> ", quantity)
	}
	for {
		in := c.ask(qtyPrompt)
		if in != "" {
			quantity = in
			break
		}
		if quantity != "" {
			break
		}
		fmt.Println("количество обязательно")
	}
	row["quantity"] = quantity
	if refBomID, refRev, ok := c.askReference(bomID, productID, current); ok {
		row["referenceBomId"] = refBomID
		row["referenceRevision"] = refRev
	}
	return row
}

func (c *cli) nodesMenu() {
	for {
		fmt.Println()
		fmt.Println("--- Узлы классификатора ---")
		fmt.Println("1. Список узлов (подгрузить дочерние и показать)")
		fmt.Println("2. Создать узел")
		fmt.Println("3. Показать узел")
		fmt.Println("4. Переименовать узел")
		fmt.Println("5. Удалить узел")
		fmt.Println("0. Назад")
		switch c.ask("выбор> ") {
		case "1":
			c.refreshNodes()
			c.printNodes()
		case "2":
			parentID := ""
			if c.confirm("Дочерний узел? (y/N)> ") {
				parentID = c.selectNode()
			}
			name := c.ask("имя узла> ")
			var resp struct {
				Node Node `json:"node"`
			}
			if err := c.send("POST", "/v1/nodes", map[string]string{"name": name, "parentId": parentID}, &resp, true); err != nil {
				c.fail(err)
				continue
			}
			c.upsertNodes(resp.Node)
		case "3":
			id := c.selectNode()
			var resp struct {
				Node Node `json:"node"`
			}
			if err := c.send("GET", "/v1/nodes/"+id, nil, &resp, true); err != nil {
				c.fail(err)
				continue
			}
			c.upsertNodes(resp.Node)
		case "4":
			id := c.selectNode()
			name := c.ask("новое имя> ")
			var resp struct {
				Node Node `json:"node"`
			}
			if err := c.send("PATCH", "/v1/nodes/"+id, map[string]string{"name": name}, &resp, true); err != nil {
				c.fail(err)
				continue
			}
			c.upsertNodes(resp.Node)
		case "5":
			id := c.selectNode()
			if err := c.send("DELETE", "/v1/nodes/"+id, nil, nil, true); err != nil {
				c.fail(err)
				continue
			}
			c.dropNode(id)
		case "0":
			return
		}
	}
}

func (c *cli) productsMenu() {
	for {
		fmt.Println()
		fmt.Println("--- Изделия ---")
		fmt.Println("1. Список изделий узла")
		fmt.Println("2. Создать изделие")
		fmt.Println("3. Показать изделие")
		fmt.Println("4. Переименовать изделие")
		fmt.Println("5. Удалить изделие")
		fmt.Println("0. Назад")
		switch c.ask("выбор> ") {
		case "1":
			nodeID := c.selectNode()
			var resp struct {
				Products []Product `json:"products"`
			}
			if err := c.send("GET", "/v1/nodes/"+nodeID+"/products", nil, &resp, true); err != nil {
				c.fail(err)
				continue
			}
			c.upsertProducts(resp.Products...)
		case "2":
			nodeID := c.selectNode()
			name := c.ask("название изделия> ")
			var resp struct {
				Product Product `json:"product"`
			}
			if err := c.send("POST", "/v1/products", map[string]string{"nodeId": nodeID, "name": name}, &resp, true); err != nil {
				c.fail(err)
				continue
			}
			c.upsertProducts(resp.Product)
		case "3":
			id := c.selectProduct()
			var resp struct {
				Product Product `json:"product"`
			}
			if err := c.send("GET", "/v1/products/"+id, nil, &resp, true); err != nil {
				c.fail(err)
				continue
			}
			c.upsertProducts(resp.Product)
		case "4":
			id := c.selectProduct()
			name := c.ask("новое название> ")
			var resp struct {
				Product Product `json:"product"`
			}
			if err := c.send("PATCH", "/v1/products/"+id, map[string]string{"name": name}, &resp, true); err != nil {
				c.fail(err)
				continue
			}
			c.upsertProducts(resp.Product)
		case "5":
			id := c.selectProduct()
			if err := c.send("DELETE", "/v1/products/"+id, nil, nil, true); err != nil {
				c.fail(err)
				continue
			}
			c.dropProduct(id)
		case "0":
			return
		}
	}
}

func (c *cli) bomsMenu() {
	for {
		fmt.Println()
		fmt.Println("--- Конфигурации (BOM) ---")
		fmt.Println("1. BOM изделия")
		fmt.Println("2. Создать BOM")
		fmt.Println("3. Удалить BOM")
		fmt.Println("4. Добавить строку")
		fmt.Println("5. Изменить строку")
		fmt.Println("6. Удалить строку")
		fmt.Println("7. Заменить все строки (собрать список)")
		fmt.Println("8. История ревизий")
		fmt.Println("9. Расчёт всей потребности (expand)")
		fmt.Println("0. Назад")
		switch c.ask("выбор> ") {
		case "1":
			productID := c.selectProduct()
			var resp struct {
				Boms []Bom `json:"boms"`
			}
			if err := c.send("GET", "/v1/products/"+productID+"/boms", nil, &resp, true); err != nil {
				c.fail(err)
				continue
			}
			c.upsertBoms(resp.Boms...)
		case "2":
			productID := c.selectProduct()
			body := map[string]any{"isDefault": true}
			if c.ask("тип: b — базовая, i — модификация (b)> ") == "i" {
				body["isDefault"] = false
				body["name"] = c.ask("название модификации> ")
			}
			var resp struct {
				Bom Bom `json:"bom"`
			}
			if err := c.send("POST", "/v1/products/"+productID+"/boms", body, &resp, true); err != nil {
				c.fail(err)
				continue
			}
			c.upsertBoms(resp.Bom)
		case "3":
			id := c.selectBom()
			if err := c.send("DELETE", "/v1/boms/"+id, nil, nil, true); err != nil {
				c.fail(err)
				continue
			}
			c.dropBom(id)
		case "4":
			bomID := c.selectBom()
			bom, err := c.loadBom(bomID)
			if err != nil {
				c.fail(err)
				continue
			}
			body := c.askRow(bomID, bom.IsDefault, nil)
			if err := c.send("POST", "/v1/boms/"+bomID+"/rows", body, nil, true); err != nil {
				c.fail(err)
			}
		case "5":
			bomID := c.selectBom()
			bom, err := c.loadBom(bomID)
			if err != nil {
				c.fail(err)
				continue
			}
			rowID := c.selectRow(bomID)
			if rowID == "" {
				continue
			}
			var current *BomRow
			for i := range bom.Rows {
				if bom.Rows[i].ID == rowID {
					current = &bom.Rows[i]
					break
				}
			}
			if current == nil {
				fmt.Println("строка не найдена в выбранном BOM")
				continue
			}
			body := c.askRow(bomID, bom.IsDefault, current)
			if err := c.send("PATCH", "/v1/boms/rows/"+rowID, body, nil, true); err != nil {
				c.fail(err)
			}
		case "6":
			bomID := c.selectBom()
			rowID := c.selectRow(bomID)
			if err := c.send("DELETE", "/v1/boms/rows/"+rowID, nil, nil, true); err != nil {
				c.fail(err)
			}
		case "7":
			bomID := c.selectBom()
			bom, err := c.loadBom(bomID)
			if err != nil {
				c.fail(err)
				continue
			}
			rows := []map[string]any{}
			for {
				rows = append(rows, c.askRow(bomID, bom.IsDefault, nil))
				if !c.confirm("добавить ещё строку? (y/N)> ") {
					break
				}
			}
			if err := c.send("PUT", "/v1/boms/"+bomID+"/rows", map[string]any{"rows": rows}, nil, true); err != nil {
				c.fail(err)
			}
		case "8":
			bomID := c.selectBom()
			var resp struct {
				Revisions []Revision `json:"revisions"`
			}
			if err := c.send("GET", "/v1/boms/"+bomID+"/revisions", nil, &resp, false); err != nil {
				c.fail(err)
				continue
			}
			for _, r := range resp.Revisions {
				fmt.Printf("r%d  %s\n", r.Revision, r.ChangedAt)
			}
			num := c.ask("номер ревизии для просмотра (Enter — пропустить)> ")
			if num == "" {
				continue
			}
			var rev struct {
				Revision  int        `json:"revision"`
				ChangedAt string     `json:"changedAt"`
				Rows      []BomRow   `json:"rows"`
				RouteRows []RouteRow `json:"routeRows"`
			}
			if err := c.send("GET", "/v1/boms/"+bomID+"/revisions/"+num, nil, &rev, true); err != nil {
				c.fail(err)
				continue
			}
			fmt.Printf("-- состав ревизии r%d (%s) --\n", rev.Revision, rev.ChangedAt)
			for _, r := range rev.Rows {
				fmt.Printf("%-30s qty=%s\n", c.resolveProductName(r.ProductID), r.Quantity)
			}
			fmt.Printf("-- маршрут ревизии r%d --\n", rev.Revision)
			for _, r := range rev.RouteRows {
				fmt.Printf("seq=%d %-20s %-30s qty=%s labor=%s\n",
					r.Seq, r.OperationClassName, c.resolveProductName(r.ProductID), r.Quantity, r.LaborNorm)
			}
		case "9":
			bomID := c.selectBom()
			var resp struct {
				Rows []BomRow `json:"rows"`
			}
			if err := c.send("GET", "/v1/boms/"+bomID+"/expand", nil, &resp, true); err != nil {
				c.fail(err)
				continue
			}
			fmt.Println("-- потребность (все уровни) --")
			for _, r := range resp.Rows {
				fmt.Printf("%-30s %s  qty=%s\n", c.productName(r.ProductID), r.ProductID, r.Quantity)
			}
		case "0":
			return
		}
	}
}
