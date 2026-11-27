package store

import (
	"context"
	"errors"

	"github.com/johu/larder/internal/i18n"
	"github.com/johu/larder/internal/model"
)

// ErrNotFound is returned, possibly wrapped, when a requested entity does
// not exist. Check for it with errors.Is.
var ErrNotFound = errors.New("store: not found")

// UnitRepository gives read access to the units. The HTTP layer depends on
// this interface only, never on the SQLite implementation.
//
// Names are returned in lang. If a unit has no text in lang, the whole text
// of that unit (name, plural and abbreviation) comes from i18n.Fallback.
type UnitRepository interface {
	// ListUnits returns the units ordered by their sort order. If dimension
	// is not empty, only units of that dimension are returned. It returns
	// an empty, non-nil slice if nothing matches.
	ListUnits(ctx context.Context, lang i18n.Lang, dimension model.Dimension) ([]model.Unit, error)

	// GetUnit returns the unit with the given code. Codes are
	// case-sensitive. It returns ErrNotFound if there is no such unit.
	GetUnit(ctx context.Context, lang i18n.Lang, code string) (model.Unit, error)
}
