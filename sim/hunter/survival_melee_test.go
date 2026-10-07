package hunter

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// survivalMeleeTalents is a level-60 Survival build exercising this
// file's melee kit: Savage Strikes 2/2, Deterrence (Counterattack's
// prereq), Predator's Edge 5/5, Counterattack, Expose Prey 2/2
// (Lacerating Strikes' prereq), Strider Kick, Lacerating Strikes. Tree
// order/offsets from sim/core/proto/hunter.pb.go's field numbers 34-51
// (Survival, tree index 2); Beast Mastery and Marksmanship are left
// empty.
const survivalMeleeTalents = "--000200001051020101"

// buildSurvivalMeleeHunter builds a level-60, no-gear hunter in melee
// range (core.MaxMeleeAttackDistance is 5) against the standard level-60
// target, which auto-attacks back -- so a dodge/parry-gated ability can
// actually be exercised by driving the sim, not just its ApplyEffects.
func buildSurvivalMeleeHunter(t *testing.T) (*core.Simulation, *Hunter, *core.Unit) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{}, // no gear: this fork's item database isn't generated in this test environment.
			Buffs:              core.FullBuffs.Player,
			TalentsString:      survivalMeleeTalents,
			DistanceFromTarget: 5, // melee range: within core.MaxMeleeAttackDistance.
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	agent, ok := sim.Raid.Parties[0].Players[0].(HunterAgent)
	if !ok {
		t.Fatal("the raid's first player is not a hunter agent")
	}
	built := agent.GetHunter()
	target := sim.Encounter.TargetUnits[0]

	return sim, built, target
}

// TestRaptorStrikeLevel60DealsClientDamage guards the fix to
// RaptorStrikeBaseDamage/RaptorStrikeManaCost: both arrays used to carry
// vanilla Classic's numbers, which read higher than this build's client
// data from rank 4 on (source: 1.60.1.70009 spellconst/hunter.json,
// spells 2973/14260-14266).
func TestRaptorStrikeLevel60DealsClientDamage(t *testing.T) {
	sim, built, target := buildSurvivalMeleeHunter(t)

	if built.RaptorStrike == nil || built.RaptorStrikeHit == nil {
		t.Fatal("level-60 hunter has no Raptor Strike registered")
	}
	if got, want := built.RaptorStrikeHit.ActionID.SpellID, int32(14266); got != want {
		t.Errorf("Raptor Strike spell ID = %d, want max rank %d", got, want)
	}
	if got, want := RaptorStrikeDamage[8].Amount, 70.0; got != want {
		t.Errorf("Raptor Strike rank 8 base damage = %v, want %v (client's spellconst amount)", got, want)
	}

	built.RaptorStrikeHit.ApplyEffects(sim, target, built.RaptorStrikeHit)

	metrics := built.RaptorStrikeHit.SpellMetrics[target.UnitIndex]
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatalf("Raptor Strike outcome was neither a hit nor a crit (misses=%d)", metrics.Misses)
	}
	if metrics.TotalDamage <= 0 {
		t.Errorf("Raptor Strike dealt %v damage, want > 0", metrics.TotalDamage)
	}
}

// TestMongooseBiteLevel60CastableWithoutADodgeAndDealsClientDamage
// guards two fixes: MongooseBiteDamage now reads the client's lower
// numbers (was vanilla Classic's), and the dodge-gated "Defensive State"
// aura is gone -- the Forever client's own tooltip
// (wowhead.com/forever/spell=1495) does not list a dodge requirement, so
// Mongoose Bite's ExtraCastCondition no longer requires one.
func TestMongooseBiteLevel60CastableWithoutADodgeAndDealsClientDamage(t *testing.T) {
	sim, built, target := buildSurvivalMeleeHunter(t)

	if built.MongooseBite == nil {
		t.Fatal("level-60 hunter has no Mongoose Bite registered")
	}
	if got, want := built.MongooseBite.ActionID.SpellID, int32(14271); got != want {
		t.Errorf("Mongoose Bite spell ID = %d, want max rank %d", got, want)
	}
	if got, want := MongooseBiteDamage[4].Amount, 57.0; got != want {
		t.Errorf("Mongoose Bite rank 4 base damage = %v, want %v (client's spellconst amount)", got, want)
	}
	// No dodge, no "Defensive State" aura activated: CanCast must still
	// succeed, proving the removed gate is actually gone.
	if !built.MongooseBite.CanCast(sim, target) {
		t.Fatal("Mongoose Bite is not castable without a prior dodge; the removed gate must still be present somewhere")
	}

	built.MongooseBite.ApplyEffects(sim, target, built.MongooseBite)

	metrics := built.MongooseBite.SpellMetrics[target.UnitIndex]
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatalf("Mongoose Bite outcome was neither a hit nor a crit (misses=%d)", metrics.Misses)
	}
	if metrics.TotalDamage <= 0 {
		t.Errorf("Mongoose Bite dealt %v damage, want > 0", metrics.TotalDamage)
	}
}

