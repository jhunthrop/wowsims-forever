package mage

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// newArcaneBlastTestMage builds a level-60 mage with the Frost reference
// build plus one point in Arcane Blast, prepulled and ready to drive
// ApplyEffects directly, for every test in this file.
func newArcaneBlastTestMage(t *testing.T) (*core.Simulation, *Mage) {
	t.Helper()
	// ForeverFrostTalents (talents.go) already spends its one Arcane
	// point in Missile Barrage; this adds Arcane Blast's own point,
	// which the Frost reference build does not take.
	talentsStr := talentStringWithRank(t, ForeverFrostTalents, "arcane_blast", 1)

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassMage,
			Race:               proto.Race_RaceTroll,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{}, // no gear: this fork's item database isn't generated in this test environment (see pet_level_test.go).
			Buffs:              core.FullBuffs.Player,
			TalentsString:      talentsStr,
			DistanceFromTarget: 5,
		},
		PlayerOptions,
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

	agent, ok := sim.Raid.Parties[0].Players[0].(MageAgent)
	if !ok {
		t.Fatal("the raid's first player is not a mage agent")
	}
	return sim, agent.GetMage()
}

// TestArcaneBlastCostsFifteenPercentOfBaseManaAndRampsPerStack pins the
// client's price for Arcane Blast (SpellPower.PowerCostPct 15 on every
// player rank in build 1.60.1.70009; the flat cost column is 0) and the
// buff's own "mana cost of Arcane Blast is increased by 175%" per stack
// (spell 400573, effect 1). Before 2026-10-07 the spell registered with
// no cost at all, so an Arcane mage spammed it for free.
func TestArcaneBlastCostsFifteenPercentOfBaseManaAndRampsPerStack(t *testing.T) {
	sim, built := newArcaneBlastTestMage(t)
	arcaneBlast := built.ArcaneBlast[ArcaneBlastRanks]
	if arcaneBlast == nil || arcaneBlast.Cost == nil {
		t.Fatal("top-rank Arcane Blast registered with no mana cost")
	}
	if got, want := arcaneBlast.Cost.BaseCost, built.BaseMana*arcaneBlastManaCostPct/100; got != want {
		t.Errorf("Arcane Blast BaseCost = %v, want %v (15%% of base mana %v)", got, want, built.BaseMana)
	}
	if got, want := arcaneBlast.Cost.Multiplier, int32(100); got != want {
		t.Errorf("Arcane Blast Cost.Multiplier before any cast = %d, want %d", got, want)
	}

	target := sim.Encounter.TargetUnits[0]
	for cast := int32(1); cast <= arcaneBlastMaxStacks; cast++ {
		arcaneBlast.ApplyEffects(sim, target, arcaneBlast)
		if got, want := arcaneBlast.Cost.Multiplier, 100+arcaneBlastCostIncreasePctPerStack*cast; got != want {
			t.Errorf("Arcane Blast Cost.Multiplier after %d cast(s) = %d, want %d", cast, got, want)
		}
	}
	// A fifth cast stays at the buff's stack cap.
	arcaneBlast.ApplyEffects(sim, target, arcaneBlast)
	if got, want := arcaneBlast.Cost.Multiplier, int32(100+arcaneBlastCostIncreasePctPerStack*arcaneBlastMaxStacks); got != want {
		t.Errorf("Arcane Blast Cost.Multiplier at the stack cap = %d, want %d", got, want)
	}

	// Letting the buff drop (any other damage spell, or its 8 s) resets
	// the ramp with the stacks.
	built.ArcaneBlastAura.Deactivate(sim)
	if got, want := arcaneBlast.Cost.Multiplier, int32(100); got != want {
		t.Errorf("Arcane Blast Cost.Multiplier after the buff dropped = %d, want %d", got, want)
	}
}

