package main

import (
	"fmt"
	"strconv"
)

type OperationClass struct {
	ID         string `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	HourlyRate string `json:"hourlyRate"`
}

type Equipment struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type Tool struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type RouteRow struct {
	ID                 string `json:"id"`
	BomID              string `json:"bomId"`
	Seq                int    `json:"seq"`
	OperationClassID   string `json:"operationClassId"`
	OperationClassName string `json:"operationClassName"`
	EquipmentID        string `json:"equipmentId"`
	EquipmentName      string `json:"equipmentName"`
	ToolID             string `json:"toolId"`
	ToolName           string `json:"toolName"`
	ProductID          string `json:"productId"`
	Quantity           string `json:"quantity"`
	LaborNorm          string `json:"laborNorm"`
	IsBase             bool   `json:"isBase"`
	IsExcluded         bool   `json:"isExcluded"`
	ReferenceBomID     string `json:"referenceBomId"`
	ReferenceRevision  int    `json:"referenceRevision"`
	Inherited          bool   `json:"inherited"`
	SourceBomID        string `json:"sourceBomId"`
}

func (c *cli) resolveProductName(id string) string {
	if name := c.productName(id); name != id {
		return name
	}
	var resp struct {
		Product Product `json:"product"`
	}
	if err := c.send("GET", "/v1/products/"+id, nil, &resp, false); err == nil {
		c.upsertProducts(resp.Product)
		return resp.Product.Name
	}
	return id
}

func (c *cli) routingMenu() {
	for {
		fmt.Println()
		fmt.Println("--- Маршрутная технология ---")
		fmt.Println("1. Классы операций")
		fmt.Println("2. Оборудование")
		fmt.Println("3. Инструменты")
		fmt.Println("4. Маршрутные строки BOM")
		fmt.Println("5. Цена BOM")
		fmt.Println("0. Назад")
		switch c.ask("выбор> ") {
		case "1":
			c.operationClassesMenu()
		case "2":
			c.equipmentMenu()
		case "3":
			c.toolsMenu()
		case "4":
			c.routeRowsMenu()
		case "5":
			c.bomPriceMenu()
		case "0":
			return
		}
	}
}

func (c *cli) operationClassesMenu() {
	for {
		fmt.Println()
		fmt.Println("--- Классы операций ---")
		fmt.Println("1. Список")
		fmt.Println("2. Создать")
		fmt.Println("3. Изменить")
		fmt.Println("4. Удалить")
		fmt.Println("0. Назад")
		switch c.ask("выбор> ") {
		case "1":
			items := c.listOperationClasses()
			for _, item := range items {
				fmt.Printf("%s  %-20s %-30s rate=%s\n", item.ID, item.Code, item.Name, item.HourlyRate)
			}
		case "2":
			body := map[string]string{
				"code":       c.ask("код> "),
				"name":       c.ask("название> "),
				"hourlyRate": c.ask("ставка в час> "),
			}
			var resp struct {
				OperationClass OperationClass `json:"operationClass"`
			}
			if err := c.send("POST", "/v1/operation-classes", body, &resp, true); err != nil {
				c.fail(err)
			}
		case "3":
			item := c.selectOperationClass()
			if item == nil {
				continue
			}
			body := map[string]string{
				"code":       c.askDefault("код", item.Code),
				"name":       c.askDefault("название", item.Name),
				"hourlyRate": c.askDefault("ставка в час", item.HourlyRate),
			}
			var resp struct {
				OperationClass OperationClass `json:"operationClass"`
			}
			if err := c.send("PATCH", "/v1/operation-classes/"+item.ID, body, &resp, true); err != nil {
				c.fail(err)
			}
		case "4":
			item := c.selectOperationClass()
			if item == nil {
				continue
			}
			if c.confirm("удалить класс? (y/N)> ") {
				if err := c.send("DELETE", "/v1/operation-classes/"+item.ID, nil, nil, true); err != nil {
					c.fail(err)
				}
			}
		case "0":
			return
		}
	}
}

func (c *cli) listOperationClasses() []OperationClass {
	var resp struct {
		OperationClasses []OperationClass `json:"operationClasses"`
	}
	if err := c.send("GET", "/v1/operation-classes", nil, &resp, false); err != nil {
		c.fail(err)
		return nil
	}
	return resp.OperationClasses
}

func (c *cli) selectOperationClass() *OperationClass {
	items := c.listOperationClasses()
	if len(items) == 0 {
		fmt.Println("(список пуст)")
		return nil
	}
	for {
		for i, item := range items {
			fmt.Printf("%d) %-12s %-30s rate=%s\n", i+1, item.Code, item.Name, item.HourlyRate)
		}
		in := c.ask("номер или ID класса (Enter — отмена)> ")
		if in == "" {
			return nil
		}
		if i, err := strconv.Atoi(in); err == nil {
			if i >= 1 && i <= len(items) {
				return &items[i-1]
			}
		}
		for i := range items {
			if items[i].ID == in {
				return &items[i]
			}
		}
		fmt.Println("нет такого класса")
	}
}

func (c *cli) equipmentMenu() {
	c.simpleDictionaryMenu("Оборудование", "/v1/equipment", "equipment", func() (string, string) {
		return c.ask("код> "), c.ask("название> ")
	})
}

func (c *cli) toolsMenu() {
	c.simpleDictionaryMenu("Инструменты", "/v1/tools", "tools", func() (string, string) {
		return c.ask("код> "), c.ask("название> ")
	})
}

func (c *cli) simpleDictionaryMenu(title, path, listField string, askFields func() (string, string)) {
	for {
		fmt.Println()
		fmt.Printf("--- %s ---\n", title)
		fmt.Println("1. Список")
		fmt.Println("2. Создать")
		fmt.Println("3. Изменить")
		fmt.Println("4. Удалить")
		fmt.Println("0. Назад")
		switch c.ask("выбор> ") {
		case "1":
			items := c.listSimpleDictionary(path, listField)
			for _, item := range items {
				fmt.Printf("%s  %-20s %s\n", item.ID, item.Code, item.Name)
			}
		case "2":
			code, name := askFields()
			var resp map[string]any
			if err := c.send("POST", path, map[string]string{"code": code, "name": name}, &resp, true); err != nil {
				c.fail(err)
			}
		case "3":
			item := c.selectSimpleDictionary(path, listField)
			if item == nil {
				continue
			}
			code := c.askDefault("код", item.Code)
			name := c.askDefault("название", item.Name)
			var resp map[string]any
			if err := c.send("PATCH", path+"/"+item.ID, map[string]string{"code": code, "name": name}, &resp, true); err != nil {
				c.fail(err)
			}
		case "4":
			item := c.selectSimpleDictionary(path, listField)
			if item == nil {
				continue
			}
			if c.confirm("удалить запись? (y/N)> ") {
				if err := c.send("DELETE", path+"/"+item.ID, nil, nil, true); err != nil {
					c.fail(err)
				}
			}
		case "0":
			return
		}
	}
}

type simpleDictionaryItem struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

func (c *cli) listSimpleDictionary(path, listField string) []simpleDictionaryItem {
	var resp struct {
		Equipment []simpleDictionaryItem `json:"equipment"`
		Tools     []simpleDictionaryItem `json:"tools"`
	}
	if err := c.send("GET", path, nil, &resp, false); err != nil {
		c.fail(err)
		return nil
	}
	if listField == "equipment" {
		return resp.Equipment
	}
	return resp.Tools
}

func (c *cli) selectSimpleDictionary(path, listField string) *simpleDictionaryItem {
	items := c.listSimpleDictionary(path, listField)
	if len(items) == 0 {
		fmt.Println("(список пуст)")
		return nil
	}
	for {
		for i, item := range items {
			fmt.Printf("%d) %-12s %s\n", i+1, item.Code, item.Name)
		}
		in := c.ask("номер или ID (Enter — отмена)> ")
		if in == "" {
			return nil
		}
		if i, err := strconv.Atoi(in); err == nil {
			if i >= 1 && i <= len(items) {
				return &items[i-1]
			}
		}
		for i := range items {
			if items[i].ID == in {
				return &items[i]
			}
		}
		fmt.Println("нет такой записи")
	}
}

func (c *cli) routeRowsMenu() {
	for {
		fmt.Println()
		fmt.Println("--- Маршрутные строки BOM ---")
		fmt.Println("1. Показать строки")
		fmt.Println("2. Добавить строку")
		fmt.Println("3. Изменить строку")
		fmt.Println("4. Удалить строку")
		fmt.Println("5. Заменить все строки")
		fmt.Println("0. Назад")
		switch c.ask("выбор> ") {
		case "1":
			bomID := c.selectBom()
			c.printRouteRows(bomID)
		case "2":
			bomID := c.selectBom()
			bom, err := c.loadBom(bomID)
			if err != nil {
				c.fail(err)
				continue
			}
			body := c.askRouteRow(bomID, bom.IsDefault, nil)
			if body == nil {
				continue
			}
			var resp struct {
				Row RouteRow `json:"row"`
			}
			if err := c.send("POST", "/v1/boms/"+bomID+"/route/rows", body, &resp, true); err != nil {
				c.fail(err)
			}
		case "3":
			bomID := c.selectBom()
			bom, err := c.loadBom(bomID)
			if err != nil {
				c.fail(err)
				continue
			}
			row := c.selectRouteRow(bomID)
			if row == nil {
				continue
			}
			body := c.askRouteRow(bomID, bom.IsDefault, row)
			if body == nil {
				continue
			}
			var resp struct {
				Row RouteRow `json:"row"`
			}
			if err := c.send("PATCH", "/v1/boms/"+bomID+"/route/rows/"+row.ID, body, &resp, true); err != nil {
				c.fail(err)
			}
		case "4":
			bomID := c.selectBom()
			row := c.selectRouteRow(bomID)
			if row == nil {
				continue
			}
			if c.confirm("удалить строку? (y/N)> ") {
				if err := c.send("DELETE", "/v1/boms/"+bomID+"/route/rows/"+row.ID, nil, nil, true); err != nil {
					c.fail(err)
				}
			}
		case "5":
			bomID := c.selectBom()
			bom, err := c.loadBom(bomID)
			if err != nil {
				c.fail(err)
				continue
			}
			rows := []map[string]any{}
			for {
				row := c.askRouteRow(bomID, bom.IsDefault, nil)
				if row == nil {
					break
				}
				rows = append(rows, row)
				if !c.confirm("добавить ещё строку? (y/N)> ") {
					break
				}
			}
			var resp struct {
				Rows []RouteRow `json:"rows"`
			}
			if err := c.send("PUT", "/v1/boms/"+bomID+"/route/rows", map[string]any{"rows": rows}, &resp, true); err != nil {
				c.fail(err)
			}
		case "0":
			return
		}
	}
}

func (c *cli) listRouteRows(bomID string) []RouteRow {
	var resp struct {
		Rows []RouteRow `json:"rows"`
	}
	if err := c.send("GET", "/v1/boms/"+bomID+"/route/rows", nil, &resp, false); err != nil {
		c.fail(err)
		return nil
	}
	return resp.Rows
}

func (c *cli) printRouteRows(bomID string) {
	rows := c.listRouteRows(bomID)
	if len(rows) == 0 {
		fmt.Println("(строки маршрута отсутствуют)")
		return
	}
	for _, row := range rows {
		inherited := ""
		if row.Inherited {
			inherited = " (унаследована)"
		}
		fmt.Printf("seq=%d  %-20s %-30s qty=%s labor=%s eq=%s tool=%s%s\n",
			row.Seq, row.OperationClassName, c.resolveProductName(row.ProductID),
			row.Quantity, row.LaborNorm, row.EquipmentName, row.ToolName, inherited)
	}
}

func (c *cli) selectRouteRow(bomID string) *RouteRow {
	rows := c.listRouteRows(bomID)
	if len(rows) == 0 {
		fmt.Println("(строки маршрута отсутствуют)")
		return nil
	}
	for {
		for i, row := range rows {
			inherited := ""
			if row.Inherited {
				inherited = " inherited"
			}
			fmt.Printf("%d) seq=%d %-20s %-30s qty=%s%s\n", i+1, row.Seq, row.OperationClassName, c.resolveProductName(row.ProductID), row.Quantity, inherited)
		}
		in := c.ask("номер или ID строки (Enter — отмена)> ")
		if in == "" {
			return nil
		}
		if i, err := strconv.Atoi(in); err == nil {
			if i >= 1 && i <= len(rows) {
				return &rows[i-1]
			}
		}
		for i := range rows {
			if rows[i].ID == in {
				return &rows[i]
			}
		}
		fmt.Println("нет такой строки")
	}
}

func (c *cli) askRouteRow(bomID string, defaultBom bool, current *RouteRow) map[string]any {
	seq := 1
	operationClassID := ""
	productID := ""
	quantity := "1"
	laborNorm := "0"
	equipmentID := ""
	toolID := ""
	excluded := false
	refBomID := ""
	refRevision := 0
	if current != nil {
		seq = current.Seq
		operationClassID = current.OperationClassID
		productID = current.ProductID
		quantity = current.Quantity
		laborNorm = current.LaborNorm
		equipmentID = current.EquipmentID
		toolID = current.ToolID
		excluded = current.IsExcluded
		refBomID = current.ReferenceBomID
		refRevision = current.ReferenceRevision
	}

	if current != nil && current.Inherited {
		fmt.Printf("seq=%d (унаследованная строка, seq не меняется)\n", seq)
	} else {
		seq = c.askInt("seq", seq)
	}
	if operationClassID == "" {
		item := c.selectOperationClass()
		if item == nil {
			return nil
		}
		operationClassID = item.ID
	} else if c.confirm("сменить класс операции? (y/N)> ") {
		item := c.selectOperationClass()
		if item == nil {
			return nil
		}
		operationClassID = item.ID
	}
	if productID == "" {
		productID = c.selectProduct()
	} else if c.confirm("сменить изделие? (y/N)> ") {
		productID = c.selectProduct()
	}
	quantity = c.askDefault("количество", quantity)
	laborNorm = c.askDefault("норма труда (часы)", laborNorm)
	equipmentID = c.askOptionalID("оборудование", equipmentID, func() string {
		item := c.selectSimpleDictionary("/v1/equipment", "equipment")
		if item == nil {
			return ""
		}
		return item.ID
	})
	toolID = c.askOptionalID("инструмент", toolID, func() string {
		item := c.selectSimpleDictionary("/v1/tools", "tools")
		if item == nil {
			return ""
		}
		return item.ID
	})
	if !defaultBom {
		excluded = c.askYes("исключить строку? (y/N)> ", excluded)
	}
	if !excluded {
		refBomID, refRevision, _ = c.askReference(bomID, productID, &BomRow{
			ReferenceBomID:    refBomID,
			ReferenceRevision: refRevision,
		})
	}
	body := map[string]any{
		"seq":               seq,
		"operationClassId":  operationClassID,
		"equipmentId":       equipmentID,
		"toolId":            toolID,
		"productId":         productID,
		"quantity":          quantity,
		"laborNorm":         laborNorm,
		"isExcluded":        excluded,
		"referenceBomId":    refBomID,
		"referenceRevision": refRevision,
	}
	return body
}

func (c *cli) askInt(prompt string, def int) int {
	in := c.askDefault(prompt, strconv.Itoa(def))
	value, err := strconv.Atoi(in)
	if err != nil {
		return def
	}
	return value
}

func (c *cli) askDefault(prompt, def string) string {
	in := c.ask(fmt.Sprintf("%s [%s]> ", prompt, def))
	if in == "" {
		return def
	}
	return in
}

func (c *cli) askOptionalID(title, current string, selectFn func() string) string {
	prompt := fmt.Sprintf("%s (Enter — оставить, - — очистить)> ", title)
	if current == "" {
		prompt = fmt.Sprintf("%s (Enter — без, иначе выбрать)> ", title)
	}
	in := c.ask(prompt)
	if in == "-" {
		return ""
	}
	if in == "" {
		return current
	}
	return selectFn()
}
func (c *cli) bomPriceMenu() {
	bomID := c.selectBom()
	bom, err := c.loadBom(bomID)
	if err != nil {
		c.fail(err)
		return
	}
	fmt.Printf("текущая цена: %s (manual=%v)\n", bom.Price, bom.PriceManual)
	manual := c.askYes("ручная цена? (y/N)> ", bom.PriceManual)
	price := "0"
	if manual {
		price = c.askDefault("цена", bom.Price)
	}
	var resp struct {
		Bom Bom `json:"bom"`
	}
	if err := c.send("PATCH", "/v1/boms/"+bomID+"/price", map[string]any{"price": price, "priceManual": manual}, &resp, true); err != nil {
		c.fail(err)
		return
	}
	c.upsertBoms(resp.Bom)
}
