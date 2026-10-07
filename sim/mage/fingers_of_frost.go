package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Fingers of Frost and Shatter: the Frozen state a mage can make for
// itself.
//
// Client rows (build 1.60.1.70009):
//   - Talent 400647 "Fingers of Frost", max rank 2: "Gives your Chill
//     effects a 15% chance to grant you the Fingers of Frost effect,
//     which treats your next 1 [2] spell[s] cast as if the target were
//     Frozen. Lasts 15 sec." SpellEffect: effect 1 is the 15 (base
//     points), effect 0 the charge count (1 at rank 1, 2 at rank 2).
//     SpellAuraOptions ProcChance 100, ProcTypeMask_0 65536 (a harmful
//     magic spell hit), no class mask: which spells are Chill effects is
//     the client's Chill class bit (SpellClassMask_0 bit 20), carried by
//     Frostbolt, Cone of Cold, Frostfire Bolt and the Chilled aura
//     Improved Blizzard applies. Frost Nova does not carry it.
//   - Effect aura 400669 (duration index 8 = 15000 ms, "Your next $s1
//     spells treat the target as if it were Frozen") and its indicator
//     400670 (cumulative 2).
//   - Shatter 11170, max rank 3: "Increases the critical strike chance
//     of all your spells against Frozen targets by 17%/33%/50%." The
//     spell row's own base points are 1; the per-rank figures are the
//     talent tooltip's.
//
// Model decisions the rows do not carry (documented in
// design/reviews/2026-10-07-frost-fingers-of-frost.md):
//   - The cast that resolves while a charge is held is the one that is
//     treated as Frozen, and it spends the charge as it resolves (as
//     TBC and WotLK did). The chill of that same bolt lands at impact,
//     after the charge is gone, so it can grant a fresh aura: a proc
//     sets the charges to the rank's count and refreshes the 15 s, it
//     does not add to a held aura.
//   - Only direct-damage casts count as "a spell cast": channelled
//     spells (Arcane Missiles, Blizzard) neither spend a charge nor
//     benefit, and neither do non-damaging casts (Ice Barrier, Cold
//     Snap), procs and periodic ticks.
//   - Every chill application rolls the 15% on its own: a landed
//     Frostbolt or Frostfire Bolt, and each Improved Blizzard tick on
//     each target.
const (
	fingersOfFrostBuffSpellId int32 = 400669
	fingersOfFrostProcChance        = 0.15
	fingersOfFrostDuration          = 15 * time.Second
)

var (
	// Charges by talent rank; index 0 is the unspent talent.
	fingersOfFrostCharges = [3]int32{0, 1, 2}
	// Shatter's crit chance points by talent rank.
	shatterCritChancePoints = [4]float64{0, 17, 33, 50}
)

func (mage *Mage) applyFingersOfFrost() {
	if mage.Talents.FingersOfFrost == 0 {
		return
	}

	charges := fingersOfFrostCharges[rankIndex(mage.Talents.FingersOfFrost, fingersOfFrostCharges[:])]
	mage.FingersOfFrostAura = mage.RegisterAura(core.Aura{
		Label:     "Fingers of Frost",
		ActionID:  core.ActionID{SpellID: fingersOfFrostBuffSpellId},
		Duration:  fingersOfFrostDuration,
		MaxStacks: charges,
	})

	mage.RegisterAura(core.Aura{
		Label:    "Fingers of Frost Talent",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			// Blizzard's chill is a proc spell with no hit of its own;
			// it rolls from its own effect (blizzard.go).
			if spell.ProcMask.Matches(core.ProcMaskSpellDamage) && spell.Flags.Matches(SpellFlagChillSpell) && result.Landed() {
				mage.rollFingersOfFrost(sim)
			}
		},
	})
}

// rollFingersOfFrost is one Chill application's chance to grant the aura.
func (mage *Mage) rollFingersOfFrost(sim *core.Simulation) {
	if mage.FingersOfFrostAura == nil || !sim.Proc(fingersOfFrostProcChance, "Fingers of Frost") {
		return
	}
	mage.grantFingersOfFrost(sim)
}

func (mage *Mage) grantFingersOfFrost(sim *core.Simulation) {
	mage.FingersOfFrostAura.Activate(sim)
	mage.FingersOfFrostAura.SetStacks(sim, mage.FingersOfFrostAura.MaxStacks)
}

// consumesFingersOfFrost reports whether spell is a "spell cast" for
// Fingers of Frost and Shatter: a direct-damage cast of the mage's own.
func (mage *Mage) consumesFingersOfFrost(spell *core.Spell) bool {
	return spell.Flags.Matches(SpellFlagMage|core.SpellFlagAPL) &&
		spell.ProcMask.Matches(core.ProcMaskSpellDamage) &&
		!spell.Flags.Matches(core.SpellFlagChanneled)
}

// applyFrozenStateToCasts wraps every eligible spell as it registers so
// its effects resolve with the Frozen state this mage can create:
// a Fingers of Frost charge is spent and makes the target count as
// Frozen for that cast (Ice Lance's multiplier, via isTargetFrozen), and
// Shatter's crit applies to any cast against a Frozen target, whether
// Frost Nova or Fingers of Frost made it so.
func (mage *Mage) applyFrozenStateToCasts() {
	shatterCrit := shatterCritChancePoints[rankIndex(mage.Talents.Shatter, shatterCritChancePoints[:])] * core.CritRatingPerCritChance
	if mage.FingersOfFrostAura == nil && shatterCrit == 0 {
		return
	}

	mage.OnSpellRegistered(func(spell *core.Spell) {
		if !mage.consumesFingersOfFrost(spell) {
			return
		}
		resolve := spell.ApplyEffects
		spell.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			fingers := mage.spendFingersOfFrostCharge(sim)
			frozen := fingers || mage.targetHasFrozenAura(target)
			if !frozen {
				resolve(sim, target, spell)
				return
			}
			mage.fingersFreezeCast = fingers
			spell.BonusCritRating += shatterCrit
			resolve(sim, target, spell)
			spell.BonusCritRating -= shatterCrit
			mage.fingersFreezeCast = false
		}
	})
}

func (mage *Mage) spendFingersOfFrostCharge(sim *core.Simulation) bool {
	if !mage.FingersOfFrostAura.IsActive() {
		return false
	}
	mage.FingersOfFrostAura.RemoveStack(sim)
	return true
}
