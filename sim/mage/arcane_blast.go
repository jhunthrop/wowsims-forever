package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Arcane Blast is the Arcane tree's tier-3 bool talent (node 105806,
// max_rank 1), not a baseline ability, so - like Ice Lance and Cold
// Snap - it is registered from ApplyTalents rather than Initialize: an
// Arcane mage that never spent the point must not have the spell.
//
// The client's rank table (spellconst/mage.json, build 1.60.1.70009)
// collides on both ends of the rank list, which a straight
// regeneration would resolve wrong (see constants_auto_gen.go's
// "skipped" comment beside ArcaneBlastRanks), so this ability keeps its
// own hand-verified arrays in the shape sim/mage/frostbolt.go already
// uses:
//
//   - "Rank 0" is seven different ids sharing the name "Arcane Blast" in
//     the raw dump - an unrelated level-35 legacy/NPC spell (18091) and
//     the stacking buff aura itself (400573, see below) among them -
//     and spellranks.json lists no rank 0 for this talent at all
//     (learnable ranks start at 1), so rank 0 is a zeroed slot here.
//   - Rank 3 has two ids: 1239697 (level 40, 169 damage, family mask
//     1610612736 - the mask every other player rank carries) and 42896
//     (level 76 - above the level cap - 1131 damage, family mask
//     536870912, with a second effect triggering spell 36032). 42896 is
//     an NPC/enrage variant, not the player's rank 3.
//
// The tooltip (ranks 1-5, all "Blasts the target with energy, dealing N
// Arcane damage. Each time you cast Arcane Blast, the damage of all
// your other spells is increased by 10% and the mana cost of Arcane
// Blast is increased by 175%. Effect stacks up to 4 times and lasts 8
// sec or until any other damage spell is cast.") is one spell id per
// rank (400574, 1239696, 1239697, 1239699, 1239700) plus a shared
// stacking-buff spell, id 400573: duration_ms 8000 (the tooltip's 8
// sec) and three generic modifier effects at misc_value 0 (+10, the
// damage bonus), 14 (+175, the cost increase) and 22 (+10, unread here
// - the tooltip names only two effects, and a third generic-modifier
// slot with no corresponding tooltip line is left unmodelled rather
// than guessed).
const ArcaneBlastRanks = 5

var ArcaneBlastSpellId = [ArcaneBlastRanks + 1]int32{0, 400574, 1239696, 1239697, 1239699, 1239700}
var ArcaneBlastLevel = [ArcaneBlastRanks + 1]int{0, 20, 30, 40, 50, 60}
var ArcaneBlastBaseDamage = [ArcaneBlastRanks + 1][]float64{{0, 0}, {54, 54}, {130, 130}, {169, 169}, {274, 274}, {394, 394}}
var ArcaneBlastSpellCoeff = [ArcaneBlastRanks + 1]float64{0, .714, .714, .714, .714, .714}

// arcaneBlastBuffSpellId is 400573, the stacking buff's own id - shared
// by every rank, the way core.WintersChillAura and
// core.ImprovedScorchAura (sim/core/debuffs.go) each use one id
// regardless of which rank of their talent triggered them.
const arcaneBlastBuffSpellId int32 = 400573

const (
	// "the damage of all your other spells is increased by 10%" per
	// stack, up to 4.
	arcaneBlastOtherSpellDamagePerStack = 0.10
	arcaneBlastMaxStacks                = 4
	arcaneBlastDuration                 = time.Second * 8
)

// Every rank's flat ManaCost in the client is 0, but that column is not
// where the client prices this spell: SpellPower.PowerCostPct is 15 for
// every player rank (400574, 1239696, 1239697, 1239699, 1239700 in
// build 1.60.1.70009), i.e. 15% of base mana per cast, the shape the
// warlock summons and Multi-Shot already use. Until 2026-10-07 this file
// read the flat column alone and registered Arcane Blast free, which the
// conformance report could not see (0 -> 0 "match") until spellconst
// carried cost_pct. The generator skips Arcane Blast's arrays (the
// hand-written rank table above), so the percentage is hand-written here
// from the same client rows.
//
// The buff (400573) effect 1 is "mana cost of Arcane Blast is increased
// by 175%" per stack, up to 4 stacks: the fifth consecutive cast costs
// 15% x (1 + 4 x 1.75) = 120% of base mana. Modeled by setting every
// registered rank's Cost.Multiplier from the stack count, the same
// runtime cost-multiplier knob Missile Barrage uses to make Arcane
// Missiles free (sim/mage/talents.go).
const (
	arcaneBlastManaCostPct             = 15.0
	arcaneBlastCostIncreasePctPerStack = 175
)

