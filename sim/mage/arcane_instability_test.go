package mage

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Arcane Instability 15058 (client 1.60.1.70707) states its crit as effect 2,
// aura 290 (SPELL_AURA_MOD_CRIT_PCT) with no class mask: "your critical
// strike chance", every crit the mage rolls, wand shots and item procs
// included. Only its damage half (effects 0 and 1, aura 108) is a class-mask
// spell modifier. The engine had both halves as class-spell modifiers, so a
// wand shot took none of the crit.
func TestArcaneInstabilityCritReachesEveryCrit(t *testing.T) {
	critOf := func(rank int) (shoot, frostbolt float64) {
		talents := talentStringWithRank(t, ForeverFrostTalents, "arcane_instability", rank)
		built, _, target := newFrostMageSimWithTalents(t, core.DefaultTargetProtoLvl60, talents)
		built.Equipment[proto.ItemSlot_ItemSlotRanged] = core.Item{
			ID:               99999,
			RangedWeaponType: proto.RangedWeaponType_RangedWeaponTypeWand,
			WeaponDamageMin:  20,
			WeaponDamageMax:  40,
			SwingSpeed:       1.5,
		}
		built.registerShootSpell()
		var bolt *core.Spell
		for _, spell := range built.Frostbolt {
			if spell != nil {
				bolt = spell
			}
		}
		return built.Shoot.SpellCritChance(target), bolt.SpellCritChance(target)
	}

	shoot0, bolt0 := critOf(0)
	shoot3, bolt3 := critOf(3)
	for name, delta := range map[string]float64{"Shoot": shoot3 - shoot0, "Frostbolt": bolt3 - bolt0} {
		if math.Abs(delta-0.03) > 1e-9 {
			t.Errorf("Arcane Instability 3/3 moved %s crit by %.4f, want 0.03", name, delta)
		}
	}
}
