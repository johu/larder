package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/johu/larder/internal/api"
	"github.com/johu/larder/internal/i18n"
	"github.com/johu/larder/internal/model"
	"github.com/johu/larder/internal/store"
)

// fakeUnits is a UnitRepository that records how it was called.
type fakeUnits struct {
	units   []model.Unit
	listErr error
	getErr  error

	listCalls, getCalls int
	gotLang             i18n.Lang
	gotDimension        model.Dimension
	gotCode             string
}

func (f *fakeUnits) ListUnits(_ context.Context, lang i18n.Lang, dimension model.Dimension) ([]model.Unit, error) {
	f.listCalls++
	f.gotLang, f.gotDimension = lang, dimension
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.units, nil
}

func (f *fakeUnits) GetUnit(_ context.Context, lang i18n.Lang, code string) (model.Unit, error) {
	f.getCalls++
	f.gotLang, f.gotCode = lang, code
	if f.getErr != nil {
		return model.Unit{}, f.getErr
	}
	for _, u := range f.units {
		if u.Code == code {
			return u, nil
		}
	}
	// Wrapped on purpose: the handler must use errors.Is.
	return model.Unit{}, fmt.Errorf("get %q: %w", code, store.ErrNotFound)
}

func fptr(f float64) *float64 { return &f }
func sptr(s string) *string   { return &s }

func fixtureUnits() []model.Unit {
	return []model.Unit{
		{Code: "g", Dimension: model.Mass, Factor: fptr(1), Name: "gram", NamePlural: "grams", Abbreviation: sptr("g")},
		{Code: "tbsp", Dimension: model.Volume, Factor: fptr(15), Name: "tablespoon", NamePlural: "tablespoons", Abbreviation: sptr("tbsp")},
		{Code: "piece", Dimension: model.Count, Name: "piece", NamePlural: "pieces", Abbreviation: sptr("pc")},
		{Code: "pinch", Dimension: model.Approx, Name: "pinch", NamePlural: "pinches"},
	}
}

func loadBundle(t *testing.T) *i18n.Bundle {
	t.Helper()
	b, err := i18n.Load()
	if err != nil {
		t.Fatalf("i18n.Load() error: %v", err)
	}
	return b
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func newHandler(t *testing.T, repo *fakeUnits) http.Handler {
	t.Helper()
	return api.NewHandler(api.Options{
		Units:       repo,
		Bundle:      loadBundle(t),
		DefaultLang: i18n.EN,
		Logger:      quiet(),
	})
}

func do(h http.Handler, method, target string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func wantJSON(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q; want application/json", ct)
	}
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// wantError checks status, JSON, the error code and that the message is the
// catalog message for lang.
func wantError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string, lang i18n.Lang) {
	t.Helper()
	if rec.Code != status {
		t.Errorf("status = %d; want %d (body %s)", rec.Code, status, rec.Body)
	}
	wantJSON(t, rec)
	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %q is not the error format: %v", rec.Body, err)
	}
	if body.Error.Code != code {
		t.Errorf("error code = %q; want %q", body.Error.Code, code)
	}
	want := loadBundle(t).Message(lang, "error."+code)
	if body.Error.Message != want || want == "error."+code {
		t.Errorf("error message = %q; want the %s catalog message %q", body.Error.Message, lang, want)
	}
}

func TestHealthz(t *testing.T) {
	repo := &fakeUnits{units: fixtureUnits()}
	rec := do(newHandler(t, repo), http.MethodGet, "/healthz")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d; want 200", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q; want empty", rec.Body)
	}
	if repo.listCalls+repo.getCalls != 0 {
		t.Error("/healthz called the repository; it must be a pure liveness check")
	}
}

