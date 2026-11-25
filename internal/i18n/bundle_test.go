package i18n

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadEmbeddedCatalogs(t *testing.T) {
	b, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if b == nil {
		t.Fatal("Load() returned a nil bundle")
	}
}

// TestCatalogParity makes sure that no language lacks a key that English has
// and that no language has keys English does not know. Empty messages are
// treated as missing.
func TestCatalogParity(t *testing.T) {
	b, err := Load()
	if err != nil || b == nil {
		t.Fatalf("Load() = %v, %v", b, err)
	}
	en := b.catalogs[EN]
	if len(en) == 0 {
		t.Fatal("english catalog is empty")
	}
	for _, lang := range Supported {
		cat, ok := b.catalogs[lang]
		if !ok {
			t.Errorf("%s: catalog missing", lang)
			continue
		}
		for key := range en {
			if cat[key] == "" {
				t.Errorf("%s: missing or empty key %q", lang, key)
			}
		}
		for key := range cat {
			if _, ok := en[key]; !ok {
				t.Errorf("%s: key %q does not exist in en", lang, key)
			}
		}
	}
}

func testFS(en, de, pt string) fstest.MapFS {
	return fstest.MapFS{
		"en.json": {Data: []byte(en)},
		"de.json": {Data: []byte(de)},
		"pt.json": {Data: []byte(pt)},
	}
}

func TestMessage(t *testing.T) {
	b, err := NewBundle(testFS(
		`{"hello": "Hello", "only_en": "Only English"}`,
		`{"hello": "Hallo"}`,
		`{"hello": "Olá"}`,
	))
	if err != nil || b == nil {
		t.Fatalf("NewBundle() = %v, %v", b, err)
	}
	tests := []struct {
		name string
		lang Lang
		key  string
		want string
	}{
		{"english", EN, "hello", "Hello"},
		{"german", DE, "hello", "Hallo"},
		{"portuguese", PT, "hello", "Olá"},
		{"falls back to english", DE, "only_en", "Only English"},
		{"unknown key returns the key", PT, "nope", "nope"},
		{"unsupported language falls back", Lang("fr"), "hello", "Hello"},
		{"empty language falls back", Lang(""), "hello", "Hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := b.Message(tt.lang, tt.key); got != tt.want {
				t.Errorf("Message(%q, %q) = %q; want %q", tt.lang, tt.key, got, tt.want)
			}
		})
	}
}

func TestNewBundleErrors(t *testing.T) {
	tests := []struct {
		name string
		fsys fstest.MapFS
	}{
		{"missing file", fstest.MapFS{
			"en.json": {Data: []byte(`{}`)},
			"de.json": {Data: []byte(`{}`)},
		}},
		{"invalid json", testFS(`{}`, `{not json`, `{}`)},
		{"not an object", testFS(`{}`, `[]`, `{}`)},
		{"null catalog", testFS(`{}`, `null`, `{}`)},
		{"non-string value", testFS(`{}`, `{"a": 1}`, `{}`)},
		{"empty file", testFS(`{}`, ``, `{}`)},
		{"missing english", fstest.MapFS{
			"de.json": {Data: []byte(`{}`)},
			"pt.json": {Data: []byte(`{}`)},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewBundle(tt.fsys); err == nil {
				t.Error("NewBundle() returned no error")
			}
		})
	}
}

func TestMessageEmptyValueFallsBack(t *testing.T) {
	b, err := NewBundle(testFS(
		`{"hello": "Hello"}`,
		`{"hello": ""}`,
		`{"hello": "Olá"}`,
	))
	if err != nil || b == nil {
		t.Fatalf("NewBundle() = %v, %v", b, err)
	}
	if got := b.Message(DE, "hello"); got != "Hello" {
		t.Errorf("Message(DE, hello) = %q; want the english fallback for an empty message", got)
	}
}

func TestNewBundleAcceptsEmptyObjects(t *testing.T) {
	b, err := NewBundle(testFS(`{}`, `{}`, `{}`))
	if err != nil || b == nil {
		t.Fatalf("NewBundle() = %v, %v; want no error for empty catalogs", b, err)
	}
	if got := b.Message(DE, "any"); got != "any" {
		t.Errorf("Message() = %q; want the key for a missing message", got)
	}
}

func TestNewBundleIgnoresUnsupportedFiles(t *testing.T) {
	fsys := testFS(`{"k": "v"}`, `{"k": "w"}`, `{"k": "x"}`)
	fsys["fr.json"] = &fstest.MapFile{Data: []byte(`not json at all`)}
	if _, err := NewBundle(fsys); err != nil {
		t.Errorf("NewBundle() error: %v; files for unsupported languages must be ignored", err)
	}
}

func TestNewBundleErrorNamesTheFile(t *testing.T) {
	_, err := NewBundle(testFS(`{}`, `{not json`, `{}`))
	if err == nil {
		t.Fatal("NewBundle() returned no error")
	}
	if !strings.Contains(err.Error(), "de.json") {
		t.Errorf("error %q does not mention the broken file de.json", err)
	}
}
