// Package model holds the plain data types of the domain. They know nothing
// about storage or HTTP.
package model

// Dimension tells what a unit measures and whether it can be converted.
type Dimension string

const (
	// Mass units convert to grams.
	Mass Dimension = "mass"
	// Volume units convert to milliliters.
	Volume Dimension = "volume"
	// Count units (piece, clove, ...) have no conversion factor but scale
	// with the number of servings.
	Count Dimension = "count"
	// Approx units (pinch) have no conversion factor and must not scale
	// linearly with the number of servings.
	Approx Dimension = "approx"
)

// ParseDimension converts s to a Dimension. It is case-sensitive and reports
// false for an empty or unknown value.
func ParseDimension(s string) (Dimension, bool) {
	switch d := Dimension(s); d {
	case Mass, Volume, Count, Approx:
		return d, true
	default:
		return "", false
	}
}

// BaseUnit returns the code of the unit that the Factor of a unit with this
// dimension refers to: "g" for Mass and "ml" for Volume. It returns "" for
// Count, Approx and unknown dimensions, which have no base unit.
func (d Dimension) BaseUnit() string {
	switch d {
	case Mass:
		return "g"
	case Volume:
		return "ml"
	default:
		return ""
	}
}

// Unit is a unit of measurement with its wording in one language.
type Unit struct {
	// Code is the language-neutral identifier, for example "tbsp".
	Code      string
	Dimension Dimension
	// Factor converts one of this unit to the base unit of its dimension.
	// It is nil for Count and Approx units.
	Factor *float64
	// Name and NamePlural are always set.
	Name       string
	NamePlural string
	// Abbreviation is nil if the unit has none in this language.
	Abbreviation *string
}