func TestListUnits(t *testing.T) {
	repo := &fakeUnits{units: fixtureUnits()}
	rec := do(newHandler(t, repo), http.MethodGet, "/units")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200 (body %s)", rec.Code, rec.Body)
	}
	wantJSON(t, rec)

	var body struct {
		Units []map[string]json.RawMessage `json:"units"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON %q: %v", rec.Body, err)
	}

	var codes []string
	for _, u := range body.Units {
		codes = append(codes, strings.Trim(string(u["code"]), `"`))
	}
	if want := []string{"g", "tbsp", "piece", "pinch"}; strings.Join(codes, ",") != strings.Join(want, ",") {
		t.Errorf("codes = %v; want %v, in the order of the repository", codes, want)
	}
}

func TestUnitJSONShape(t *testing.T) {
	repo := &fakeUnits{units: fixtureUnits()}
	h := newHandler(t, repo)

	tests := []struct {
		code string
		want map[string]string // raw JSON values
	}{
		{"tbsp", map[string]string{
			"code": `"tbsp"`, "dimension": `"volume"`, "base_unit": `"ml"`, "factor": `15`,
			"name": `"tablespoon"`, "name_plural": `"tablespoons"`, "abbreviation": `"tbsp"`,
		}},
		{"g", map[string]string{
			"code": `"g"`, "dimension": `"mass"`, "base_unit": `"g"`, "factor": `1`,
			"name": `"gram"`, "name_plural": `"grams"`, "abbreviation": `"g"`,
		}},
		{"piece", map[string]string{
			"code": `"piece"`, "dimension": `"count"`, "base_unit": `null`, "factor": `null`,
			"name": `"piece"`, "name_plural": `"pieces"`, "abbreviation": `"pc"`,
		}},
		{"pinch", map[string]string{
			"code": `"pinch"`, "dimension": `"approx"`, "base_unit": `null`, "factor": `null`,
			"name": `"pinch"`, "name_plural": `"pinches"`, "abbreviation": `null`,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			rec := do(h, http.MethodGet, "/units/"+tt.code)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d; want 200 (body %s)", rec.Code, rec.Body)
			}
			wantJSON(t, rec)

			var got map[string]json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("invalid JSON %q: %v", rec.Body, err)
			}
			if len(got) != len(tt.want) {
				t.Errorf("unit has %d keys; want exactly %d: %s", len(got), len(tt.want), rec.Body)
			}
			for key, want := range tt.want {
				raw, ok := got[key]
				if !ok {
					t.Errorf("key %q is missing; every key must be present, with null if empty", key)
					continue
				}
				if string(raw) != want {
					t.Errorf("%s = %s; want %s", key, raw, want)
				}
			}
		})
	}
}

func TestListUnitsIsAnEmptyArrayNotNull(t *testing.T) {
	for name, units := range map[string][]model.Unit{"empty": {}, "nil": nil} {
		t.Run(name, func(t *testing.T) {
			rec := do(newHandler(t, &fakeUnits{units: units}), http.MethodGet, "/units")
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d; want 200", rec.Code)
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("invalid JSON %q: %v", rec.Body, err)
			}
			if got := string(body["units"]); got != "[]" {
				t.Errorf(`"units" = %s; want []`, got)
			}
		})
	}
}

func TestListUnitsDimensionFilter(t *testing.T) {
	tests := []struct {
		query string
		want  model.Dimension
	}{
		{"/units", ""},
		{"/units?dimension=", ""},
		{"/units?dimension=mass", model.Mass},
		{"/units?dimension=volume", model.Volume},
		{"/units?dimension=count", model.Count},
		{"/units?dimension=approx", model.Approx},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			repo := &fakeUnits{units: fixtureUnits()}
			rec := do(newHandler(t, repo), http.MethodGet, tt.query)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d; want 200 (body %s)", rec.Code, rec.Body)
			}
			if repo.listCalls != 1 || repo.gotDimension != tt.want {
				t.Errorf("repository got dimension %q in %d calls; want %q in 1", repo.gotDimension, repo.listCalls, tt.want)
			}
		})
	}
}

