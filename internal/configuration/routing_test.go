package configuration

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestResolveRouteRow(t *testing.T) {
	svc := &Service{}
	owner := "00000000-0000-0000-0000-000000000001"
	bomID := "00000000-0000-0000-0000-000000000002"

	t.Run("valid default row", func(t *testing.T) {
		in, err := svc.resolveRouteRow(bomID, owner, true, RouteRowInput{
			Seq:              1,
			OperationClassID: "op",
			ProductID:        "00000000-0000-0000-0000-000000000003",
			Quantity:         "2",
			LaborNorm:        "0.5",
		})
		if err != nil {
			t.Fatal(err)
		}
		if in.Seq != 1 || in.Quantity != "2" {
			t.Fatalf("unexpected row: %+v", in)
		}
	})

	t.Run("self product", func(t *testing.T) {
		_, err := svc.resolveRouteRow(bomID, owner, false, RouteRowInput{
			Seq:              1,
			OperationClassID: "op",
			ProductID:        owner,
			Quantity:         "1",
		})
		if !errors.Is(err, ErrSelfReference) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("reference pair", func(t *testing.T) {
		_, err := svc.resolveRouteRow(bomID, owner, false, RouteRowInput{
			Seq:              1,
			OperationClassID: "op",
			ProductID:        "00000000-0000-0000-0000-000000000003",
			Quantity:         "1",
			ReferenceBomID:   "00000000-0000-0000-0000-000000000004",
		})
		if !errors.Is(err, ErrReferencePair) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("default cannot exclude", func(t *testing.T) {
		_, err := svc.resolveRouteRow(bomID, owner, true, RouteRowInput{
			Seq:              1,
			OperationClassID: "op",
			ProductID:        "00000000-0000-0000-0000-000000000003",
			IsExcluded:       true,
		})
		if !errors.Is(err, ErrExcludedOnlyModification) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("excluded normalizes", func(t *testing.T) {
		in, err := svc.resolveRouteRow(bomID, owner, false, RouteRowInput{
			Seq:               1,
			OperationClassID:  "op",
			ProductID:         "00000000-0000-0000-0000-000000000003",
			Quantity:          "10",
			LaborNorm:         "5",
			EquipmentID:       "eq",
			ToolID:            "tool",
			IsExcluded:        true,
			ReferenceBomID:    "00000000-0000-0000-0000-000000000004",
			ReferenceRevision: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		if in.Quantity != "0" || in.LaborNorm != "0" || in.EquipmentID != "" || in.ToolID != "" || in.ReferenceBomID != "" || in.ReferenceRevision != 0 {
			t.Fatalf("not normalized: %+v", in)
		}
	})

	t.Run("invalid quantity", func(t *testing.T) {
		_, err := svc.resolveRouteRow(bomID, owner, false, RouteRowInput{
			Seq:              1,
			OperationClassID: "op",
			ProductID:        "00000000-0000-0000-0000-000000000003",
			Quantity:         "0",
		})
		if !errors.Is(err, ErrInvalidQuantity) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("invalid labor norm", func(t *testing.T) {
		_, err := svc.resolveRouteRow(bomID, owner, false, RouteRowInput{
			Seq:              1,
			OperationClassID: "op",
			ProductID:        "00000000-0000-0000-0000-000000000003",
			Quantity:         "1",
			LaborNorm:        "-1",
		})
		if !errors.Is(err, ErrInvalidRouteRow) {
			t.Fatalf("got %v", err)
		}
	})
}

func TestFormatMoney(t *testing.T) {
	cases := map[float64]string{
		0:      "0.00",
		1.234:  "1.23",
		1.235:  "1.24",
		10.999: "11.00",
	}
	for in, want := range cases {
		if got := formatMoney(in); got != want {
			t.Fatalf("formatMoney(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestParseQuantity(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want float64
		ok   bool
	}{
		{name: "integer", in: "10", want: 10, ok: true},
		{name: "decimal", in: "2.5", want: 2.5, ok: true},
		{name: "spaces", in: " 3 ", want: 3, ok: true},
		{name: "empty", in: "", ok: false},
		{name: "text", in: "x", ok: false},
		{name: "nan", in: "NaN", ok: false},
		{name: "inf", in: "Inf", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseQuantity(tc.in)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("parseQuantity(%q) = (%v, %v), want (%v, %v)", tc.in, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestNormalizeRouteInput(t *testing.T) {
	t.Run("keeps values", func(t *testing.T) {
		qty, labor, equipment, tool, refBom, refRev := normalizeRouteInput(RouteRowInput{
			Quantity:          "2",
			LaborNorm:         "0.5",
			EquipmentID:       "eq",
			ToolID:            "tool",
			ReferenceBomID:    "bom",
			ReferenceRevision: 2,
		})
		if qty != "2" || labor != "0.5" {
			t.Fatalf("unexpected quantities: %q %q", qty, labor)
		}
		if equipment == nil || *equipment != "eq" || tool == nil || *tool != "tool" {
			t.Fatalf("unexpected ids: %v %v", equipment, tool)
		}
		if refBom == nil || *refBom != "bom" || refRev == nil || *refRev != 2 {
			t.Fatalf("unexpected reference: %v %v", refBom, refRev)
		}
	})

	t.Run("excluded clears values", func(t *testing.T) {
		qty, labor, equipment, tool, refBom, refRev := normalizeRouteInput(RouteRowInput{
			Quantity:          "2",
			LaborNorm:         "0.5",
			EquipmentID:       "eq",
			ToolID:            "tool",
			ReferenceBomID:    "bom",
			ReferenceRevision: 2,
			IsExcluded:        true,
		})
		if qty != "0" || labor != "0" {
			t.Fatalf("unexpected quantities: %q %q", qty, labor)
		}
		if equipment != nil || tool != nil || refBom != nil || refRev != nil {
			t.Fatalf("expected nil references, got %v %v %v %v", equipment, tool, refBom, refRev)
		}
	})
}

func TestDecodeRouteRowSnapshots(t *testing.T) {
	t.Run("decodes and skips excluded", func(t *testing.T) {
		raw := []byte(`[
			{"id":"r1","bom_id":"source","seq":10,"operation_class_id":"op","operation_class_name":"Cut","product_id":"p1","quantity":"2.000","labor_norm":"0.500","inherited":true},
			{"id":"r2","seq":20,"operation_class_id":"op","product_id":"p2","quantity":"1.000","labor_norm":"0","is_excluded":true}
		]`)
		rows, err := decodeRouteRowSnapshots("current", raw)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 {
			t.Fatalf("got %d rows, want 1", len(rows))
		}
		row := rows[0]
		if row.ID != "r1" || row.BomID != "current" || row.SourceBomID != "source" {
			t.Fatalf("unexpected row: %+v", row)
		}
		if row.Seq != 10 || row.OperationClassName != "Cut" || row.Quantity != "2.000" || !row.Inherited {
			t.Fatalf("unexpected row fields: %+v", row)
		}
	})

	t.Run("falls back to current bom", func(t *testing.T) {
		raw := []byte(`[{"id":"r1","seq":10,"operation_class_id":"op","product_id":"p1","quantity":"1","labor_norm":"0"}]`)
		rows, err := decodeRouteRowSnapshots("current", raw)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0].SourceBomID != "current" {
			t.Fatalf("unexpected rows: %+v", rows)
		}
	})

	t.Run("empty snapshot", func(t *testing.T) {
		rows, err := decodeRouteRowSnapshots("current", []byte(`[]`))
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 0 {
			t.Fatalf("got %d rows, want 0", len(rows))
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		if _, err := decodeRouteRowSnapshots("current", []byte(`{`)); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestMapDictionaryError(t *testing.T) {
	cases := []struct {
		name string
		code string
		want error
	}{
		{name: "duplicate code", code: "23505", want: ErrDictionaryCodeExists},
		{name: "foreign key", code: "23503", want: ErrDictionaryInUse},
		{name: "restrict", code: "23001", want: ErrDictionaryInUse},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := mapDictionaryError(&pgconn.PgError{Code: tc.code})
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}

	t.Run("unknown pg error", func(t *testing.T) {
		err := mapDictionaryError(&pgconn.PgError{Code: "XX000"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("non pg error", func(t *testing.T) {
		want := errors.New("boom")
		if got := mapDictionaryError(want); got != want {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}

func TestMapPgError(t *testing.T) {
	cases := []struct {
		name       string
		code       string
		constraint string
		want       error
	}{
		{name: "route seq unique", code: "23505", constraint: "bom_route_rows_seq_unique", want: ErrRouteSeqExists},
		{name: "reference not found", code: "23503", constraint: "bom_route_rows_reference_fk", want: ErrReferenceNotFound},
		{name: "bom not found", code: "23503", constraint: "bom_rows_bom_id_fkey", want: ErrNotFound},
		{name: "dictionary not found", code: "23503", constraint: "bom_route_rows_equipment_id_fkey", want: ErrNotFound},
		{name: "price check", code: "23514", constraint: "boms_price_check", want: ErrInvalidPrice},
		{name: "route check", code: "23514", constraint: "bom_route_rows_quantity_check", want: ErrInvalidRouteRow},
		{name: "reference restrict", code: "23001", constraint: "bom_route_rows_reference_fk", want: ErrReferenceNotFound},
		{name: "restrict fallback", code: "23001", constraint: "other", want: ErrInvalidRow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := mapPgError(&pgconn.PgError{Code: tc.code, ConstraintName: tc.constraint})
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}

	t.Run("non pg error", func(t *testing.T) {
		want := errors.New("boom")
		if got := mapPgError(want); got != want {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}
