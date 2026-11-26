package store

import (
	"context"
	"database/sql"
	"testing"

	"github.com/johu/larder/internal/i18n"
)

// openMigrated returns a database with all embedded migrations applied.
func openMigrated(t *testing.T) *sql.DB {
	t.Helper()
	db := openTest(t)
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}
	return db
}

func TestSeedUnits(t *testing.T) {
	db := openMigrated(t)

	want := []struct {
		code      string
		dimension string
		factor    sql.NullFloat64
	}{
		{"g", "mass", nf(1)},
		{"kg", "mass", nf(1000)},
		{"ml", "volume", nf(1)},
		{"l", "volume", nf(1000)},
		{"tsp", "volume", nf(5)},
		{"tbsp", "volume", nf(15)},
		{"pinch", "approx", sql.NullFloat64{}},
		{"piece", "count", sql.NullFloat64{}},
		{"clove", "count", sql.NullFloat64{}},
		{"slice", "count", sql.NullFloat64{}},
		{"can", "count", sql.NullFloat64{}},
		{"pack", "count", sql.NullFloat64{}},
	}

	rows, err := db.Query("SELECT code, dimension, base_factor FROM unit ORDER BY sort_order")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		var code, dimension string
		var factor sql.NullFloat64
		if err := rows.Scan(&code, &dimension, &factor); err != nil {
			t.Fatal(err)
		}
		if i >= len(want) {
			t.Fatalf("unexpected extra unit %q", code)
		}
		if w := want[i]; code != w.code || dimension != w.dimension || factor != w.factor {
			t.Errorf("unit %d = %s, %s, %v; want %s, %s, %v", i, code, dimension, factor, w.code, w.dimension, w.factor)
		}
		i++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if i != len(want) {
		t.Errorf("got %d units; want %d", i, len(want))
	}
}

func nf(v float64) sql.NullFloat64 { return sql.NullFloat64{Float64: v, Valid: true} }

// TestSeedHasTextForEverySupportedLanguage fails when a language is added to
// i18n.Supported without adding the unit wording for it.
func TestSeedHasTextForEverySupportedLanguage(t *testing.T) {
	db := openMigrated(t)

	for _, lang := range i18n.Supported {
		var missing int
		err := db.QueryRow(`
			SELECT count(*) FROM unit u
			WHERE NOT EXISTS (
				SELECT 1 FROM unit_text t
				WHERE t.code = u.code AND t.lang = ? AND t.name <> '' AND t.name_plural <> ''
			)`, string(lang)).Scan(&missing)
		if err != nil {
			t.Fatal(err)
		}
		if missing != 0 {
			t.Errorf("%d units have no complete text in %q", missing, lang)
		}
	}
}

func TestSeedHasNoTextForUnknownUnitsOrExtraLanguages(t *testing.T) {
	db := openMigrated(t)

	var orphans int
	if err := db.QueryRow(`SELECT count(*) FROM unit_text WHERE code NOT IN (SELECT code FROM unit)`).Scan(&orphans); err != nil {
		t.Fatal(err)
	}
	if orphans != 0 {
		t.Errorf("%d text rows belong to no unit", orphans)
	}

	var units, texts int
	if err := db.QueryRow(`SELECT (SELECT count(*) FROM unit), (SELECT count(*) FROM unit_text)`).Scan(&units, &texts); err != nil {
		t.Fatal(err)
	}
	if want := units * len(i18n.Supported); texts != want {
		t.Errorf("unit_text has %d rows; want %d (units x supported languages)", texts, want)
	}
}

