package store

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/johu/larder/internal/i18n"
	"github.com/johu/larder/internal/model"
)

// The queries are split into a shared SELECT and the part that differs per
// method. Both methods bind the arguments in the order of the placeholders:
// the requested language and the fallback language (unit_select.sql), then
// those of the method's own file.

//go:embed queries/unit_select.sql
var unitSelect string

//go:embed queries/list_units.sql
var listUnitsWhere string

//go:embed queries/get_unit.sql
var getUnitByCode string

var (
	listUnitsQuery = unitSelect + listUnitsWhere
	getUnitQuery   = unitSelect + getUnitByCode
)

// SQLiteUnits implements UnitRepository on top of the unit and unit_text
// tables.
type SQLiteUnits struct {
	db *sql.DB
}

// NewSQLiteUnits returns a UnitRepository that reads from db. The
// migrations must have been applied.
func NewSQLiteUnits(db *sql.DB) *SQLiteUnits {
	return &SQLiteUnits{db: db}
}

// scanner is implemented by both *sql.Row and *sql.Rows, so scanUnit serves
// GetUnit and ListUnits.
type scanner interface {
	Scan(dest ...any) error
}

// ListUnits implements UnitRepository. It returns the units ordered by their
// sort order, optionally limited to one dimension (an empty dimension means
// no filter), and an empty, non-nil slice if nothing matches.
func (s *SQLiteUnits) ListUnits(ctx context.Context, lang i18n.Lang, dimension model.Dimension) ([]model.Unit, error) {
	rows, err := s.db.QueryContext(ctx, listUnitsQuery, lang, i18n.Fallback, dimension, dimension)
	if err != nil {
		return nil, fmt.Errorf("store: list units: %w", err)
	}
	defer rows.Close()

	units := make([]model.Unit, 0)
	for rows.Next() {
		unit, err := scanUnit(rows)
		if err != nil {
			return nil, fmt.Errorf("store: list units: scan: %w", err)
		}

		units = append(units, unit)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list units: %w", err)
	}

	return units, nil
}

// GetUnit implements UnitRepository. It returns ErrNotFound if there is no
// unit with this code.
func (s *SQLiteUnits) GetUnit(ctx context.Context, lang i18n.Lang, code string) (model.Unit, error) {
	row := s.db.QueryRowContext(ctx, getUnitQuery, lang, i18n.Fallback, code)
	unit, err := scanUnit(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Unit{}, ErrNotFound
		}

		return model.Unit{}, fmt.Errorf("store: get unit %q: %w", code, err)
	}

	return unit, nil
}

// scanUnit reads one row. The columns must be in the order of
// queries/unit_select.sql.
func scanUnit(s scanner) (model.Unit, error) {
	var unit model.Unit
	err := s.Scan(&unit.Code, &unit.Dimension, &unit.Factor, &unit.Name, &unit.NamePlural, &unit.Abbreviation)
	if err != nil {
		return model.Unit{}, err
	}

	return unit, nil
}