func TestListUnitsInvalidDimension(t *testing.T) {
	for _, value := range []string{"weight", "Mass", "MASS", "mass%20", "mass,volume", "1"} {
		t.Run(value, func(t *testing.T) {
			repo := &fakeUnits{units: fixtureUnits()}
			rec := do(newHandler(t, repo), http.MethodGet, "/units?dimension="+value)

			wantError(t, rec, http.StatusBadRequest, "invalid_dimension", i18n.EN)
			if repo.listCalls != 0 {
				t.Error("the repository was called for an invalid dimension")
			}
		})
	}
}

func TestLanguageSelection(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		headers []string
		want    i18n.Lang
	}{
		{"default", "/units", nil, i18n.EN},
		{"accept-language", "/units", []string{"Accept-Language", "de"}, i18n.DE},
		{"accept-language with region and q", "/units", []string{"Accept-Language", "fr, pt-BR;q=0.8, en;q=0.1"}, i18n.PT},
		{"query beats header", "/units?lang=pt", []string{"Accept-Language", "de"}, i18n.PT},
		{"query with region", "/units?lang=de-AT", nil, i18n.DE},
		{"unsupported query falls through to header", "/units?lang=fr", []string{"Accept-Language", "de"}, i18n.DE},
		{"unsupported everywhere uses the default", "/units?lang=fr", []string{"Accept-Language", "es"}, i18n.EN},
		{"malformed header uses the default", "/units", []string{"Accept-Language", ";;q=,,"}, i18n.EN},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUnits{units: fixtureUnits()}
			rec := do(newHandler(t, repo), http.MethodGet, tt.target, tt.headers...)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d; want 200", rec.Code)
			}
			if repo.gotLang != tt.want {
				t.Errorf("repository got language %q; want %q", repo.gotLang, tt.want)
			}
			if got := rec.Header().Get("Content-Language"); got != string(tt.want) {
				t.Errorf("Content-Language = %q; want %q", got, tt.want)
			}
			if vary := rec.Header().Values("Vary"); !strings.Contains(strings.Join(vary, ","), "Accept-Language") {
				t.Errorf("Vary = %v; want it to contain Accept-Language", vary)
			}
		})
	}
}

func TestDefaultLanguageIsConfigurable(t *testing.T) {
	for _, tt := range []struct {
		name string
		def  i18n.Lang
		want i18n.Lang
	}{
		{"german", i18n.DE, i18n.DE},
		{"portuguese", i18n.PT, i18n.PT},
		{"empty means the fallback language", "", i18n.Fallback},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUnits{units: fixtureUnits()}
			h := api.NewHandler(api.Options{Units: repo, Bundle: loadBundle(t), DefaultLang: tt.def, Logger: quiet()})
			rec := do(h, http.MethodGet, "/units")

			if repo.gotLang != tt.want {
				t.Errorf("repository got language %q; want %q", repo.gotLang, tt.want)
			}
			if got := rec.Header().Get("Content-Language"); got != string(tt.want) {
				t.Errorf("Content-Language = %q; want %q", got, tt.want)
			}
		})
	}
}

func TestLanguageHeadersOnEveryResponse(t *testing.T) {
	repo := &fakeUnits{units: fixtureUnits(), getErr: errors.New("boom")}
	h := newHandler(t, repo)

	for _, tt := range []struct{ method, target string }{
		{http.MethodGet, "/healthz"},
		{http.MethodGet, "/units/tbsp"}, // 500
		{http.MethodGet, "/units?dimension=x"},
		{http.MethodGet, "/nope"},
		{http.MethodPost, "/units"},
	} {
		t.Run(tt.method+" "+tt.target, func(t *testing.T) {
			rec := do(h, tt.method, tt.target, "Accept-Language", "de")
			if got := rec.Header().Get("Content-Language"); got != "de" {
				t.Errorf("Content-Language = %q; want de (status %d)", got, rec.Code)
			}
			if vary := strings.Join(rec.Header().Values("Vary"), ","); !strings.Contains(vary, "Accept-Language") {
				t.Errorf("Vary = %q; want it to contain Accept-Language", vary)
			}
		})
	}
}

