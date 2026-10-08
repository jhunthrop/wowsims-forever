package item_sets

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Readers for the client rows the hand-written set models are built from.
// A model names its bonus spell and takes every amount, chance and duration
// from that spell's row and the row of the spell it triggers, so a number
// is never typed twice.

// The client's effect and aura kinds the models read.
const (
	clientEffectHealthLeech int32 = 9
	clientEffectEnergize    int32 = 30

	clientAuraDummy            int32 = 4
	clientAuraPeriodicDamage   int32 = 3
	clientAuraPeriodicEnergize int32 = 24
	clientAuraSkill            int32 = 30
	clientAuraModTargetResist  int32 = 123
	clientAuraAttackSpeed      int32 = 319
)

// The client's power types, as an energize effect names them in Misc0.
const (
	clientPowerMana   int32 = 0
	clientPowerRage   int32 = 1
	clientPowerEnergy int32 = 3
)

const (
	// The client stores rage in tenths.
	clientRageTenths = 10

	// SpellAuraOptions.ProcChance of a "chance on" bonus whose chance the
	// client does not state (its tooltip prints no percentage). A stated
	// chance is below this.
	clientUnstatedProcChance = 100

	percent = 100.0
)

// triggeredSpell is a spell a bonus triggers, with its client row.
type triggeredSpell struct {
	id  int32
	row core.ClientSpell
}

func (spell triggeredSpell) actionID() core.ActionID {
	return core.ActionID{SpellID: spell.id}
}

func (spell triggeredSpell) duration() time.Duration {
	return time.Duration(spell.row.DurationMS) * time.Millisecond
}

// icdOf is a proc bonus's internal cooldown, the client's
// SpellAuraOptions.ProcCategoryRecovery.
func icdOf(row core.ClientSpell) time.Duration {
	return time.Duration(row.InternalCooldownMS) * time.Millisecond
}

// spellOf is a spell a model reads by id.
func spellOf(id int32) triggeredSpell {
	return triggeredSpell{id: id, row: core.MustClientSpellRow(id)}
}

// triggerOf is the spell a proc bonus triggers.
func triggerOf(bonusID int32) triggeredSpell {
	for _, effect := range core.MustClientSpellRow(bonusID).Effects {
		if effect.Trigger != 0 {
			return spellOf(effect.Trigger)
		}
	}
	panic(fmt.Sprintf("item_sets: client spell %d triggers nothing", bonusID))
}

// effectOf is the first effect of a spell that keep accepts.
func effectOf(spellID int32, row core.ClientSpell, keep func(core.ClientEffect) bool) core.ClientEffect {
	for _, effect := range row.Effects {
		if keep(effect) {
			return effect
		}
	}
	panic(fmt.Sprintf("item_sets: client spell %d (%s) lacks the effect the model reads", spellID, row.Name))
}

func auraEffectOf(spellID int32, row core.ClientSpell, aura int32) core.ClientEffect {
	return effectOf(spellID, row, func(effect core.ClientEffect) bool { return effect.Aura == aura })
}

// procRate is how often a bonus procs: a chance per eligible event, or
// procs per minute for a melee or ranged hit.
type procRate struct {
	chance float64
	ppm    float64
}

// rateOf is the bonus's stated chance, or unstated when the client states
// none. It stops the engine at init when neither is available.
func rateOf(bonusID int32, unstated procRate) procRate {
	row := core.MustClientSpellRow(bonusID)
	if row.ProcChance > 0 && row.ProcChance < clientUnstatedProcChance {
		return procRate{chance: float64(row.ProcChance) / percent}
	}
	if unstated == (procRate{}) {
		panic(fmt.Sprintf("item_sets: client spell %d (%s) states no proc chance and the model gives none", bonusID, row.Name))
	}
	return unstated
}

// alwaysProcs is a bonus that procs on every eligible event: the client
// gives it a 100% chance and an internal cooldown that only spaces the
// procs.
var alwaysProcs = procRate{chance: 1}