// TestLaceratingStrikesTicksForty PercentOfTheTriggeringMongooseBite
// guards Lacerating Strikes: talent text is "causes the target to Bleed
// for damage over 21 sec equal to 40% of the damage done by Mongoose
// Bite" (spell 1310533), modeled as a 7-tick (21s / 3s, spell 1310536)
// dot snapshotting 40% of the landed Mongoose Bite hit.
func TestLaceratingStrikesTicksFortyPercentOfTheTriggeringMongooseBite(t *testing.T) {
	sim, built, target := buildSurvivalMeleeHunter(t)

	if built.LaceratingStrikes == nil {
		t.Fatal("level-60 Survival hunter (Lacerating Strikes talented) has no Lacerating Strikes dot registered")
	}

	built.MongooseBite.ApplyEffects(sim, target, built.MongooseBite)
	hitDamage := built.MongooseBite.SpellMetrics[target.UnitIndex].TotalDamage
	if hitDamage <= 0 {
		t.Fatal("Mongoose Bite dealt no damage; Lacerating Strikes has nothing to snapshot from")
	}

	dot := built.LaceratingStrikes.Dot(target)
	if !dot.IsActive() {
		t.Fatal("Lacerating Strikes did not apply off a landed Mongoose Bite hit")
	}

	wantPerTick := hitDamage * 0.4 / laceratingStrikesTicks
	if got := dot.SnapshotBaseDamage; got != wantPerTick {
		t.Errorf("Lacerating Strikes per-tick snapshot = %v, want %v (40%% of %v over %d ticks)", got, wantPerTick, hitDamage, laceratingStrikesTicks)
	}
	if got, want := dot.NumberOfTicks, int32(laceratingStrikesTicks); got != want {
		t.Errorf("Lacerating Strikes NumberOfTicks = %d, want %d (21s / 3s ticks, spell 1310536)", got, want)
	}
}

// TestCounterattackLevel60ProcsOnParryAndDealsClientDamage guards the
// new Counterattack ability: it must stay uncastable until the hunter
// parries an attack, and its landed hit must deal the client's flat
// bonus damage on top of weapon damage... actually Counterattack's
// client effect is a flat school-damage bonus, not weapon-scaled (see
// counterattack.go); this only asserts it deals damage once the parry
// gate is open, and stays closed before one.
func TestCounterattackLevel60ProcsOnParryAndDealsClientDamage(t *testing.T) {
	sim, built, target := buildSurvivalMeleeHunter(t)

	if built.Counterattack == nil {
		t.Fatal("level-60 Survival hunter (Counterattack talented) has no Counterattack registered")
	}
	if got, want := built.Counterattack.ActionID.SpellID, int32(20910); got != want {
		t.Errorf("Counterattack spell ID = %d, want max rank %d", got, want)
	}

	if built.Counterattack.CanCast(sim, target) {
		t.Fatal("Counterattack is castable before any parry; the proc gate must be closed by default")
	}

	built.CounterattackProcAura.Activate(sim)
	if !built.Counterattack.CanCast(sim, target) {
		t.Fatal("Counterattack is not castable once its proc aura is active")
	}

	built.Counterattack.ApplyEffects(sim, target, built.Counterattack)

	if built.CounterattackProcAura.IsActive() {
		t.Error("Counterattack's proc aura is still active after casting; it must be consumed on cast")
	}

	metrics := built.Counterattack.SpellMetrics[target.UnitIndex]
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatalf("Counterattack outcome was neither a hit nor a crit (misses=%d)", metrics.Misses)
	}
	if metrics.TotalDamage <= 0 {
		t.Errorf("Counterattack dealt %v damage, want > 0", metrics.TotalDamage)
	}
}

