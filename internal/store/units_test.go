package store

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"

	"github.com/johu/larder/internal/i18n"
	"github.com/johu/larder/internal/model"
)

// The SQLite implementation must satisfy the interface the API depends on.
var _ UnitRepository = (*SQLiteUnits)(nil)

func newUnits(t *testing.T) (*SQLiteUnits, *sql.DB) {
	t.Helper()
	db := openMigrated(t)
	return NewSQLiteUnits(db), db
}

func codes(units []model.Unit) []string {
	out := make([]string, 0, len(units))
	for _, u := range units {
		out = append(out, u.Code)
	}
	return out
}

func find(t *testing.T, units []model.Unit, code string) model.Unit {
	t.Helper()
	for _, u := range units {
		if u.Code == code {
			return u
		}
	}
	t.Fatalf("unit %q not found in %v", code, codes(units))
	return model.Unit{}
}

func strp(s string) *string     { return &s }
func floatp(f float64) *float64 { return &f }

func TestListUnitsOrder(t *testing.T) {
	units, _ := newUnits(t)

	got, err := units.ListUnits(context.Background(), i18n.EN, "")
	if err != nil {
		t.Fatalf("ListUnits() error: %v", err)
	}

	want := []string{"g", "kg", "ml", "l", "tsp", "tbsp", "pinch", "piece", "clove", "slice", "can", "pack"}
	if !slices.Equal(codes(got), want) {
		t.Errorf("codes = %v; want %v (ordered by sort order)", codes(got), want)
	}
}

func TestListUnitsWording(t *testing.T) {
	units, _ := newUnits(t)

	tests := []struct {
		lang   i18n.Lang
		code   string
		name   string
		plural string
		abbr   *string
	}{
		{i18n.EN, "tbsp", "tablespoon", "tablespoons", strp("tbsp")},
		{i18n.DE, "tbsp", "Esslöffel", "Esslöffel", strp("EL")},
		{i18n.PT, "tbsp", "colher de sopa", "colheres de sopa", nil},
		{i18n.EN, "piece", "piece", "pieces", strp("pc")},
		{i18n.DE, "piece", "Stück", "Stück", strp("Stk.")},
		{i18n.PT, "piece", "unidade", "unidades", strp("un.")},
		{i18n.DE, "pinch", "Prise", "Prisen", nil},
		{i18n.PT, "clove", "dente", "dentes", nil},
	}
	for _, tt := range tests {
		t.Run(string(tt.lang)+"/"+tt.code, func(t *testing.T) {
			list, err := units.ListUnits(context.Background(), tt.lang, "")
			if err != nil {
				t.Fatalf("ListUnits() error: %v", err)
			}
			u := find(t, list, tt.code)
			if u.Name != tt.name || u.NamePlural != tt.plural {
				t.Errorf("name, plural = %q, %q; want %q, %q", u.Name, u.NamePlural, tt.name, tt.plural)
			}
			switch {
			case tt.abbr == nil && u.Abbreviation != nil:
				t.Errorf("abbreviation = %q; want nil", *u.Abbreviation)
			case tt.abbr != nil && (u.Abbreviation == nil || *u.Abbreviation != *tt.abbr):
				t.Errorf("abbreviation = %v; want %q", u.Abbreviation, *tt.abbr)
			}
		})
	}
}

func TestListUnitsDimensionAndFactor(t *testing.T) {
	units, _ := newUnits(t)

	list, err := units.ListUnits(context.Background(), i18n.EN, "")
	if err != nil {
		t.Fatalf("ListUnits() error: %v", err)
	}

	tests := []struct {
		code      string
		dimension model.Dimension
		factor    *float64
	}{
		{"g", model.Mass, floatp(1)},
		{"kg", model.Mass, floatp(1000)},
		{"ml", model.Volume, floatp(1)},
		{"l", model.Volume, floatp(1000)},
		{"tsp", model.Volume, floatp(5)},
		{"tbsp", model.Volume, floatp(15)},
		{"pinch", model.Approx, nil},
		{"piece", model.Count, nil},
		{"pack", model.Count, nil},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			u := find(t, list, tt.code)
			if u.Dimension != tt.dimension {
				t.Errorf("dimension = %q; want %q", u.Dimension, tt.dimension)
			}
			switch {
			case tt.factor == nil && u.Factor != nil:
				t.Errorf("factor = %v; want nil", *u.Factor)
			case tt.factor != nil && (u.Factor == nil || *u.Factor != *tt.factor):
				t.Errorf("factor = %v; want %v", u.Factor, *tt.factor)
			}
		})
	}
}

