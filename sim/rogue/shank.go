package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Shank is a baseline rogue ability in 70755 (SkillLineAbility 59899 and
// 59900, skill line 38): rank 1 1329283 at level 24, rank 2 1329549 at 44
// (SpellLevels.SpellLevel). Both ranks: 5 Energy (SpellPower), 10 sec
// cooldown and 1 sec GCD (SpellCooldowns), e0 effect 31 at 50% weapon
// damage, e1 effect 30 energize Energy (misc 3) 15 / 25, e2 effect 30
// energize combo points (misc 4) 0. SpellEquippedItems needs a one-handed
// weapon and Attributes_3 17235968 is REQUIRES_OFF_HAND_WEAPON (0x1000000),
// ALWAYS_HIT (0x40000), SUPPRESS_TARGET_PROCS (0x20000) and
// SUPPRESS_CASTER_PROCS (0x10000): an off-hand strike that cannot miss, be
// dodged or parried, still rolls its crit, and procs nothing (no poisons,
// no weapon procs, no Seal Fate). It has no effect 121, so its weapon
// damage is not normalized.
//
// Shank's talents are applied here: Flash of Steel (1329284) adds Energy
// to e1 (aura 107 misc 12) and cuts the cooldown (misc 11), Boot Knife
// (1329286) gives it a chance to trigger 1329285 (2 combo points), and
// Venom (1329328) triggers 1310703 on every Shank (venom.go).
const (
	shankRanks          = 2
	shankEnergyCost     = 5.0
	shankCooldown       = 10 * time.Second
	shankWeaponDamage   = 0.50
	shankMaxTalentRanks = 5
)

var shankSpellID = [shankRanks + 1]int32{0, 1329283, 1329549}
var shankLearnLevels = []int{24, 44}

// shankEnergyGenerated is each rank's e1 energize (index 0 unused).
var shankEnergyGenerated = [shankRanks + 1]float64{0, 15, 25}

// Flash of Steel, per rank: the trait curves (TraitDefinitionEffectPoints
// 23393/25223, curves 83037/128124) give +3/6/9/12/15 Energy and
// -0.4/0.8/1.2/1.6/2.0 sec. The talent spell's own base points (15 and
// -2500) are not the rank values.
const (
	flashOfSteelEnergyPerRank   = 3.0
	flashOfSteelCooldownPerRank = 400 * time.Millisecond
)

// Boot Knife: 1329286 e0 triggers 1329285 (e0 effect 30, 2 combo points)
// at the trait curve's 20/40/60/80/100% (TraitDefinitionEffectPoints
// 23401, curve 83008). Shank's tooltip says "1 Combo Point"; the trigger
// row awards 2, and the row wins.
const (
	bootKnifeChancePerRank = 0.20
	bootKnifeComboPoints   = 2
	bootKnifeSpellID       = 1329286
)

func clampTalentRank(rank int32) float64 {
	return float64(max(0, min(rank, shankMaxTalentRanks)))
}

// shankEnergy is a Shank rank's Energy gain with Flash of Steel.
func shankEnergy(rank int, flashOfSteel int32) float64 {
	return shankEnergyGenerated[rank] + flashOfSteelEnergyPerRank*clampTalentRank(flashOfSteel)
}

// shankCooldownWith is Shank's cooldown with Flash of Steel.
func shankCooldownWith(flashOfSteel int32) time.Duration {
	return shankCooldown - time.Duration(clampTalentRank(flashOfSteel))*flashOfSteelCooldownPerRank
}

// bootKnifeChance is Boot Knife's chance at a rank.
func bootKnifeChance(rank int32) float64 {
	return bootKnifeChancePerRank * clampTalentRank(rank)
}

func (rogue *Rogue) registerShankSpell() {
	rank := core.HighestRankAtLevel(shankLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	actionID := core.ActionID{SpellID: shankSpellID[rank]}
	energy := shankEnergy(rank, rogue.Talents.FlashOfSteel)
	energyMetrics := rogue.NewEnergyMetrics(actionID)
	comboChance := bootKnifeChance(rogue.Talents.BootKnife)
	comboMetrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: bootKnifeSpellID})
	venomAura := rogue.registerVenomAura()

	rogue.Shank = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:      SpellCode_RogueShank,
		ClassSpellMask: RogueSpellMaskShank,
		ActionID:       actionID,
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		// Empty: the client suppresses every proc off Shank. isOffHandStrike
		// still treats it as an off-hand swing for off-hand crit.
		ProcMask:      core.ProcMaskEmpty,
		Flags:         core.SpellFlagMeleeMetrics | core.SpellFlagAPL | core.SpellFlagSuppressWeaponProcs | core.SpellFlagSuppressEquipProcs,
		RequiredLevel: shankLearnLevels[rank-1],
		Rank:          rank,

		EnergyCost: core.EnergyCostOptions{
			Cost: shankEnergyCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: shankCooldownWith(rogue.Talents.FlashOfSteel),
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return rogue.AutoAttacks.IsDualWielding
		},

		// Dual Wield Specialization is aura 122, the off-hand damage
		// percent, which every off-hand strike reads.
		DamageMultiplier: shankWeaponDamage * rogue.dwsMultiplier(),
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			baseDamage := spell.Unit.OHWeaponDamage(sim, spell.MeleeAttackPower(target))
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialCritOnly)

			rogue.AddEnergy(sim, energy, energyMetrics)
			if comboChance > 0 && sim.Proc(comboChance, "Boot Knife") {
				rogue.AddComboPoints(sim, bootKnifeComboPoints, target, comboMetrics)
			}
			if venomAura != nil {
				venomAura.Activate(sim)
			}
		},
	})
}
