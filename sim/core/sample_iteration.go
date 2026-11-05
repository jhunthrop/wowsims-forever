package core

import (
	"slices"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// sampleRecorder collects one unit's casts, damage events and aura changes
// for one iteration. It only ever reads the simulation: it rolls nothing and
// changes nothing a spell or an aura reads, so a replay with a recorder is
// the same fight as one without (TestSampleRecorderLeavesTheFightUntouched).
type sampleRecorder struct {
	unit  *Unit
	casts []*proto.SampleCast
	// castActions is parallel to casts: the action behind each row, so a
	// landing that arrives after its cast (a batched white swing, travel
	// time) finds its row. Actions rather than spells, because a white swing
	// deals its damage through a copy of the spell, frozen at the swing.
	castActions []ActionID
	// frames is the stack of casts whose effects are running right now. A
	// proc's cast nests inside the cast that triggered it.
	frames []sampleCastFrame
	auras  []*proto.SampleAuraEvent
}

// sampleCastFrame is the landing of one cast in progress.
type sampleCastFrame struct {
	action  ActionID
	outcome proto.SampleOutcome
	amount  float64
}

// iterationSample is one (dps, seed) pair from the main run. Sixteen
// bytes an iteration is the whole memory cost of knowing which iteration
// was the median one.
type iterationSample struct {
	dps  float64
	seed int64
}

// recordIterationSample appends the just-finished iteration's (dps, seed)
// pair to the run's sample buffer. Both call sites in run() share this body;
// only the seed differs, since iteration 0 and every later iteration derive
// theirs differently (see iterationSeed).
func (sim *Simulation) recordIterationSample(sampleUnit *Unit, seed int64) {
	sim.iterationSamples = append(sim.iterationSamples, iterationSample{
		dps:  sampleUnit.Metrics.dps.LastIterationValue,
		seed: seed,
	})
}

// beginSampleCast opens a frame for the damage results the cast's effects
// deal. Spell.applyEffects calls it just before the effects run.
func (sim *Simulation) beginSampleCast(spell *Spell) {
	if sim.sample == nil || spell.Unit != sim.sample.unit {
		return
	}
	sim.sample.frames = append(sim.sample.frames, sampleCastFrame{action: spell.ActionID})
}

// recordSampleCast appends one cast. Spell.applyEffects calls it after
// the effects have run, so the resources read are the ones the player saw
// after the cast - costs already spent, and any rage or energy the cast
// itself generated already added. The frame beginSampleCast opened carries
// how the cast landed.
func (sim *Simulation) recordSampleCast(spell *Spell, target *Unit) {
	if sim.sample == nil || spell.Unit != sim.sample.unit {
		return
	}

	top := len(sim.sample.frames) - 1
	frame := sim.sample.frames[top]
	sim.sample.frames = sim.sample.frames[:top]

	cast := sim.newSampleRow(spell, target)
	cast.Outcome = frame.outcome
	cast.Amount = frame.amount
	sim.sample.appendRow(spell, cast)
}

// newSampleRow is one row of the log at the current time, for spell.
func (sim *Simulation) newSampleRow(spell *Spell, target *Unit) *proto.SampleCast {
	row := &proto.SampleCast{
		AtMs:      sim.CurrentTime.Milliseconds(),
		ActionId:  spell.ActionID.ToProto(),
		Resources: unitResources(spell.Unit),
	}
	if target != nil {
		row.Target = target.Label
	}
	return row
}

func (recorder *sampleRecorder) appendRow(spell *Spell, row *proto.SampleCast) {
	recorder.casts = append(recorder.casts, row)
	recorder.castActions = append(recorder.castActions, spell.ActionID)
}

// recordSampleDamage notes one damage result. dealDamageInternal calls it
// after the metrics are booked.
//
//   - A periodic result is a row of its own: a tick is an event the player
//     sees, and it has no cast of its own to sit on.
//   - A direct result landing inside its own cast's effects sits on that
//     cast's row. More than one result (a cleave) adds its damage to the
//     first result's row.
//   - A direct result landing later (a batched white swing, travel time)
//     fills the action's latest row that has no landing yet.
//   - Anything else - damage from a spell that was never cast - is a row of
//     its own, so the log's damage still sums to the fight's.
func (sim *Simulation) recordSampleDamage(spell *Spell, isPeriodic bool, result *SpellResult) {
	if sim.sample == nil || spell.Unit != sim.sample.unit {
		return
	}
	recorder := sim.sample
	outcome := sampleOutcomeOf(result, isPeriodic)

	if isPeriodic {
		row := sim.newSampleRow(spell, result.Target)
		row.Outcome = outcome
		row.Amount = result.Damage
		recorder.appendRow(spell, row)
		return
	}

	for i := len(recorder.frames) - 1; i >= 0; i-- {
		frame := &recorder.frames[i]
		if frame.action == spell.ActionID {
			if frame.outcome == proto.SampleOutcome_SampleOutcomeNone {
				frame.outcome = outcome
			}
			frame.amount += result.Damage
			return
		}
	}

	for i := len(recorder.casts) - 1; i >= 0; i-- {
		if recorder.castActions[i] == spell.ActionID && recorder.casts[i].Outcome == proto.SampleOutcome_SampleOutcomeNone {
			recorder.casts[i].Outcome = outcome
			recorder.casts[i].Amount = result.Damage
			return
		}
	}

	row := sim.newSampleRow(spell, result.Target)
	row.Outcome = outcome
	row.Amount = result.Damage
	recorder.appendRow(spell, row)
}

// sampleOutcomeOf names a result the way the engine's own metrics count it
// (dealDamageInternal's order: a blocked crit is a block, not a crit).
func sampleOutcomeOf(result *SpellResult, isPeriodic bool) proto.SampleOutcome {
	switch {
	case result.Outcome.Matches(OutcomeMiss):
		return proto.SampleOutcome_SampleOutcomeMiss
	case result.DidDodge():
		return proto.SampleOutcome_SampleOutcomeDodge
	case result.DidParry():
		return proto.SampleOutcome_SampleOutcomeParry
	case isPeriodic && result.DidCrit():
		return proto.SampleOutcome_SampleOutcomeTickCrit
	case isPeriodic:
		return proto.SampleOutcome_SampleOutcomeTick
	case result.DidBlockCrit():
		return proto.SampleOutcome_SampleOutcomeBlock
	case result.DidCrit():
		return proto.SampleOutcome_SampleOutcomeCrit
	case result.DidGlance():
		return proto.SampleOutcome_SampleOutcomeGlance
	case result.DidBlock():
		return proto.SampleOutcome_SampleOutcomeBlock
	case result.DidCrush():
		return proto.SampleOutcome_SampleOutcomeCrush
	default:
		return proto.SampleOutcome_SampleOutcomeHit
	}
}

// recordSampleAura notes one change to an aura on the sampled player or on
// an enemy. at is passed in because an aura that faded lazily ended at its
// expiry, not at the time the tracker noticed.
func (sim *Simulation) recordSampleAura(aura *Aura, at time.Duration) {
	if sim.sample == nil || aura.ActionID.IsEmptyAction() {
		return
	}
	event := &proto.SampleAuraEvent{
		AtMs:   at.Milliseconds(),
		AuraId: aura.ActionID.ToProto(),
		Active: aura.active,
		Stacks: aura.stacks,
	}
	switch {
	case aura.Unit == sim.sample.unit:
	case aura.Unit.Type == EnemyUnit:
		event.Target = aura.Unit.Label
	default:
		return
	}
	sim.sample.auras = append(sim.sample.auras, event)
}

// unitResources reads the bars the unit actually has. A mage has no rage
// bar and an empty key would read as "0 rage" on the report, which is a
// different claim from "this class has no rage".
func unitResources(unit *Unit) map[string]int32 {
	resources := make(map[string]int32, 4)
	if unit.HasManaBar() {
		resources["mana"] = int32(unit.CurrentMana())
	}
	if unit.HasRageBar() {
		resources["rage"] = int32(unit.CurrentRage())
	}
	if unit.HasEnergyBar() {
		resources["energy"] = int32(unit.CurrentEnergy())
		resources["combo_points"] = unit.ComboPoints()
	}
	return resources
}

// sampleUnit is the unit whose casts the log is of: the first player in
// the raid. Every sim the site sends is a single character, and a raid
// sim's sample log has no single subject to be about.
func (sim *Simulation) sampleUnit() *Unit {
	character := firstPlayerCharacter(sim.Raid)
	if character == nil {
		return nil
	}
	return &character.Unit
}

func firstPlayerCharacter(raid *Raid) *Character {
	for _, party := range raid.Parties {
		for _, player := range party.Players {
			return player.GetCharacter()
		}
	}
	return nil
}

// runSampleIteration replays the median-DPS iteration of a finished run.
//
// It builds a FRESH Simulation from the same request rather than reusing
// the finished one, because runOnce ends in Cleanup, which feeds
// doneIteration, which would add a twenty-first sample to a
// twenty-iteration run's aggregates. A fresh environment costs one
// construction and buys exact aggregates.
func runSampleIteration(rsr *proto.RaidSimRequest, samples []iterationSample, baseDuration time.Duration, signals simsignals.Signals) *proto.SampleIteration {
	if len(samples) == 0 {
		return nil
	}

	ordered := slices.Clone(samples)
	slices.SortStableFunc(ordered, func(a, b iterationSample) int {
		switch {
		case a.dps < b.dps:
			return -1
		case a.dps > b.dps:
			return 1
		default:
			return 0
		}
	})
	median := ordered[len(ordered)/2]

	sim := NewSim(rsr, signals)
	unit := sim.sampleUnit()
	if unit == nil {
		return nil
	}

	// A health fight's duration is estimated from the presim; carry the
	// finished run's estimate over so the replay is the same fight.
	sim.BaseDuration = baseDuration
	sim.Encounter.DurationIsEstimate = false

	sim.sample = &sampleRecorder{unit: unit}
	sim.reseedTo(median.seed)
	sim.runOnce()

	return &proto.SampleIteration{
		Dps:             median.dps,
		DurationSeconds: sim.Duration.Seconds(),
		Casts:           sim.sample.casts,
		Auras:           sim.sample.auras,
		Sheet:           sampleSheet(rsr),
	}
}

// sampleSheetDebuffSettle is how far the sheet lets the request's debuffs
// land. They are scheduled, not instant: Sunder Armor takes 0.8 s to reach
// five stacks, a raid Expose Armor lands at 3 s (SchedulePeriodicDebuffApplication).
const sampleSheetDebuffSettle = 5 * time.Second

// sampleSheet reads the character and the target as the fight starts. It
// builds an environment of its own, so nothing it activates can reach the
// replay. The player's half is the same build-phase measurement the engine's
// ComputeStats reports (Character.FillPlayerStats). The target's armor then
// comes from letting that environment's clock run with nobody pulled: no
// swing, no rotation, only the debuffs the request schedules.
func sampleSheet(rsr *proto.RaidSimRequest) *proto.SampleSheet {
	env, _, _ := NewEnvironment(rsr.Raid, rsr.Encounter, false)
	character := firstPlayerCharacter(env.Raid)
	if character == nil || len(env.Encounter.TargetUnits) == 0 {
		return nil
	}
	target := env.Encounter.TargetUnits[0]

	character.applyBuildPhaseAuras(CharacterBuildPhaseAll)
	sheet := readSheet(character, target)
	character.clearBuildPhaseAuras(CharacterBuildPhaseAll)

	sheet.TargetLevel = target.Level
	sheet.TargetArmor = target.Armor()
	sim := newSimWithEnv(env, &proto.SimOptions{Iterations: 1, RandomSeed: rsr.SimOptions.GetRandomSeed()}, simsignals.CreateSignals())
	sim.reset()
	for sim.CurrentTime < sampleSheetDebuffSettle {
		if finished := sim.Step(); finished {
			break
		}
	}
	sheet.TargetArmorAfterDebuffs = target.Armor()
	return sheet
}

func readSheet(character *Character, target *Unit) *proto.SampleSheet {
	unit := &character.Unit
	return &proto.SampleSheet{
		AttackPower:          unit.GetStat(stats.AttackPower),
		RangedAttackPower:    unit.GetStat(stats.RangedAttackPower),
		MeleeHitPct:          unit.GetStat(stats.Hit) / HitRatingPerHitChance,
		MeleeCritPctVsTarget: unit.meleeCritPctVs(target),
		MeleeHastePct:        (unit.SwingSpeed() - 1) * 100,
		MainHandSkill:        unit.weaponSkillWith(character.MainHand()),
		OffHandSkill:         unit.weaponSkillWith(character.OffHand()),
		RangedSkill:          unit.weaponSkillWith(character.Ranged()),
		SpellPower:           unit.GetStat(stats.SpellPower) + unit.GetStat(stats.SpellDamage),
		SpellHitPct:          unit.GetStat(stats.Hit) / HitRatingPerHitChance,
		SpellCritPct:         unit.spellTableCritRating() / CritRatingPerCritChance,
		HasManaBar:           unit.HasManaBar(),
	}
}

// meleeCritPctVs is PhysicalCritChance's own arithmetic for a white swing
// on this target's table, in percent points.
func (unit *Unit) meleeCritPctVs(target *Unit) float64 {
	table := unit.AttackTables[target.UnitIndex][proto.CastType_CastTypeMainHand]
	return unit.meleeTableCritRating()/CritRatingPerCritChance - table.MeleeCritSuppression*100
}

// weaponSkillWith is the skill NewAttackTable rolls this weapon with, or zero
// with nothing in the slot.
func (unit *Unit) weaponSkillWith(weapon *Item) float64 {
	if weapon == nil || weapon.ID == 0 {
		return 0
	}
	return float64(unit.Level*5) + GetWeaponSkill(unit, weapon)
}
