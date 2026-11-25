package i18n

import "testing"

func TestMatch(t *testing.T) {
	tests := []struct {
		tag  string
		want Lang
		ok   bool
	}{
		{"en", EN, true},
		{"de", DE, true},
		{"pt", PT, true},
		{"EN", EN, true},
		{"De", DE, true},
		{"en-US", EN, true},
		{"de-DE", DE, true},
		{"de-AT", DE, true},
		{"pt-PT", PT, true},
		{"pt-BR", PT, true},
		{"fr", "", false},
		{"fr-FR", "", false},
		{"es", "", false},
		{"*", "", false},
		{"", "", false},
		{"  ", "", false},
		{"e", "", false},
		{"english", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got, ok := Match(tt.tag)
			if got != tt.want || ok != tt.ok {
				t.Errorf("Match(%q) = %q, %v; want %q, %v", tt.tag, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestFromAcceptLanguage(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   Lang
		ok     bool
	}{
		{"single", "de", DE, true},
		{"region", "pt-BR", PT, true},
		{"empty", "", "", false},
		{"only unsupported", "fr, es;q=0.8", "", false},
		{"wildcard only", "*", "", false},
		{"highest q wins", "en;q=0.5, pt-BR;q=0.9", PT, true},
		{"implicit q is 1", "de-DE,de;q=0.9,en;q=0.8", DE, true},
		{"unsupported skipped", "fr, de;q=0.5", DE, true},
		{"first wins on tie", "de, pt", DE, true},
		{"first wins on tie reversed", "pt, de", PT, true},
		{"q zero is not acceptable", "pt;q=0", "", false},
		{"q zero skipped, next used", "pt;q=0, en;q=0.1", EN, true},
		{"spaces", " de ; q=0.8 , en ", EN, true},
		{"uppercase tag", "DE", DE, true},
		{"malformed q skipped", "de;q=abc, en;q=0.5", EN, true},
		{"q above 1 skipped", "de;q=2, en;q=0.5", EN, true},
		{"empty entries", ",, de ,,", DE, true},
		{"wildcard ignored", "*;q=1, pt;q=0.5", PT, true},
		{"nan weight skipped", "de;q=NaN, pt;q=0.5", PT, true},
		{"nan weight does not poison later entries", "pt;q=0.5, de;q=NaN, en;q=0.1", PT, true},
		{"nan weight only", "de;q=NaN", "", false},
		{"infinite weight skipped", "de;q=Inf, en;q=0.5", EN, true},
		{"negative weight skipped", "de;q=-1, en;q=0.5", EN, true},
		{"extra parameter skipped", "de;q=0.9;x=1, en;q=0.1", EN, true},
		{"unknown parameter skipped", "de;level=1, en;q=0.1", EN, true},
		{"underscore tag is not bcp47", "en_US", "", false},
		{"uppercase q parameter", "de;Q=0.9, en;q=0.1", DE, true},
		{"unsupported entry with higher q does not win", "fr;q=1, de;q=0.1", DE, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := FromAcceptLanguage(tt.header)
			if got != tt.want || ok != tt.ok {
				t.Errorf("FromAcceptLanguage(%q) = %q, %v; want %q, %v", tt.header, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		header string
		def    Lang
		want   Lang
	}{
		{"nothing given uses default", "", "", EN, EN},
		{"default is respected", "", "", DE, DE},
		{"header beats default", "", "pt", EN, PT},
		{"query beats header", "de", "pt", EN, DE},
		{"query beats default", "pt", "", EN, PT},
		{"query with region", "pt-BR", "", EN, PT},
		{"unsupported query falls through to header", "fr", "de", EN, DE},
		{"unsupported query and header use default", "fr", "es", PT, PT},
		{"unsupported header uses default", "", "fr", DE, DE},
		{"wildcard query is ignored", "*", "pt", EN, PT},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Resolve(tt.query, tt.header, tt.def); got != tt.want {
				t.Errorf("Resolve(%q, %q, %q) = %q; want %q", tt.query, tt.header, tt.def, got, tt.want)
			}
		})
	}
}

// TestMatchUsesSupported makes sure that Match derives the supported set from
// Supported, so that adding a language needs no other code change.
//
// It temporarily modifies the package-level Supported and restores it
// afterwards. Do not call t.Parallel() in this test, or in any other test of
// this package: a parallel test could see the extra language.
func TestMatchUsesSupported(t *testing.T) {
	const fr Lang = "fr"

	old := Supported
	Supported = append(append([]Lang(nil), old...), fr)
	t.Cleanup(func() { Supported = old })

	got, ok := Match("fr-CA")
	if got != fr || !ok {
		t.Errorf("Match(%q) = %q, %v; want %q, true after adding it to Supported", "fr-CA", got, ok, fr)
	}
	if got, ok := FromAcceptLanguage("fr;q=0.9, de;q=0.5"); got != fr || !ok {
		t.Errorf("FromAcceptLanguage() = %q, %v; want %q, true", got, ok, fr)
	}
}

// TestResolveReturnsDefaultVerbatim checks that the default is returned
// unchanged when neither the query nor the header can be matched.
func TestResolveReturnsDefaultVerbatim(t *testing.T) {
	for _, def := range Supported {
		if got := Resolve("xx", "yy", def); got != def {
			t.Errorf("Resolve(unsupported, unsupported, %q) = %q; want the default", def, got)
		}
	}
}
