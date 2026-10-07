package core

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
)

const MaxRage = 100.0
const ThreatPerRageGained = 5

// OnRageChange is called any time rage is increased.
type OnRageChange func(aura *Aura, sim *Simulation, metrics *ResourceMetrics)

type rageBar struct {
	unit *Unit

	damageDealtMultiplier float64 // Multiplier for rage generation from damage dealt
	damageTakenMultiplier float64 // Multiplier for rage generation from damage taken
	// offHandDamageDealtMultiplier scales rage from off-hand auto-attacks
	// only, on top of damageDealtMultiplier: Forever's Dual Wield
	// Specialization "increases ... off-hand Rage generation by 20%" a
	// point (build 1.60.1.70009 rank text). 1 when nothing has raised it.
	offHandDamageDealtMultiplier float64

	flatDamageDealtBonusRage float64
	flatDamageTakenBonusRage float64

	startingRage float64
	currentRage  float64

	RageRefundMetrics *ResourceMetrics
}

type RageBarOptions struct {
	StartingRage          float64
	DamageDealtMultiplier float64
	DamageTakenMultiplier float64
}

func GetRageConversion(attacker_level int32) float64 {
	if attacker_level == 25 {
		return 82.25 // Tested
	} else if attacker_level == 40 {
		return 140.5 // Tested
	} else if attacker_level < 45 {
		// Poor fit, but better then current formula below 45
		return 0.0215*float64(attacker_level^2) + 2.66*float64(attacker_level) + 0.89
	} else {
		// Rage conversion is adjusted according to target stats (https://web.archive.org/web/20201118213002/https://blue.mmo-champion.com/topic/18325-the-new-rage-formula-by-kalgan/)\
		// So this is probably only the base value formula and will be slightly wrong for most target
		return 0.0091107836*float64(attacker_level^2) + 3.225598133*float64(attacker_level) + 4.2652911
	}
}

// Forever's normalized rage. The beta replaced vanilla's damage-based
// rage (GetRageConversion above, now used only for rage from damage
// TAKEN) with a fixed amount per successful auto-attack that depends on
// the weapon's speed and hand, not on the damage it dealt, so a level-60
// epic and a grey of the same speed generate the same rage per swing.
//
// Sources, in order of confidence:
//   - Blizzard's 1 October 2026 beta development notes: "Additional
//     Rage generated from landing Critical Strikes increased to 100%
//     increased Rage (was 75%)" - a crit pays double
//     (normalizedRageCritMultiplier).
//   - Community measurement on the 24 September beta build (Icy Veins,
//     25 September 2026): one-handers generate weapon speed x 3.46 per
//     landed swing, two-handers weapon speed x 4.50
//     (normalizedRagePerSpeedOneHand / TwoHand). These are measured, not
//     stated, numbers; the owner's own combat log is the check.
//   - The off-hand constant is NOT measured anywhere public. It is taken
//     as half the one-hand constant, the ratio Blizzard's own TBC-era
//     normalization used for off-hand swings (hit factor 1.75 against a
//     main hand's 3.5, which the measured 3.46 all but names); an
//     assumption until a log settles it.
//
// Talents that add rage on top (Unbridled Wrath, Dual Wield
// Specialization's off-hand clause) keep their own hooks.
const (
	normalizedRagePerSpeedOneHand = 3.46
	normalizedRagePerSpeedTwoHand = 4.50
	normalizedRageOffHandFactor   = 0.5
	normalizedRageCritMultiplier  = 2.0
)

// normalizedSwingRage is the rage one landed auto-attack with weapon
// generates before any multiplier or flat bonus the rage bar carries.
func normalizedSwingRage(weapon *Weapon, offHand bool, crit bool) float64 {
	if weapon == nil || weapon.SwingSpeed <= 0 {
		return 0
	}
	perSpeed := normalizedRagePerSpeedOneHand
	if weapon.TwoHanded {
		perSpeed = normalizedRagePerSpeedTwoHand
	}
	rage := weapon.SwingSpeed * perSpeed
	if offHand {
		rage *= normalizedRageOffHandFactor
	}
	if crit {
		rage *= normalizedRageCritMultiplier
	}
	return rage
}