// setArcaneBlastCostMultiplier applies the buff's cost ramp to every
// rank registered so far: 100 at zero stacks, +175 per stack.
func (mage *Mage) setArcaneBlastCostMultiplier(stacks int32) {
	multiplier := 100 + arcaneBlastCostIncreasePctPerStack*stacks
	for _, spell := range mage.ArcaneBlast {
		if spell != nil && spell.Cost != nil {
			spell.Cost.Multiplier = multiplier
		}
	}
}

func (mage *Mage) registerArcaneBlastSpell() {
	if !mage.Talents.ArcaneBlast {
		return
	}

	mage.ArcaneBlast = make([]*core.Spell, ArcaneBlastRanks+1)

	// "your other spells" for the stacking damage buff - every mage
	// spell this package registers except Arcane Blast itself, captured
	// as each one registers (ApplyTalents runs before Initialize, so
	// most of them do not exist yet when this function runs).
	var otherSpells []*core.Spell
	mage.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(SpellFlagMage) && spell.ClassSpellMask != MageSpellMaskArcaneBlast {
			otherSpells = append(otherSpells, spell)
		}
	})

	mage.ArcaneBlastAura = mage.RegisterAura(core.Aura{
		Label:     "Arcane Blast",
		ActionID:  core.ActionID{SpellID: arcaneBlastBuffSpellId},
		Duration:  arcaneBlastDuration,
		MaxStacks: arcaneBlastMaxStacks,
		OnStacksChange: func(aura *core.Aura, sim *core.Simulation, oldStacks int32, newStacks int32) {
			bonus := arcaneBlastOtherSpellDamagePerStack * float64(newStacks-oldStacks)
			for _, spell := range otherSpells {
				spell.DamageMultiplierAdditive += bonus
			}
			mage.setArcaneBlastCostMultiplier(newStacks)
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			// The cast that refreshes/stacks the buff is Arcane Blast's
			// own ApplyEffects below, not this listener.
			if spell.ClassSpellMask == MageSpellMaskArcaneBlast {
				return
			}
			// "until any other damage spell is cast" - a damage spell
			// is one core.NewSpell would have required a ProcMask for
			// (sim/core/spell.go's validation), so that is the signal
			// used here rather than re-deriving "deals damage" another
			// way.
			if !spell.Flags.Matches(SpellFlagMage) || !spell.ProcMask.Matches(core.ProcMaskSpellDamage) {
				return
			}
			aura.Deactivate(sim)
		},
	})

	for rank := 1; rank <= ArcaneBlastRanks; rank++ {
		config := mage.getArcaneBlastConfig(rank)

		if config.RequiredLevel <= int(mage.Level) {
			mage.ArcaneBlast[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

func (mage *Mage) getArcaneBlastConfig(rank int) core.SpellConfig {
	baseDamageLow := ArcaneBlastBaseDamage[rank][0]
	baseDamageHigh := ArcaneBlastBaseDamage[rank][1]

	return core.SpellConfig{
		ActionID:       core.ActionID{SpellID: ArcaneBlastSpellId[rank]},
		ClassSpellMask: MageSpellMaskArcaneBlast,
		SpellCode:      SpellCode_MageArcaneBlast,
		SpellSchool:    core.SpellSchoolArcane,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          SpellFlagMage | core.SpellFlagAPL,
		// unconfirmed: the client's data carries no projectile speed for
		// this spell; 20 is what Arcane Missiles' tick spell (the same
		// school) uses in this package.
		MissileSpeed: 20,

		RequiredLevel: ArcaneBlastLevel[rank],
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			BaseCost: arcaneBlastManaCostPct / 100,
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond * 2500,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: ArcaneBlastSpellCoeff[rank],

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageLow, baseDamageHigh)
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				if result.Landed() {
					spell.DealDamage(sim, result)
				}
			})

			mage.ArcaneBlastAura.Activate(sim)
			mage.ArcaneBlastAura.AddStack(sim)
		},
	}
}