func TestGetUnit(t *testing.T) {
	repo := &fakeUnits{units: fixtureUnits()}
	rec := do(newHandler(t, repo), http.MethodGet, "/units/tbsp?lang=pt")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200 (body %s)", rec.Code, rec.Body)
	}
	if repo.gotCode != "tbsp" || repo.gotLang != i18n.PT {
		t.Errorf("repository got %q in %q; want tbsp in pt", repo.gotCode, repo.gotLang)
	}
}

func TestGetUnitNotFound(t *testing.T) {
	for _, lang := range i18n.Supported {
		t.Run(string(lang), func(t *testing.T) {
			rec := do(newHandler(t, &fakeUnits{units: fixtureUnits()}), http.MethodGet, "/units/nope?lang="+string(lang))
			wantError(t, rec, http.StatusNotFound, "unit_not_found", lang)
		})
	}
}

func TestErrorMessagesAreLocalized(t *testing.T) {
	h := newHandler(t, &fakeUnits{})
	messages := map[string]bool{}
	for _, lang := range i18n.Supported {
		rec := do(h, http.MethodGet, "/units/nope?lang="+string(lang))
		var body errorBody
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		messages[body.Error.Message] = true
	}
	if len(messages) != len(i18n.Supported) {
		t.Errorf("got %d distinct messages for %d languages: %v", len(messages), len(i18n.Supported), messages)
	}
}

func TestUnknownPathIsNotFound(t *testing.T) {
	h := newHandler(t, &fakeUnits{units: fixtureUnits()})

	for _, target := range []string{"/nope", "/", "/units/", "/units/tbsp/extra", "/healthz/x"} {
		t.Run(target, func(t *testing.T) {
			wantError(t, do(h, http.MethodGet, target), http.StatusNotFound, "not_found", i18n.EN)
		})
	}
	t.Run("localized", func(t *testing.T) {
		wantError(t, do(h, http.MethodGet, "/nope", "Accept-Language", "pt"), http.StatusNotFound, "not_found", i18n.PT)
	})
}

func TestWrongMethodIsNotAllowed(t *testing.T) {
	h := newHandler(t, &fakeUnits{units: fixtureUnits()})

	for _, tt := range []struct{ method, target string }{
		{http.MethodPost, "/units"},
		{http.MethodPut, "/units"},
		{http.MethodDelete, "/units/tbsp"},
		{http.MethodPost, "/units/tbsp"},
		{http.MethodPost, "/healthz"},
	} {
		t.Run(tt.method+" "+tt.target, func(t *testing.T) {
			rec := do(h, tt.method, tt.target)
			wantError(t, rec, http.StatusMethodNotAllowed, "method_not_allowed", i18n.EN)
			if allow := rec.Header().Get("Allow"); !strings.Contains(allow, "GET") {
				t.Errorf("Allow = %q; want it to contain GET", allow)
			}
		})
	}
}

func TestInternalErrors(t *testing.T) {
	secret := "disk on fire"

	tests := []struct {
		name   string
		target string
		repo   *fakeUnits
	}{
		{"list", "/units", &fakeUnits{listErr: errors.New(secret)}},
		{"get", "/units/tbsp", &fakeUnits{getErr: errors.New(secret)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			h := api.NewHandler(api.Options{
				Units: tt.repo, Bundle: loadBundle(t), DefaultLang: i18n.EN,
				Logger: slog.New(slog.NewTextHandler(&logs, nil)),
			})
			rec := do(h, http.MethodGet, tt.target)

			wantError(t, rec, http.StatusInternalServerError, "internal", i18n.EN)
			if strings.Contains(rec.Body.String(), secret) {
				t.Errorf("the response leaks the internal error: %s", rec.Body)
			}
			if !strings.Contains(logs.String(), secret) {
				t.Errorf("the internal error was not logged: %q", logs.String())
			}
		})
	}
}

func TestContextErrorsAreNotNotFound(t *testing.T) {
	repo := &fakeUnits{getErr: context.Canceled}
	rec := do(newHandler(t, repo), http.MethodGet, "/units/tbsp")

	if rec.Code == http.StatusNotFound {
		t.Error("a repository error that is not ErrNotFound was reported as 404")
	}
}