func (unit *Unit) EnableRageBar(options RageBarOptions) {
	rageFromDamageTakenMetrics := unit.NewRageMetrics(ActionID{OtherID: proto.OtherAction_OtherActionDamageTaken})

	unit.SetCurrentPowerBar(RageBar)
	unit.RegisterAura(Aura{
		Label:    "RageBar",
		Duration: NeverExpires,
		OnInit: func(aura *Aura, sim *Simulation) {
			// Initialize resource metrics for rage gain for auto attacks here to make sure tag is correct.
			// Extra attacks change the tag from 1 to 3 for mh hits.
			mhSpell := unit.AutoAttacks.MHAuto()
			if mhSpell != nil {
				mhSpell.ResourceMetrics = unit.NewRageMetrics(mhSpell.ActionID)
			}
			ohSpell := unit.AutoAttacks.OHAuto()
			if ohSpell != nil {
				ohSpell.ResourceMetrics = unit.NewRageMetrics(ohSpell.ActionID)
			}
		},
		OnReset: func(aura *Aura, sim *Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *Aura, sim *Simulation, spell *Spell, result *SpellResult) {
			if unit.GetCurrentPowerBar() != RageBar {
				return
			}
			if result.Outcome.Matches(OutcomeMiss) {
				return
			}
			if spell.ProcMask != ProcMaskMeleeMHAuto && spell.ProcMask != ProcMaskMeleeOHAuto {
				return
			}

			// Forever normalizes rage from auto-attacks: a fixed amount
			// per swing from the weapon's own speed and hand, doubled on
			// a critical strike, with no damage term at all (see
			// normalizedSwingRage). A dodged or parried swing still pays
			// its base, as vanilla's damage-based formula did through
			// PreOutcomeDamage; a miss pays nothing (returned above).
			offHand := spell.ProcMask == ProcMaskMeleeOHAuto
			weapon := unit.AutoAttacks.MH()
			if offHand {
				weapon = unit.AutoAttacks.OH()
			}
			generatedRage := normalizedSwingRage(weapon, offHand, result.DidCrit())
			generatedRage *= unit.rageBar.damageDealtMultiplier
			if offHand {
				generatedRage *= unit.rageBar.offHandDamageDealtMultiplier
			}
			generatedRage += unit.rageBar.flatDamageDealtBonusRage

			var metrics *ResourceMetrics
			if spell.Cost != nil {
				metrics = spell.Cost.SpellCostFunctions.(*RageCost).ResourceMetrics
			} else {
				// Seems like only auto attacks are using this. See OnInit handler of this aura.
				if spell.ResourceMetrics == nil {
					panic(fmt.Sprintf("Spell ResourceMetrics are nil for spell %v", spell.ActionID))
				}
				metrics = spell.ResourceMetrics
			}
			unit.AddRage(sim, generatedRage, metrics)
		},
		OnSpellHitTaken: func(aura *Aura, sim *Simulation, spell *Spell, result *SpellResult) {
			if unit.GetCurrentPowerBar() != RageBar {
				return
			}
			rageConversionDamageTaken := GetRageConversion(spell.Unit.Level)
			generatedRage := result.Damage * 2.5 / rageConversionDamageTaken
			generatedRage *= unit.rageBar.damageTakenMultiplier
			generatedRage += unit.rageBar.flatDamageTakenBonusRage
			unit.AddRage(sim, generatedRage, rageFromDamageTakenMetrics)
		},
	})

	// Not a real spell, just holds metrics from rage gain threat.
	unit.RegisterSpell(SpellConfig{
		ActionID: ActionID{OtherID: proto.OtherAction_OtherActionRageGain},
	})

	unit.rageBar = rageBar{
		unit:                         unit,
		damageDealtMultiplier:        options.DamageDealtMultiplier,
		damageTakenMultiplier:        options.DamageTakenMultiplier,
		offHandDamageDealtMultiplier: 1,
		startingRage:                 max(0, min(options.StartingRage, MaxRage)),
		RageRefundMetrics:            unit.NewRageMetrics(ActionID{OtherID: proto.OtherAction_OtherActionRefund}),
	}
}

func (unit *Unit) HasRageBar() bool {
	return unit.rageBar.unit != nil
}

func (unit *Unit) AddDamageDealtRageMultiplier(multi float64) {
	unit.rageBar.damageDealtMultiplier *= multi
}

func (unit *Unit) AddDamageTakenRageMultiplier(multi float64) {
	unit.rageBar.damageTakenMultiplier *= multi
}

// AddOffHandDamageDealtRageMultiplier scales rage from off-hand
// auto-attacks only (Dual Wield Specialization's own clause).
func (unit *Unit) AddOffHandDamageDealtRageMultiplier(multi float64) {
	unit.rageBar.offHandDamageDealtMultiplier *= multi
}

func (unit *Unit) AddDamageDealtRageBonus(bonus float64) {
	unit.rageBar.flatDamageDealtBonusRage += bonus
}

func (unit *Unit) AddDamageTakenRageBonus(bonus float64) {
	unit.rageBar.flatDamageTakenBonusRage += bonus
}

func (rb *rageBar) CurrentRage() float64 {
	return rb.currentRage
}

