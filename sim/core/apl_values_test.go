package core

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

func TestValueConst(t *testing.T) {
	sim := &Simulation{}
	unit := &Unit{}
	rot := &APLRotation{
		unit: unit,
	}

	stringVal := rot.newValueConst(&proto.APLValueConst{Val: "test str"})
	if stringVal.GetString(sim) != "test str" {
		t.Fatalf("Unexpected string value %s", stringVal.GetString(sim))
	}

	intVal := rot.newValueConst(&proto.APLValueConst{Val: "10"})
	if intVal.GetInt(sim) != 10 {
		t.Fatalf("Unexpected int value %d", intVal.GetInt(sim))
	}

	floatVal := rot.newValueConst(&proto.APLValueConst{Val: "10.123"})
	if floatVal.GetFloat(sim) != 10.123 {
		t.Fatalf("Unexpected float value %f", floatVal.GetFloat(sim))
	}

	durVal := rot.newValueConst(&proto.APLValueConst{Val: "10.123s"})
	if durVal.GetDuration(sim) != time.Millisecond*10123 {
		t.Fatalf("Unexpected duration value %s", durVal.GetDuration(sim))
	}

	coercedDurVal := rot.coerceTo(floatVal, proto.APLValueType_ValueTypeDuration)
	if _, ok := coercedDurVal.(*APLValueConst); !ok {
		t.Fatalf("Failed to skip coerce wrapper for duration value")
	}
	if coercedDurVal.GetDuration(sim) != time.Millisecond*10123 {
		t.Fatalf("Unexpected coerced duration value %s", coercedDurVal.GetDuration(sim))
	}
}

// Round 19 (priest lane): a rotation line can be scoped to a level range with
// the character's level, e.g. currentLevel >= 40 and < 60.
func TestValueCurrentLevelReadsTheCharactersLevel(t *testing.T) {
	sim := &Simulation{}
	rot := &APLRotation{unit: &Unit{Level: 45}}
	value := rot.newValueCurrentLevel(&proto.APLValueCurrentLevel{})
	if value.Type() != proto.APLValueType_ValueTypeInt {
		t.Fatalf("type = %v, want int", value.Type())
	}
	if got := value.GetInt(sim); got != 45 {
		t.Fatalf("currentLevel = %d, want 45", got)
	}
}

// Round 23 (platform, specced in round 22 C1): mana income over a trailing window, the
// paladin pacing rules' "currentMana + income x remainingTime" term.
func TestValueManaIncomePerSecondReadsTheTrailingWindow(t *testing.T) {
	sim := &Simulation{}
	unit := &Unit{}
	unit.manaBar.unit = unit
	unit.stats[stats.Mana] = 10000
	metrics := &ResourceMetrics{}
	rot := &APLRotation{unit: unit}
	value := rot.newValueManaIncomePerSecond(&proto.APLValueManaIncomePerSecond{})
	if value == nil || value.Type() != proto.APLValueType_ValueTypeFloat {
		t.Fatalf("manaIncomePerSecond = %v, want a float value", value)
	}

	// A steady 20 mana every 2 s is 10 per second once the 10 s default window fills.
	for tick := 1; tick <= 10; tick++ {
		sim.CurrentTime = time.Duration(tick) * 2 * time.Second
		unit.AddMana(sim, 20, metrics)
	}
	if got := value.GetFloat(sim); got != 10 {
		t.Fatalf("income after a full window = %v, want 10", got)
	}

	// Early in the fight it divides by the time elapsed: 20 mana in 4 s.
	early := &Simulation{}
	unit.manaBar.reset()
	unit.currentMana = 0 // reset fills the bar; start empty so the gain lands
	early.CurrentTime = 4 * time.Second
	unit.AddMana(early, 20, metrics)
	if got := value.GetFloat(early); got != 5 {
		t.Fatalf("income at 4 s = %v, want 5", got)
	}

	// Ticks that stop for a whole window leave nothing.
	early.CurrentTime = 15 * time.Second
	if got := value.GetFloat(early); got != 0 {
		t.Fatalf("income a window after the last tick = %v, want 0", got)
	}
}

func TestValueManaIncomePerSecondCountsOnlyWhatTheBarTook(t *testing.T) {
	sim := &Simulation{CurrentTime: 10 * time.Second}
	unit := &Unit{}
	unit.manaBar.unit = unit
	unit.stats[stats.Mana] = 1000
	unit.currentMana = 950
	rot := &APLRotation{unit: unit}
	value := rot.newValueManaIncomePerSecond(&proto.APLValueManaIncomePerSecond{WindowSeconds: 5})
	unit.AddMana(sim, 200, &ResourceMetrics{})
	if got := value.GetFloat(sim); got != 10 {
		t.Fatalf("income with 150 overflowing = %v, want 50 over 5 s = 10", got)
	}
}

// Round 23 (platform): the pacing budget multiplies a rate by the time left
// (manaIncomePerSecond x remainingTime). A product or quotient of a number and a
// duration read the duration through GetFloat and panicked, or multiplied two
// durations in nanoseconds, constants included. It is now worked in floats, the
// duration in seconds.
func TestMathOfANumberAndADurationWorksInSeconds(t *testing.T) {
	sim := &Simulation{CurrentTime: 2 * time.Second}
	rot := &APLRotation{unit: &Unit{Level: 45}}
	level := rot.newValueCurrentLevel(&proto.APLValueCurrentLevel{})
	now := rot.newValueCurrentTime(&proto.APLValueCurrentTime{})
	constant := rot.newValueConst(&proto.APLValueConst{Val: "45"}) // reads as an int
	for _, operands := range [][2]APLValue{{level, now}, {now, level}, {constant, now}, {now, constant}} {
		lhs, rhs := rot.coerceMathOperands(proto.APLValueMath_OpMul, operands[0], operands[1])
		product := &APLValueMath{op: proto.APLValueMath_OpMul, lhs: lhs, rhs: rhs}
		if product.Type() != proto.APLValueType_ValueTypeFloat {
			t.Fatalf("%s x %s is %v, want float", operands[0], operands[1], product.Type())
		}
		if got := product.GetFloat(sim); got != 90 {
			t.Fatalf("%s x %s = %v, want 45 x 2 s = 90", operands[0], operands[1], got)
		}
	}
}
