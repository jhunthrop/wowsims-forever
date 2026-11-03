package priest

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// The client's Vampiric Embrace (15286) states the party's share of the
// priest's Shadow damage as its first effect's amount, in percent.
func TestVampiricEmbraceHealsThePartyTheClientsShare(t *testing.T) {
	spell, ok := loadClient(t).ByID(15286)
	if !ok || len(spell.Effects) == 0 {
		t.Fatal("the client states no Vampiric Embrace 15286 effect")
	}
	if want := spell.Effects[0].Amount / 100; vampiricEmbraceHealShare != want {
		t.Errorf("Vampiric Embrace heals %v of Shadow damage, the client states %v", vampiricEmbraceHealShare, want)
	}
}

// 15286's proc row (SpellAuraOptions 128620, ProcTypeMask 0x800A0000) fires on
// harmful magic spells and periodic damage the target takes, and the text says
// "any Shadow spell damage you deal": the priest's own Shadow hits and ticks
// heal the party, nobody else's and no other school's.
func TestVampiricEmbraceHealsOffThePriestsOwnShadowHitsAndTicks(t *testing.T) {
	var priestUnit, otherUnit core.Unit
	hit := &core.SpellResult{Damage: 1000, Outcome: core.OutcomeHit}
	miss := &core.SpellResult{Damage: 0, Outcome: core.OutcomeMiss}
	shadow := &core.Spell{Unit: &priestUnit, SpellSchool: core.SpellSchoolShadow}
	holy := &core.Spell{Unit: &priestUnit, SpellSchool: core.SpellSchoolHoly}
	othersShadow := &core.Spell{Unit: &otherUnit, SpellSchool: core.SpellSchoolShadow}

	for _, c := range []struct {
		name   string
		spell  *core.Spell
		result *core.SpellResult
		want   float64
	}{
		{"own Shadow damage", shadow, hit, 200},
		{"own Shadow miss", shadow, miss, 0},
		{"own Holy damage", holy, hit, 0},
		{"another player's Shadow damage", othersShadow, hit, 0},
	} {
		if got := vampiricEmbraceHealing(&priestUnit, c.spell, c.result); got != c.want {
			t.Errorf("%s heals %v, want %v", c.name, got, c.want)
		}
	}

	aura := vampiricEmbraceAuraConfig(&priestUnit, &otherUnit, func(*core.Simulation, float64) {})
	if aura.OnSpellHitTaken == nil || aura.OnPeriodicDamageTaken == nil {
		t.Error("Vampiric Embrace must heal off both direct Shadow hits and Shadow ticks")
	}
}