// procKind is the event a bonus procs on.
type procKind struct {
	name     string
	callback core.AuraCallback
	procMask core.ProcMask
	outcome  core.HitOutcome
}

var (
	onSpellcast = procKind{
		name:     "spellcast",
		callback: core.CallbackOnCastComplete,
		procMask: core.ProcMaskSpellDamage | core.ProcMaskSpellHealing,
	}
	onHarmfulSpellcast = procKind{
		name:     "harmful spellcast",
		callback: core.CallbackOnCastComplete,
		procMask: core.ProcMaskSpellDamage,
	}
	onAutoattack = procKind{
		name:     "autoattack",
		callback: core.CallbackOnSpellHitDealt,
		procMask: core.ProcMaskWhiteHit,
		outcome:  core.OutcomeLanded,
	}
	onRangedAttack = procKind{
		name:     "ranged attack",
		callback: core.CallbackOnSpellHitDealt,
		procMask: core.ProcMaskRanged,
		outcome:  core.OutcomeLanded,
	}
	onMeleeAutoattack = procKind{
		name:     "melee autoattack",
		callback: core.CallbackOnSpellHitDealt,
		procMask: core.ProcMaskMeleeWhiteHit,
		outcome:  core.OutcomeLanded,
	}
	onMeleeHit = procKind{
		name:     "melee hit",
		callback: core.CallbackOnSpellHitDealt,
		procMask: core.ProcMaskMelee,
		outcome:  core.OutcomeLanded,
	}
	onMeleeOrRangedHit = procKind{
		name:     "hit",
		callback: core.CallbackOnSpellHitDealt,
		procMask: core.ProcMaskMeleeOrRanged,
		outcome:  core.OutcomeLanded,
	}
	onDamageTaken = procKind{
		name:     "damage taken",
		callback: core.CallbackOnSpellHitTaken,
		outcome:  core.OutcomeLanded,
	}
)

// procLabel is the label of the aura that carries a bonus's proc: the
// trigger spells share their names with the bonus and the buff they grant.
func procLabel(bonusName string, kind procKind) string {
	return fmt.Sprintf("%s (%s)", bonusName, kind.name)
}

func (kind procKind) trigger(bonusName string, actionID core.ActionID, rate procRate, handler core.ProcHandler) core.ProcTrigger {
	return core.ProcTrigger{
		Name:       procLabel(bonusName, kind),
		ActionID:   actionID,
		Callback:   kind.callback,
		ProcMask:   kind.procMask,
		Outcome:    kind.outcome,
		ProcChance: rate.chance,
		PPM:        rate.ppm,
		Handler:    handler,
	}
}

// resourceGrant is what an energize effect restores.
type resourceGrant struct {
	power  int32
	amount float64
}

func grantOf(spell triggeredSpell) resourceGrant {
	effect := effectOf(spell.id, spell.row, func(effect core.ClientEffect) bool {
		return effect.Effect == clientEffectEnergize
	})
	grant := resourceGrant{power: effect.Misc0, amount: effect.Points}
	if grant.power == clientPowerRage {
		grant.amount /= clientRageTenths
	}
	return grant
}

// giver is the function that restores the grant to a character, and does
// nothing for a character without that resource.
func (grant resourceGrant) giver(character *core.Character, actionID core.ActionID) func(*core.Simulation) {
	switch grant.power {
	case clientPowerMana:
		metrics := character.NewManaMetrics(actionID)
		return func(sim *core.Simulation) {
			if character.HasManaBar() {
				character.AddMana(sim, grant.amount, metrics)
			}
		}
	case clientPowerEnergy:
		metrics := character.NewEnergyMetrics(actionID)
		return func(sim *core.Simulation) {
			if character.HasEnergyBar() {
				character.AddEnergy(sim, grant.amount, metrics)
			}
		}
	case clientPowerRage:
		metrics := character.NewRageMetrics(actionID)
		return func(sim *core.Simulation) {
			if character.HasRageBar() {
				character.AddRage(sim, grant.amount, metrics)
			}
		}
	}
	panic(fmt.Sprintf("item_sets: no model for client power type %d", grant.power))
}
