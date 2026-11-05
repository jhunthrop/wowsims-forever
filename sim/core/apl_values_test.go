package core

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
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

// Round 21 (platform; warrior round 20 X1): a spell registered at every
// level but gated by its RequiredLevel (Whirlwind at 36) is not known
// below that level, so spellIsKnown agrees with the trainer and with a
// guide's level gate.
func TestValueSpellIsKnownHonoursRequiredLevel(t *testing.T) {
	sim := &Simulation{}
	for _, c := range []struct {
		level int32
		want  bool
	}{{30, false}, {35, false}, {36, true}, {60, true}} {
		unit := &Unit{Level: c.level}
		spell := &Spell{Unit: unit, RequiredLevel: 36}
		value := &APLValueSpellIsKnown{spell: spell}
		if got := value.GetBool(sim); got != c.want {
			t.Errorf("level %d: spellIsKnown = %v, want %v", c.level, got, c.want)
		}
	}
	if (&APLValueSpellIsKnown{}).GetBool(sim) {
		t.Errorf("an unregistered spell is known")
	}
}
