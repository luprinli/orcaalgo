// Package fixed implements fixed-point decimal arithmetic.
// DEPRECATED: This package has zero importers except a TODO comment.
// Since P3 (types.Price) migration is complete, either wire into
// types.Price as the canonical fixed-point provider or remove.
package fixed

import "math"

// Scale is the fixed-point scale factor. Prices are stored as int64
// representing price × Scale. E.g., price 452.30123 → 45_230_123.
const Scale = 100_000

// Price is a fixed-point price value stored as integer × Scale.
// Zero value is 0 (no price).
type Price int64

// Qty is a fixed-point quantity stored as integer × Scale.
type Qty int64

// New creates a Price from an int64 raw value (already scaled).
func New(raw int64) Price { return Price(raw) }

// FromFloat64 converts a float64 price to fixed-point by scaling.
// Uses math.Round to minimize IEEE 754 conversion error.
func FromFloat64(f float64) Price { return Price(int64(math.Round(f * Scale))) }

// FromInt64 converts a raw (unscaled) int64 price to fixed-point.
func FromInt64(i int64) Price { return Price(i * Scale) }

// ToFloat64 converts to float64. Use only at display/boundary layers.
func (p Price) ToFloat64() float64 { return float64(p) / Scale }

// Raw returns the underlying int64 storage value (scaled).
func (p Price) Raw() int64 { return int64(p) }

// IsZero returns true if the price is zero.
func (p Price) IsZero() bool { return p == 0 }

// Add returns p + q.
func (p Price) Add(q Price) Price { return Price(int64(p) + int64(q)) }

// Sub returns p - q.
func (p Price) Sub(q Price) Price { return Price(int64(p) - int64(q)) }

// Mul multiplies by an integer scalar, returning a Price.
func (p Price) Mul(n int64) Price { return Price(int64(p) * n) }

// Div divides by an integer scalar, returning a Price.
// Division uses integer truncation; for price ratios use MulFloat.
func (p Price) Div(n int64) Price {
	if n == 0 {
		return 0
	}
	return Price(int64(p) / n)
}

// MulFloat multiplies by a float64 scaling factor and rounds.
// Use for Kelly multipliers, VIX factors, regime multipliers, etc.
func (p Price) MulFloat(f float64) Price {
	return Price(int64(math.Round(float64(p) * f)))
}

// DivFloat divides by a float64 factor and rounds.
func (p Price) DivFloat(f float64) Price {
	if f == 0 {
		return 0
	}
	return Price(int64(math.Round(float64(p) / f)))
}

// Ratio returns p/q as a float64. Use Multiplier for percentage computations.
func (p Price) Ratio(q Price) float64 {
	if q == 0 {
		return 0
	}
	return float64(p) / float64(q)
}

// Gt returns true if p > q.
func (p Price) Gt(q Price) bool { return p > q }

// Lt returns true if p < q.
func (p Price) Lt(q Price) bool { return p < q }

// Gte returns true if p >= q.
func (p Price) Gte(q Price) bool { return p >= q }

// Lte returns true if p <= q.
func (p Price) Lte(q Price) bool { return p <= q }

// Max returns the larger of p and q.
func Max(p, q Price) Price {
	if p > q {
		return p
	}
	return q
}

// Min returns the smaller of p and q.
func Min(p, q Price) Price {
	if p < q {
		return p
	}
	return q
}

// QtyFromFloat64 converts a float64 quantity to fixed-point.
func QtyFromFloat64(f float64) Qty { return Qty(int64(math.Round(f * Scale))) }

// QtyToFloat64 converts to float64.
func (q Qty) ToFloat64() float64 { return float64(q) / Scale }

// QtyRaw returns the underlying int64.
func (q Qty) Raw() int64 { return int64(q) }

// QtyMulFloat scales a quantity by a float factor.
func (q Qty) MulFloat(f float64) Qty {
	return Qty(int64(math.Round(float64(q) * f)))
}
