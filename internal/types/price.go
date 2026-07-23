package types

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const PriceScaleFactor = 100000

// Price represents a fixed-point price value with scale factor 100000.
// Internal representation is int64 (the raw BIGINT value).
// Public methods convert to/from float64 for backward compatibility
// with the existing IEEE 754 codebase.
//
// Migration path:
//   1. Phase 1 (now): Add Price type, convert at I/O boundaries
//   2. Phase 2: Replace float64 struct fields with Price type
//   3. Phase 3: Replace float64 arithmetic with Price methods
type Price int64

func FromFloat64(f float64) Price {
	return Price(int64(math.Round(f * PriceScaleFactor)))
}

func PriceFromFloat(f float64) Price {
	return FromFloat64(f)
}

func PriceFromInt64(raw int64) Price {
	return Price(raw)
}

func (p Price) ToFloat64() float64 {
	return float64(p) / PriceScaleFactor
}

func (p Price) Float64() float64 {
	return p.ToFloat64()
}

func (p Price) Int64() int64 {
	return int64(p)
}

func (p Price) String() string {
	return fmt.Sprintf("%.5f", p.Float64())
}

func (p Price) MarshalJSON() ([]byte, error) {
	return []byte(p.String()), nil
}

func (p *Price) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("price: invalid value %q: %w", s, err)
	}
	*p = PriceFromFloat(f)
	return nil
}

func (p Price) IsZero() bool {
	return p == 0
}

func (p Price) Add(other Price) Price {
	return p + other
}

func (p Price) Sub(other Price) Price {
	return p - other
}

// MulRational multiplies by a factor expressed as a rational (numerator, denominator).
// No floating-point conversion — purely integer arithmetic.
func (p Price) MulRational(num, den int64) Price {
	if den == 0 {
		return 0
	}
	return Price((int64(p) * num) / den)
}

// MulFloat multiplies by a float64 factor, using math.Round to preserve scale.
// Phase 2 migration: callers should be converted to MulRational where the factor
// can be expressed as a rational; MulFloat remains for cases where a rational
// representation is impractical.
func (p Price) MulFloat(factor float64) Price {
	return Price(int64(math.Round(float64(p) * factor)))
}

// DivFloat divides by a float64 factor, using math.Round to preserve scale.
// Phase 2 migration: prefer MulRational with an inverted rational.
func (p Price) DivFloat(factor float64) Price {
	if factor == 0 {
		return 0
	}
	return Price(int64(math.Round(float64(p) / factor)))
}

// Mul multiplies by a float64 factor. Deprecated in favor of MulRational or MulFloat.
func (p Price) Mul(factor float64) Price {
	return Price(int64(math.Round(float64(p) * factor)))
}

// Div divides by a float64 factor. Deprecated in favor of MulRational or DivFloat.
func (p Price) Div(factor float64) Price {
	if factor == 0 {
		return 0
	}
	return Price(int64(math.Round(float64(p) / factor)))
}

func MinPrice(a, b Price) Price {
	if a < b {
		return a
	}
	return b
}

func MaxPrice(a, b Price) Price {
	if a > b {
		return a
	}
	return b
}

func (p Price) Compare(other Price) int {
	if p < other {
		return -1
	}
	if p > other {
		return 1
	}
	return 0
}

func (p Price) MulInt(qty int64) Price {
	return Price(int64(p) * qty / PriceScaleFactor)
}

func (p Price) SpreadBps(bps float64) Price {
	return PriceFromFloat(p.Float64() * (1 + bps/10000.0))
}

// PnLFromPrices computes realized PnL for a round-trip trade.
func PnLFromPrices(entry Price, exit Price, qty float64, side string) float64 {
	if side == "BUY" {
		return (exit.Float64() - entry.Float64()) * qty
	}
	return (entry.Float64() - exit.Float64()) * qty
}

// Notional computes the notional value of a position.
func Notional(price Price, qty float64) float64 {
	return price.Float64() * qty
}