func TestListUnitsFilterByDimension(t *testing.T) {
	units, _ := newUnits(t)

	tests := []struct {
		dimension model.Dimension
		want      []string
	}{
		{model.Mass, []string{"g", "kg"}},
		{model.Volume, []string{"ml", "l", "tsp", "tbsp"}},
		{model.Count, []string{"piece", "clove", "slice", "can", "pack"}},
		{model.Approx, []string{"pinch"}},
	}
	for _, tt := range tests {
		t.Run(string(tt.dimension), func(t *testing.T) {
			got, err := units.ListUnits(context.Background(), i18n.DE, tt.dimension)
			if err != nil {
				t.Fatalf("ListUnits() error: %v", err)
			}
			if !slices.Equal(codes(got), tt.want) {
				t.Errorf("codes = %v; want %v", codes(got), tt.want)
			}
		})
	}
}

func TestListUnitsNoMatchIsAnEmptyNonNilSlice(t *testing.T) {
	units, _ := newUnits(t)

	for _, dimension := range []model.Dimension{"weight", "Mass", "mass' OR '1'='1"} {
		t.Run(string(dimension), func(t *testing.T) {
			got, err := units.ListUnits(context.Background(), i18n.EN, dimension)
			if err != nil {
				t.Fatalf("ListUnits() error: %v", err)
			}
			if got == nil {
				t.Error("result is nil; want an empty slice, so that the API encodes []")
			}
			if len(got) != 0 {
				t.Errorf("got %v; want no units", codes(got))
			}
		})
	}
}

func TestListUnitsFallsBackToEnglishForAnUnsupportedLanguage(t *testing.T) {
	units, _ := newUnits(t)
	ctx := context.Background()

	fr, err := units.ListUnits(ctx, i18n.Lang("fr"), "")
	if err != nil {
		t.Fatalf("ListUnits(fr) error: %v", err)
	}
	en, err := units.ListUnits(ctx, i18n.EN, "")
	if err != nil {
		t.Fatalf("ListUnits(en) error: %v", err)
	}

	if len(fr) != len(en) {
		t.Fatalf("got %d units for fr; want %d, as for en", len(fr), len(en))
	}
	for i := range en {
		if fr[i].Name != en[i].Name || fr[i].NamePlural != en[i].NamePlural {
			t.Errorf("%s: got %q/%q; want the english wording %q/%q", en[i].Code, fr[i].Name, fr[i].NamePlural, en[i].Name, en[i].NamePlural)
		}
	}
}

// TestFallbackIsDecidedPerUnit adds a unit that only has english text and
// checks that it is still listed, in english, while the other units keep
// their german wording.
func TestFallbackIsDecidedPerUnit(t *testing.T) {
	units, db := newUnits(t)

	for _, stmt := range []string{
		`INSERT INTO unit VALUES ('dash', 'approx', NULL, 900)`,
		`INSERT INTO unit_text VALUES ('dash', 'en', 'dash', 'dashes', 'ds')`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}

	list, err := units.ListUnits(context.Background(), i18n.DE, "")
	if err != nil {
		t.Fatalf("ListUnits() error: %v", err)
	}

	dash := find(t, list, "dash")
	if dash.Name != "dash" || dash.NamePlural != "dashes" {
		t.Errorf("dash = %q/%q; want the english wording", dash.Name, dash.NamePlural)
	}
	if dash.Abbreviation == nil || *dash.Abbreviation != "ds" {
		t.Errorf("dash abbreviation = %v; want %q from the english text", dash.Abbreviation, "ds")
	}
	if got := find(t, list, "tbsp").Name; got != "Esslöffel" {
		t.Errorf("tbsp = %q; want the german wording", got)
	}
}

