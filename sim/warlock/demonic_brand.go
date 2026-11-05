package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Demonic Brand (talent 1293695, Demonology tier 4): "Your Searing Pain
// generates 17/33/50% less threat and brands the target for 10 sec. Your
// pet's next 2/4/6 attacks against the target deal ((60 - 26) x 1.5 + 14 +
// 0.078 x Shadow spell power) to (... + 17 + ...) Fire or Shadow damage
// based on the pet."
//
// Client 1.60.1.70790 (no hotfix row touches any of these): the talent's
// TraitDefinition 135427 curves give the threat reduction (curve 83195:
// -17/-33/-50, effect 0, aura 108 misc 2 on Searing Pain's family bit 0x100)
// and the charges (curve 103069: 2/4/6, effect 1, aura 107 misc 4 on the
// brand). The brand is 1293696 (10 s, proc on the target's taken melee,
// ranged and harmful spells). The consumed hit is 1293697 (Shadow) or
// 1293698 (Fire): both carry 0 in SpellEffect, a server script, and their
// own Wowhead Forever tooltips state the amount the talent text gives,
// "Consuming a brand deals ((60 - 26) x 1.5 + 14 + 0.078 x Shadow spell
// power) to (...+ 17 ...) Shadow damage", where 60 is the caster level and
// 26 the talent's required level. 1293699 (school Physical, 1 damage) is a
// third consume row no pet uses.
//
// The amount is that one client statement (the talent text and the
// tooltips are one source); no Forever log has a branded pet yet.
const (
	demonicBrandDuration      = 10 * time.Second
	demonicBrandCoefficient   = 0.078
	demonicBrandBaseLevel     = 26
	demonicBrandPerLevel      = 1.5
	demonicBrandMinBonus      = 14
	demonicBrandMaxBonus      = 17
	demonicBrandAuraSpellID   = 1293696
	demonicBrandShadowSpellID = 1293697
	demonicBrandFireSpellID   = 1293698
)

var demonicBrandCharges = [4]int32{0, 2, 4, 6}
var demonicBrandThreatReduction = [4]float64{0, 0.17, 0.33, 0.50}

// demonicBrandDamageRange is one consumed charge's flat roll at level.
func demonicBrandDamageRange(level int) (float64, float64) {
	base := float64(level-demonicBrandBaseLevel) * demonicBrandPerLevel
	return base + demonicBrandMinBonus, base + demonicBrandMaxBonus
}

func (warlock *Warlock) applyDemonicBrand() {
	rank := warlock.Talents.DemonicBrand
	if rank == 0 {
		return
	}
	charges := demonicBrandCharges[rank]

	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if spell.SpellCode == SpellCode_WarlockSearingPain {
			spell.ThreatMultiplier *= 1 - demonicBrandThreatReduction[rank]
		}
	})

	brands := warlock.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return target.GetOrRegisterAura(core.Aura{
			Label:     "Demonic Brand-" + warlock.Label,
			ActionID:  core.ActionID{SpellID: demonicBrandAuraSpellID},
			Duration:  demonicBrandDuration,
			MaxStacks: charges,
		})
	})

	core.MakePermanent(warlock.RegisterAura(core.Aura{
		Label: "Demonic Brand Trigger",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell.SpellCode != SpellCode_WarlockSearingPain || !result.Landed() {
				return
			}
			brand := brands.Get(result.Target)
			brand.Activate(sim)
			brand.SetStacks(sim, charges)
		},
	}))

	for _, basePet := range warlock.BasePets {
		warlock.registerDemonicBrandConsume(basePet, brands)
	}
}

// registerDemonicBrandConsume gives one demon its consumed-brand hit: Fire
// for the Imp, Shadow for the others ("Fire or Shadow damage based on the
// pet"), read off the warlock's spell power of that school.
func (warlock *Warlock) registerDemonicBrandConsume(pet *WarlockPet, brands core.AuraArray) {
	spellID, school, power := int32(demonicBrandShadowSpellID), core.SpellSchoolShadow, stats.ShadowPower
	if pet == warlock.Imp {
		spellID, school, power = demonicBrandFireSpellID, core.SpellSchoolFire, stats.FirePower
	}

	hit := pet.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID},
		SpellSchool: school,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			lo, hi := demonicBrandDamageRange(int(warlock.Level))
			spellPower := warlock.GetStat(stats.SpellPower) + warlock.GetStat(power)
			damage := sim.Roll(lo, hi) + demonicBrandCoefficient*spellPower
			spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMagicHitAndCrit)
		},
	})

	core.MakePermanent(pet.RegisterAura(core.Aura{
		Label: "Demonic Brand Consume",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell == hit || !result.Landed() {
				return
			}
			brand := brands.Get(result.Target)
			if !brand.IsActive() {
				return
			}
			brand.RemoveStack(sim)
			hit.Cast(sim, result.Target)
		},
	}))
}
