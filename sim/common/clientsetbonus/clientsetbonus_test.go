package clientsetbonus

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	counterspellBonus = 1301013 // Mage 3P: Counterspell cooldown -5 s
	eviscerateBonus   = 1301708 // Rogue 5P: Envenom and Eviscerate cost -5
	lifeTapBonus      = 1301715 // Warlock 5P: Life Tap +20% mana
	counterspellMask  = uint64(1 << 3)
)

// counterspellTable maps the client family the Mage 3P row names.
var counterspellTable = core.ClassMaskTable{
	{Client: core.ClientClassMask{16384}, Engine: counterspellMask},
}

func TestTableModReadsACooldownRow(t *testing.T) {
	mod := TableMod(counterspellBonus, counterspellTable)
	if mod.Kind != core.SpellMod_Cooldown_Flat || mod.TimeValue != -5*time.Second || mod.ClassMask != counterspellMask {
		t.Errorf("got kind %v, %v on mask %d; want a -5s cooldown mod on mask %d", mod.Kind, mod.TimeValue, mod.ClassMask, counterspellMask)
	}
}

func TestTableModReadsACostRow(t *testing.T) {
	table := core.ClassMaskTable{{Client: core.ClientClassMask{131072, 8}, Engine: counterspellMask}}
	mod := TableMod(eviscerateBonus, table)
	if mod.Kind != core.SpellMod_PowerCost_Flat || mod.IntValue != -5 {
		t.Errorf("got kind %v value %d; want a -5 cost mod", mod.Kind, mod.IntValue)
	}
}

func TestTableModPanicsWhenTheTableLacksTheFamily(t *testing.T) {
	assertPanics(t, "empty table", func() { TableMod(counterspellBonus, core.ClassMaskTable{}) })
}

func TestDummyPercentReadsTheRow(t *testing.T) {
	if got := DummyPercent(lifeTapBonus); got != 0.2 {
		t.Errorf("dummy percent %v, want 0.2", got)
	}
}

func TestDummyPercentPanicsOnAModifierRow(t *testing.T) {
	assertPanics(t, "dummy percent on a modifier row", func() { DummyPercent(counterspellBonus) })
}

func assertPanics(t *testing.T, what string, call func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s did not panic", what)
		}
	}()
	call()
}