func (rb *rageBar) AddRage(sim *Simulation, amount float64, metrics *ResourceMetrics) {
	if amount < 0 {
		panic("Trying to add negative rage!")
	}

	newRage := min(rb.currentRage+amount, MaxRage)
	metrics.AddEvent(amount, newRage-rb.currentRage)

	if sim.Log != nil {
		rb.unit.Log(sim, "Gained %0.3f rage from %s (%0.3f --> %0.3f).", amount, metrics.ActionID, rb.currentRage, newRage)
	}

	rb.currentRage = newRage
	if !sim.Options.Interactive && rb.unit.Rotation != nil {
		rb.unit.Rotation.DoNextAction(sim)
	}
	StartDelayedAction(sim, DelayedActionOptions{
		DoAt: sim.CurrentTime + time.Millisecond*1,
		OnAction: func(sim *Simulation) {
			rb.unit.OnRageChange(sim, metrics)
		},
	})

}

func (rb *rageBar) SpendRage(sim *Simulation, amount float64, metrics *ResourceMetrics) {
	if amount < 0 {
		panic("Trying to spend negative rage!")
	}

	newRage := rb.currentRage - amount
	metrics.AddEvent(-amount, -amount)

	if sim.Log != nil {
		rb.unit.Log(sim, "Spent %0.3f rage from %s (%0.3f --> %0.3f).", amount, metrics.ActionID, rb.currentRage, newRage)
	}

	rb.currentRage = newRage

	rb.unit.OnRageChange(sim, metrics)
}

func (rb *rageBar) reset(_ *Simulation) {
	if rb.unit == nil {
		return
	}

	rb.currentRage = rb.startingRage
}

func (rb *rageBar) doneIteration() {
	if rb.unit == nil {
		return
	}

	rageGainSpell := rb.unit.GetSpell(ActionID{OtherID: proto.OtherAction_OtherActionRageGain})

	for _, resourceMetrics := range rb.unit.Metrics.resources {
		if resourceMetrics.Type != proto.ResourceType_ResourceTypeRage {
			continue
		}
		if resourceMetrics.ActionID.SameActionIgnoreTag(ActionID{OtherID: proto.OtherAction_OtherActionDamageTaken}) {
			continue
		}
		if resourceMetrics.ActionID.SameActionIgnoreTag(ActionID{OtherID: proto.OtherAction_OtherActionRefund}) {
			continue
		}
		if resourceMetrics.ActualGainForCurrentIteration() <= 0 {
			continue
		}

		// Need to exclude rage gained from white hits. Rather than have a manual list of all IDs that would
		// apply here (autos, WF attack, sword spec procs, etc), just check if the effect caused any damage.
		sourceSpell := rb.unit.GetSpell(resourceMetrics.ActionID)
		if sourceSpell != nil && sourceSpell.SpellMetrics[0].TotalDamage > 0 {
			continue
		}

		rageGainSpell.SpellMetrics[0].Casts += resourceMetrics.EventsForCurrentIteration()
		rageGainSpell.ApplyAOEThreatIgnoreMultipliers(resourceMetrics.ActualGainForCurrentIteration() * ThreatPerRageGained)
	}
}

type RageCostOptions struct {
	Cost float64

	Refund        float64
	RefundMetrics *ResourceMetrics // Optional, will default to unit.RageRefundMetrics if not supplied.
}
type RageCost struct {
	Refund          float64
	RefundMetrics   *ResourceMetrics
	ResourceMetrics *ResourceMetrics
}

func newRageCost(spell *Spell, options RageCostOptions) *SpellCost {
	if options.Refund > 0 && options.RefundMetrics == nil {
		options.RefundMetrics = spell.Unit.RageRefundMetrics
	}

	return &SpellCost{
		spell:      spell,
		BaseCost:   options.Cost,
		Multiplier: 100,
		SpellCostFunctions: &RageCost{
			Refund:          options.Refund * options.Cost,
			RefundMetrics:   options.RefundMetrics,
			ResourceMetrics: spell.Unit.NewRageMetrics(spell.ActionID),
		},
	}
}

func (rc *RageCost) CostType() CostType {
	return CostTypeRage
}

func (rc *RageCost) MeetsRequirement(_ *Simulation, spell *Spell) bool {
	spell.CurCast.Cost = spell.Cost.GetCurrentCost()
	return spell.Unit.CurrentRage() >= spell.CurCast.Cost
}
func (rc *RageCost) CostFailureReason(sim *Simulation, spell *Spell) string {
	return fmt.Sprintf("not enough rage (Current Rage = %0.03f, Rage Cost = %0.03f)", spell.Unit.CurrentRage(), spell.CurCast.Cost)
}
func (rc *RageCost) SpendCost(sim *Simulation, spell *Spell) {
	if spell.CurCast.Cost > 0 {
		spell.Unit.SpendRage(sim, spell.CurCast.Cost, rc.ResourceMetrics)
	}
}
func (rc *RageCost) IssueRefund(sim *Simulation, spell *Spell) {
	if rc.Refund > 0 {
		spell.Unit.AddRage(sim, rc.Refund, rc.RefundMetrics)
	}
}