// TestSeedWording checks the expected wording.
func TestSeedWording(t *testing.T) {
	db := openMigrated(t)

	// abbr "" means NULL.
	tests := []struct {
		code, lang, name, plural, abbr string
	}{
		{"g", "en", "gram", "grams", "g"},
		{"g", "de", "Gramm", "Gramm", "g"},
		{"g", "pt", "grama", "gramas", "g"},
		{"kg", "en", "kilogram", "kilograms", "kg"},
		{"kg", "de", "Kilogramm", "Kilogramm", "kg"},
		{"kg", "pt", "quilograma", "quilogramas", "kg"},
		{"ml", "en", "milliliter", "milliliters", "ml"},
		{"ml", "de", "Milliliter", "Milliliter", "ml"},
		{"ml", "pt", "mililitro", "mililitros", "ml"},
		{"l", "en", "liter", "liters", "l"},
		{"l", "de", "Liter", "Liter", "l"},
		{"l", "pt", "litro", "litros", "l"},
		{"tsp", "en", "teaspoon", "teaspoons", "tsp"},
		{"tsp", "de", "Teelöffel", "Teelöffel", "TL"},
		{"tsp", "pt", "colher de chá", "colheres de chá", ""},
		{"tbsp", "en", "tablespoon", "tablespoons", "tbsp"},
		{"tbsp", "de", "Esslöffel", "Esslöffel", "EL"},
		{"tbsp", "pt", "colher de sopa", "colheres de sopa", ""},
		{"pinch", "en", "pinch", "pinches", ""},
		{"pinch", "de", "Prise", "Prisen", ""},
		{"pinch", "pt", "pitada", "pitadas", ""},
		{"piece", "en", "piece", "pieces", "pc"},
		{"piece", "de", "Stück", "Stück", "Stk."},
		{"piece", "pt", "unidade", "unidades", "un."},
		{"clove", "en", "clove", "cloves", ""},
		{"clove", "de", "Zehe", "Zehen", ""},
		{"clove", "pt", "dente", "dentes", ""},
		{"slice", "en", "slice", "slices", ""},
		{"slice", "de", "Scheibe", "Scheiben", ""},
		{"slice", "pt", "fatia", "fatias", ""},
		{"can", "en", "can", "cans", ""},
		{"can", "de", "Dose", "Dosen", ""},
		{"can", "pt", "lata", "latas", ""},
		{"pack", "en", "pack", "packs", ""},
		{"pack", "de", "Packung", "Packungen", ""},
		{"pack", "pt", "embalagem", "embalagens", ""},
	}

	for _, tt := range tests {
		t.Run(tt.code+"/"+tt.lang, func(t *testing.T) {
			var name, plural string
			var abbr sql.NullString
			err := db.QueryRow(
				"SELECT name, name_plural, abbreviation FROM unit_text WHERE code = ? AND lang = ?",
				tt.code, tt.lang).Scan(&name, &plural, &abbr)
			if err != nil {
				t.Fatalf("query: %v", err)
			}
			if name != tt.name || plural != tt.plural {
				t.Errorf("name, plural = %q, %q; want %q, %q", name, plural, tt.name, tt.plural)
			}
			if tt.abbr == "" && abbr.Valid {
				t.Errorf("abbreviation = %q; want NULL", abbr.String)
			}
			if tt.abbr != "" && abbr.String != tt.abbr {
				t.Errorf("abbreviation = %v; want %q", abbr, tt.abbr)
			}
		})
	}
}

func TestSchemaRejectsInvalidData(t *testing.T) {
	db := openMigrated(t)

	tests := []struct {
		name string
		stmt string
	}{
		{"unknown dimension", `INSERT INTO unit VALUES ('x1', 'weight', 1, 900)`},
		{"mass without factor", `INSERT INTO unit VALUES ('x2', 'mass', NULL, 900)`},
		{"volume without factor", `INSERT INTO unit VALUES ('x3', 'volume', NULL, 900)`},
		{"count with factor", `INSERT INTO unit VALUES ('x4', 'count', 2, 900)`},
		{"approx with factor", `INSERT INTO unit VALUES ('x5', 'approx', 2, 900)`},
		{"zero factor", `INSERT INTO unit VALUES ('x6', 'mass', 0, 900)`},
		{"negative factor", `INSERT INTO unit VALUES ('x7', 'mass', -1, 900)`},
		{"null code", `INSERT INTO unit VALUES (NULL, 'count', NULL, 900)`},
		{"duplicate code", `INSERT INTO unit VALUES ('g', 'mass', 1, 900)`},
		{"text for an unknown unit", `INSERT INTO unit_text VALUES ('nope', 'en', 'a', 'b', NULL)`},
		{"duplicate text row", `INSERT INTO unit_text VALUES ('g', 'en', 'a', 'b', NULL)`},
		{"empty name", `INSERT INTO unit_text VALUES ('g', 'fr', '', 'b', NULL)`},
		{"empty plural", `INSERT INTO unit_text VALUES ('g', 'fr', 'a', '', NULL)`},
		{"empty abbreviation", `INSERT INTO unit_text VALUES ('g', 'fr', 'a', 'b', '')`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := db.Exec(tt.stmt); err == nil {
				t.Errorf("%s was accepted", tt.stmt)
			}
		})
	}
}

// TestSchemaAcceptsAnyLanguageCode documents that the supported languages
// are not restricted in the database.
func TestSchemaAcceptsAnyLanguageCode(t *testing.T) {
	db := openMigrated(t)

	if _, err := db.Exec(`INSERT INTO unit_text VALUES ('g', 'fr', 'gramme', 'grammes', 'g')`); err != nil {
		t.Errorf("a text row for a new language was rejected: %v", err)
	}
}

func TestDeletingAUnitWithTextIsRejected(t *testing.T) {
	db := openMigrated(t)

	if _, err := db.Exec(`DELETE FROM unit WHERE code = 'g'`); err == nil {
		t.Error("a unit that still has text rows was deleted; foreign keys are not enforced")
	}
}
