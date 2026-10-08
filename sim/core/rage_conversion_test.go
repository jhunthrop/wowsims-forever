package core

import (
	"math"
	"testing"
)

// The level-60 conversion is the published base formula evaluated at 60:
// 0.0091107836*60^2 + 3.225598133*60 + 4.2652911. Go's ^ is XOR, and
// 60^2 is 62, which quietly cut the square term to nothing and gave
// every rage class about 16% more rage from damage taken than Classic.
func TestRageConversionSquaresTheLevel(t *testing.T) {
	want := 0.0091107836*3600 + 3.225598133*60 + 4.2652911
	if got := GetRageConversion(60); math.Abs(got-want) > 1e-9 {
		t.Fatalf("GetRageConversion(60) = %v, want %v", got, want)
	}
	wantLow := 0.0215*30*30 + 2.66*30 + 0.89
	if got := GetRageConversion(30); math.Abs(got-wantLow) > 1e-9 {
		t.Fatalf("GetRageConversion(30) = %v, want %v", got, wantLow)
	}
}