// TestArcaneBlastAndMissileBarrageCastAndDealDamage rebuilds the case
// talents.go used to leave stubbed: mage.Talents.ArcaneBlast and
// mage.Talents.MissileBarrage were both read and discarded
// (`_ = mage.Talents.ArcaneBlast`), so no mage - however talented -
// could ever cast Arcane Blast, and Missile Barrage never affected
// Arcane Missiles. A level-60 mage with both talents must have Arcane
// Blast registered at its max rank and dealing damage and stacking its
// own buff, and Missile Barrage must be able to make the next Arcane
// Missiles cast free and double-speed.
func TestArcaneBlastAndMissileBarrageCastAndDealDamage(t *testing.T) {
	sim, built := newArcaneBlastTestMage(t)

	if !built.Talents.MissileBarrage {
		t.Fatal("ForeverFrostTalents is expected to spend its one Arcane point in Missile Barrage")
	}
	if built.MissileBarrageAura == nil {
		t.Fatal("Missile Barrage talent taken but registerMissileBarrage registered no aura")
	}
	if len(built.ArcaneBlast) <= ArcaneBlastRanks || built.ArcaneBlast[ArcaneBlastRanks] == nil {
		t.Fatalf("level-60 mage with Arcane Blast talent has no rank %d registered", ArcaneBlastRanks)
	}
	if got, want := built.ArcaneBlast[ArcaneBlastRanks].ActionID.SpellID, ArcaneBlastSpellId[ArcaneBlastRanks]; got != want {
		t.Errorf("Arcane Blast top rank SpellID = %d, want %d", got, want)
	}

	target := sim.Encounter.TargetUnits[0]

	// Drive the spell's own ApplyEffects directly (the real code a Cast
	// would run), the way aimed_shot_test.go does, rather than going
	// through Cast so the assertion isn't gated on GCD/mana bookkeeping.
	arcaneBlast := built.ArcaneBlast[ArcaneBlastRanks]
	arcaneBlast.ApplyEffects(sim, target, arcaneBlast)
	waitForOutcomes(sim)

	metrics := arcaneBlast.SpellMetrics[target.UnitIndex]
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatalf("Arcane Blast outcome was neither a hit nor a crit (misses=%d)", metrics.Misses)
	}
	if metrics.TotalDamage <= 0 {
		t.Errorf("Arcane Blast dealt %v damage, want > 0", metrics.TotalDamage)
	}
	if got, want := built.ArcaneBlastAura.GetStacks(), int32(1); got != want {
		t.Errorf("Arcane Blast stacks after one cast = %d, want %d", got, want)
	}

	// Missile Barrage: force the proc (rather than rely on its 40%
	// chance on Arcane Blast) by activating the buff directly, then
	// confirm arcane_missiles.go's ApplyEffects reads it - cost zeroed
	// and ticking twice as fast.
	arcaneMissiles := built.ArcaneMissiles[ArcaneMissilesRanks]
	if arcaneMissiles == nil {
		t.Fatal("level-60 mage has no top-rank Arcane Missiles registered")
	}
	if arcaneMissiles.Cost == nil {
		t.Fatal("Arcane Missiles has no mana cost to zero")
	}

	built.MissileBarrageAura.Activate(sim)
	if got, want := arcaneMissiles.Cost.Multiplier, int32(0); got != want {
		t.Errorf("Arcane Missiles Cost.Multiplier while Missile Barrage is up = %d, want %d", got, want)
	}
	// missile_barrage.go's OnCastComplete guard (mirroring Clearcasting's
	// own) only skips deactivation on the exact event that just granted
	// the buff; advance sim time first so this cast reads as a later,
	// separate one - as it always is outside a test driving both calls
	// back to back with zero elapsed time.
	waitForOutcomes(sim)

	arcaneMissiles.ApplyEffects(sim, target, arcaneMissiles)
	if got, want := arcaneMissiles.Dot(target).TickLength, missileBarrageTickLength; got != want {
		t.Errorf("Arcane Missiles tick length under Missile Barrage = %s, want %s", got, want)
	}

	// The real Cast() pipeline invokes this after ApplyEffects returns
	// (sim/core/cast.go); calling it directly here is what actually
	// consumes the buff, the same way Clearcasting consumes itself.
	arcaneMissiles.Unit.OnCastComplete(sim, arcaneMissiles)
	if built.MissileBarrageAura.IsActive() {
		t.Error("Missile Barrage did not consume itself after the Arcane Missiles cast it empowered")
	}
	if got, want := arcaneMissiles.Cost.Multiplier, int32(100); got != want {
		t.Errorf("Arcane Missiles Cost.Multiplier after Missile Barrage is consumed = %d, want %d", got, want)
	}
}