// TestFallbackIsNotDecidedPerColumn guards against COALESCE per column: in
// pt, tbsp has no abbreviation on purpose, and the english "tbsp" must not
// appear instead.
func TestFallbackIsNotDecidedPerColumn(t *testing.T) {
	units, _ := newUnits(t)

	u, err := units.GetUnit(context.Background(), i18n.PT, "tbsp")
	if err != nil {
		t.Fatalf("GetUnit() error: %v", err)
	}
	if u.Abbreviation != nil {
		t.Errorf("abbreviation = %q; want nil (pt has none)", *u.Abbreviation)
	}
}

func TestGetUnit(t *testing.T) {
	units, _ := newUnits(t)

	got, err := units.GetUnit(context.Background(), i18n.DE, "tbsp")
	if err != nil {
		t.Fatalf("GetUnit() error: %v", err)
	}

	if got.Code != "tbsp" || got.Dimension != model.Volume {
		t.Errorf("code, dimension = %q, %q; want tbsp, volume", got.Code, got.Dimension)
	}
	if got.Factor == nil || *got.Factor != 15 {
		t.Errorf("factor = %v; want 15", got.Factor)
	}
	if got.Name != "Esslöffel" || got.NamePlural != "Esslöffel" {
		t.Errorf("name, plural = %q, %q; want Esslöffel, Esslöffel", got.Name, got.NamePlural)
	}
	if got.Abbreviation == nil || *got.Abbreviation != "EL" {
		t.Errorf("abbreviation = %v; want EL", got.Abbreviation)
	}
}

func TestGetUnitWithoutFactor(t *testing.T) {
	units, _ := newUnits(t)

	got, err := units.GetUnit(context.Background(), i18n.EN, "pinch")
	if err != nil {
		t.Fatalf("GetUnit() error: %v", err)
	}
	if got.Factor != nil {
		t.Errorf("factor = %v; want nil", *got.Factor)
	}
	if got.Dimension != model.Approx {
		t.Errorf("dimension = %q; want approx", got.Dimension)
	}
}

func TestGetUnitFallsBackToEnglish(t *testing.T) {
	units, _ := newUnits(t)

	got, err := units.GetUnit(context.Background(), i18n.Lang("fr"), "tbsp")
	if err != nil {
		t.Fatalf("GetUnit() error: %v", err)
	}
	if got.Name != "tablespoon" || got.NamePlural != "tablespoons" {
		t.Errorf("name, plural = %q, %q; want the english wording", got.Name, got.NamePlural)
	}
	if got.Abbreviation == nil || *got.Abbreviation != "tbsp" {
		t.Errorf("abbreviation = %v; want tbsp", got.Abbreviation)
	}
}

func TestGetUnitNotFound(t *testing.T) {
	units, _ := newUnits(t)

	// Codes are case-sensitive.
	for _, code := range []string{"nope", "", "TBSP", "tbsp ", "tbsp' OR '1'='1"} {
		t.Run(code, func(t *testing.T) {
			_, err := units.GetUnit(context.Background(), i18n.EN, code)
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("GetUnit(%q) error = %v; want ErrNotFound", code, err)
			}
		})
	}
}

func TestUnitsStopWithACancelledContext(t *testing.T) {
	units, _ := newUnits(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := units.ListUnits(ctx, i18n.EN, ""); err == nil {
		t.Error("ListUnits() returned no error for a cancelled context")
	}
	_, err := units.GetUnit(ctx, i18n.EN, "g")
	if err == nil {
		t.Error("GetUnit() returned no error for a cancelled context")
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("GetUnit() reported ErrNotFound for a cancelled context; want the context error")
	}
}
