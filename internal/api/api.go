// Package api implements the HTTP layer. It depends on the repository
// interfaces of the store package only, never on SQLite.
package api

import (
	"log/slog"
	"net/http"

	"github.com/johu/larder/internal/i18n"
	"github.com/johu/larder/internal/store"
)

// Options are the dependencies of the handler.
type Options struct {
	// Units gives access to the units. Required.
	Units store.UnitRepository
	// Bundle holds the message catalogs. Required.
	Bundle *i18n.Bundle
	// DefaultLang is used when neither ?lang= nor Accept-Language name a
	// supported language. If it is empty, i18n.Fallback is used.
	DefaultLang i18n.Lang
	// Logger receives the errors that are not shown to the client. If it
	// is nil, slog.Default() is used.
	Logger *slog.Logger
}

// NewHandler returns the handler for the whole API.
//
// Routes (see api_test.go):
//   - GET /healthz: 200 with an empty body; it does not touch the
//     repository
//   - GET /units: {"units": [...]}, in the order the repository returns
//     them; "units" is [] and never null. Optional ?dimension=: one of
//     mass, volume, count, approx (case-sensitive); absent or empty means
//     no filter; any other value is 400 invalid_dimension and the
//     repository is not called
//   - GET /units/{code}: one unit, or 404 unit_not_found. Use errors.Is,
//     as the repository may wrap store.ErrNotFound
//
// Unit JSON: code, dimension, base_unit, factor, name, name_plural and
// abbreviation, always all seven keys. base_unit is "g" for mass and "ml"
// for volume (model.Dimension.BaseUnit); base_unit and factor are null for
// count and approx units; abbreviation is null if there is none.
//
// Language, for every response including errors and 404/405: ?lang=, then
// Accept-Language, then DefaultLang (i18n.Resolve). The resolved language
// goes to the repository, into the Content-Language header, and selects the
// error message. Every response also has "Vary: Accept-Language".
//
// Errors: Content-Type application/json and the body
//
//	{"error": {"code": "<code>", "message": "<localized>"}}
//
// where the message is bundle.Message(lang, "error."+code). Codes:
//
//	400 invalid_dimension
//	404 unit_not_found, not_found (any unknown path, including /units/)
//	405 method_not_allowed (known path, wrong method), with an Allow header
//	500 internal (log the real error, never show it to the client)
//
// All successful responses have Content-Type application/json.
func NewHandler(opts Options) http.Handler {
	// TODO: implement.
	return http.NewServeMux()
}
