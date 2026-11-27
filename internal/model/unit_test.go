package model

import "testing"

func TestParseDimension(t *testing.T) {
	tests := []struct {
		in   string
		want Dimension
		ok   bool
	}{
		{"mass", Mass, true},
		{"volume", Volume, true},
		{"count", Count, true},
		{"approx", Approx, true},
		{"Mass", "", false},
		{"VOLUME", "", false},
		{" mass", "", false},
		{"weight", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := ParseDimension(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Errorf("ParseDimension(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestDimensionBaseUnit(t *testing.T) {
	tests := []struct {
		dimension Dimension
		want      string
	}{
		{Mass, "g"},
		{Volume, "ml"},
		{Count, ""},
		{Approx, ""},
		{Dimension("unknown"), ""},
		{Dimension(""), ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.dimension), func(t *testing.T) {
			if got := tt.dimension.BaseUnit(); got != tt.want {
				t.Errorf("%q.BaseUnit() = %q; want %q", tt.dimension, got, tt.want)
			}
		})
	}
}
