package warrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

func warriorAtLevel(level int32) *Warrior {
	return &Warrior{
		Character: core.Character{Unit: core.Unit{Level: level}},
		Talents:   &proto.WarriorTalents{},
	}
}

// Berserker Stance 2458 is SpellLevels BaseLevel 30 (the level-30 quest):
// a warrior below 30 has only Battle and Defensive Stance, so any
// Berserker Stance cast waits for 30. (Fury's default stance waits for
// Whirlwind at 36: round9_test.go.)
func TestBerserkerStanceIsLearnedAtThirty(t *testing.T) {
	for _, tc := range []struct {
		level int32
		knows bool
	}{{20, false}, {29, false}, {30, true}, {60, true}} {
		w := warriorAtLevel(tc.level)
		if got := w.knowsStance(BerserkerStance); got != tc.knows {
			t.Errorf("level %d knows Berserker Stance = %v, want %v", tc.level, got, tc.knows)
		}
	}
	if !warriorAtLevel(10).knowsStance(DefensiveStance) || warriorAtLevel(9).knowsStance(DefensiveStance) {
		t.Error("Defensive Stance 71 is BaseLevel 10")
	}
}

// Whirlwind 1680 is SpellLevels BaseLevel 36.
func TestWhirlwindIsLearnedAtThirtySix(t *testing.T) {
	if warriorAtLevel(35).knowsWhirlwind() {
		t.Error("a level-35 warrior knows Whirlwind; its BaseLevel is 36")
	}
	if !warriorAtLevel(36).knowsWhirlwind() {
		t.Error("a level-36 warrior does not know Whirlwind")
	}
}
