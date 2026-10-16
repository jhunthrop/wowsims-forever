package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Arcane Blast is the Arcane tree's tier-3 bool talent (node 105806,
// max_rank 1), not a baseline ability, so - like Ice Lance and Cold
// Snap - it is registered from ApplyTalents rather than Initialize: an
// Arcane mage that never spent the point must not have the spell.
//
// The client's rank table (spellconst/mage.json, since build 1.60.1.70009)
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
// The tooltip (ranks 1-5, client 1.60.1.70425: "Blasts the target with
// energy, dealing N Arcane damage. Each time you cast Arcane Blast, you
// gain 15% of your Intellect as Spell Damage and the mana cost of Arcane
// Blast is increased by 175%. Effect stacks up to 4 times and lasts 15
// sec from time of initial application.") is one spell id per rank
// (400574, 1239696, 1239697, 1239699, 1239700; SpellEffect base points
// 32/78/101/168/236 with variance 0.151, points per level
// 0.5/0.8/1.0/1.3/1.6, coefficient 0.429) plus a shared stacking-buff
// spell, id 400573: aura 174 (spell damage from a percentage of a stat)
// at +15 of Intellect and a cost modifier of +175 a stack. The
// 1.60.1.70374 version was +10% damage to the mage's other spells for 8
// sec, dropped by any other damage spell; both halves are gone from the
// client's text and rows.
const ArcaneBlastRanks = 5

var ArcaneBlastSpellId = [ArcaneBlastRanks + 1]int32{0, 400574, 1239696, 1239697, 1239699, 1239700}
var ArcaneBlastLevel = [ArcaneBlastRanks + 1]int{0, 20, 30, 40, 50, 60}
var ArcaneBlastBaseDamage = [ArcaneBlastRanks + 1][]float64{{0, 0}, {29.584, 34.416}, {72.111, 83.889}, {93.3745, 108.6255}, {155.316, 180.684}, {218.182, 253.818}}
var ArcaneBlastPointsPerLevel = [ArcaneBlastRanks + 1]float64{0, 0.5, 0.8, 1.0, 1.3, 1.6}
var ArcaneBlastMaxLevel = [ArcaneBlastRanks + 1]int{0, 28, 38, 48, 58, 68}
var ArcaneBlastSpellCoeff = [ArcaneBlastRanks + 1]float64{0, 0.429, 0.429, 0.429, 0.429, 0.429}

// arcaneBlastBuffSpellId is 400573, the stacking buff's own id - shared
// by every rank, the way core.WintersChillAura and
// core.ImprovedScorchAura (sim/core/debuffs.go) each use one id
// regardless of which rank of their talent triggered them.
const arcaneBlastBuffSpellId int32 = 400573

const (
	// "you gain 15% of your Intellect as Spell Damage" per stack, up to 4.
	arcaneBlastIntellectShare = 0.15
	arcaneBlastMaxStacks      = 4
	// "lasts 15 sec from time of initial application": a further cast
	// adds a stack without renewing the 15 sec.
	arcaneBlastDuration = time.Second * 15
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

	// The spell power each stack granted, so the whole grant comes back
	// off when the buff ends even if Intellect moved meanwhile.
	var grantedSpellPower float64
	mage.ArcaneBlastAura = mage.RegisterAura(core.Aura{
		Label:     "Arcane Blast",
		ActionID:  core.ActionID{SpellID: arcaneBlastBuffSpellId},
		Duration:  arcaneBlastDuration,
		MaxStacks: arcaneBlastMaxStacks,
		OnStacksChange: func(aura *core.Aura, sim *core.Simulation, oldStacks int32, newStacks int32) {
			if newStacks > oldStacks {
				grant := arcaneBlastIntellectShare * mage.GetStat(stats.Intellect) * float64(newStacks-oldStacks)
				mage.AddStatDynamic(sim, stats.SpellPower, grant)
				grantedSpellPower += grant
			} else if newStacks == 0 {
				mage.AddStatDynamic(sim, stats.SpellPower, -grantedSpellPower)
				grantedSpellPower = 0
			}
			mage.setArcaneBlastCostMultiplier(newStacks)
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
	roll := mage.clientRoll(ArcaneBlastBaseDamage[rank], ArcaneBlastPointsPerLevel[rank], ArcaneBlastLevel[rank], ArcaneBlastMaxLevel[rank])
	baseDamageLow, baseDamageHigh := roll[0], roll[1]

	return core.SpellConfig{
		ActionID:         core.ActionID{SpellID: ArcaneBlastSpellId[rank]},
		ClassSpellMask:   MageSpellMaskArcaneBlast,
		SpellCode:        SpellCode_MageArcaneBlast,
		SpellSchool:      core.SpellSchoolArcane,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		ClientBaseDamage: roll,
		Flags:            SpellFlagMage | core.SpellFlagAPL,
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
				GCD: core.GCDDefault,
				// SpellMisc.CastingTimeIndex on every rank reads 1.5 sec in
				// client 1.60.1.70425 (2.5 sec before).
				CastTime: time.Millisecond * 1500,
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

			// "lasts 15 sec from time of initial application": only the
			// first cast starts the clock.
			if !mage.ArcaneBlastAura.IsActive() {
				mage.ArcaneBlastAura.Activate(sim)
			}
			mage.ArcaneBlastAura.AddStack(sim)
		},
	}
}