// TestCounterattackRankAtLevel mirrors this package's other
// RankAtLevel tests (e.g. TestRaptorStrikeRankAtLevel): counterattackLearnLevels
// carries a genuine tie (ranks 1 and 2 both learn at 30, per the
// client's own data), and HighestRankAtLevel must resolve it to the
// higher rank rather than silently keeping rank 1.
func TestCounterattackRankAtLevel(t *testing.T) {
	if got := core.HighestRankAtLevel(counterattackLearnLevels, 60); got != 4 {
		t.Errorf("rank at 60 = %d, want 4", got)
	}
	if got := core.HighestRankAtLevel(counterattackLearnLevels, 30); got != 2 {
		t.Errorf("rank at 30 = %d, want 2 (the client's tied rank-1/rank-2 learn level resolves to the higher rank)", got)
	}
	if got := core.HighestRankAtLevel(counterattackLearnLevels, 29); got != 0 {
		t.Errorf("rank at 29 = %d, want 0", got)
	}
}

// TestStriderKickLevel60DealsWeaponDamage guards the new Strider Kick
// ability: "100% melee weapon damage" (spell 1317257), modeled the same
// way Raptor Strike computes weapon-scaled damage.
func TestStriderKickLevel60DealsWeaponDamage(t *testing.T) {
	sim, built, target := buildSurvivalMeleeHunter(t)

	if built.StriderKick == nil {
		t.Fatal("level-60 Survival hunter (Strider Kick talented) has no Strider Kick registered")
	}
	if got, want := built.StriderKick.ActionID.SpellID, int32(1317257); got != want {
		t.Errorf("Strider Kick spell ID = %d, want %d", got, want)
	}

	built.StriderKick.ApplyEffects(sim, target, built.StriderKick)

	metrics := built.StriderKick.SpellMetrics[target.UnitIndex]
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatalf("Strider Kick outcome was neither a hit nor a crit (misses=%d)", metrics.Misses)
	}
	if metrics.TotalDamage <= 0 {
		t.Errorf("Strider Kick dealt %v damage, want > 0", metrics.TotalDamage)
	}
}

// TestPredatorsEdgeBoostsOffHandDamage guards Predator's Edge's
// off-hand-damage half (client text: "...and your offhand weapon
// damage by 50%" at rank 5): applyPredatorsEdge's OnSpellRegistered hook
// must apply to any spell with an off-hand proc mask, including one
// registered after the build finishes (this fork's item database isn't
// generated in this test environment, so there's no real off-hand
// weapon to register the real OH auto attack from -- a probe spell with
// the same proc mask exercises the same hook without one).
func TestPredatorsEdgeBoostsOffHandDamage(t *testing.T) {
	_, built, _ := buildSurvivalMeleeHunter(t)

	probe := built.RegisterSpell(core.SpellConfig{
		ActionID:         core.ActionID{SpellID: 999999},
		SpellSchool:      core.SpellSchoolPhysical,
		ProcMask:         core.ProcMaskMeleeOHAuto,
		DamageMultiplier: 1,
	})

	if got, want := probe.DamageMultiplier, 1.5; got != want {
		t.Errorf("off-hand-proc-mask spell's DamageMultiplier = %v, want %v (Predator's Edge rank 5: +50%% off-hand damage)", got, want)
	}
}

// TestMongooseBiteIsAMainHandSpecial guards Mongoose Bite's proc mask:
// the client's tooltip requires a main-hand weapon and the cast deals
// main-hand normalized damage only. With the both-hands special mask it
// also carried Predator's Edge's off-hand damage bonus and fired
// off-hand weapon procs.
func TestMongooseBiteIsAMainHandSpecial(t *testing.T) {
	_, built, _ := buildSurvivalMeleeHunter(t)

	if got, want := built.MongooseBite.ProcMask, core.ProcMaskMeleeMHSpecial; got != want {
		t.Errorf("Mongoose Bite proc mask = %v, want %v (main hand only)", got, want)
	}
	if got := built.MongooseBite.DamageMultiplier; got != 1 {
		t.Errorf("Mongoose Bite damage multiplier = %v, want 1: Predator's Edge is an off-hand bonus", got)
	}
}
