package elemental

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/shaman"
)

// callOfFlameTalentsString sets only Call of Flame (the Elemental tree's
// 5th tracked node - tier 1, column 2, in talents/shaman.json - to the
// given rank), the same "isolate the one talent this lane added"
// convention as LavaBurstOnlyTalentsString above.
func callOfFlameTalentsString(rank int) string {
	nodes := [16]byte{}
	for i := range nodes {
		nodes[i] = '0'
	}
	nodes[4] = byte('0' + rank)
	return string(nodes[:]) + "--"
}

// newCallOfFlameShaman is newLavaBurstShaman's shape, with an arbitrary
// talents string instead of the Lava Burst-only one, so this file can
// vary Call of Flame's rank independently.
func newCallOfFlameShaman(t *testing.T, talentsString string) (*core.Simulation, *shaman.Shaman) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassShaman,
			Race:          proto.Race_RaceTroll,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			Buffs:         core.FullBuffs.Player,
			TalentsString: talentsString,
		},
		&proto.Player_ElementalShaman{
			ElementalShaman: &proto.ElementalShaman{
				Options: &proto.ElementalShaman_Options{},
			},
		},
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

	agent, ok := sim.Raid.Parties[0].Players[0].(shaman.ShamanAgent)
	if !ok {
		t.Fatal("the raid's first player is not a shaman agent")
	}
	return sim, agent.GetShaman()
}

// TestFlameShockAppliesCallOfFlameMultiplier guards
// sim/shaman/flame_shock.go against regressing to
// newShockSpellConfig's own flat DamageMultiplier: 1: Call of Flame's
// tooltip (talents/shaman.json node 104770) names "Fire Totems and ...
// Flame Shock, Fire Nova, and Lava Burst" explicitly, and Lava Burst
// and the Fire Totems already apply this 5%-per-rank multiplier via
// shaman.callOfFlameMultiplier() (lava_burst.go, fire_totems.go) -
// Flame Shock was the one left out.
func TestFlameShockAppliesCallOfFlameMultiplier(t *testing.T) {
	cases := []struct {
		ranks int
		want  float64
	}{
		{0, 1.00},
		{1, 1.05},
		{2, 1.10},
		{3, 1.15},
	}
	for _, c := range cases {
		_, built := newCallOfFlameShaman(t, callOfFlameTalentsString(c.ranks))
		if got, want := built.Talents.CallOfFlame, int32(c.ranks); got != want {
			t.Fatalf("callOfFlameTalentsString(%d) gave Talents.CallOfFlame = %d, want %d", c.ranks, got, want)
		}
		maxRank := built.FlameShock[len(built.FlameShock)-1]
		if got := maxRank.DamageMultiplier; got != c.want {
			t.Errorf("Call of Flame %d/3: Flame Shock DamageMultiplier = %.2f, want %.2f", c.ranks, got, c.want)
		}
	}
}

// TestFlameShockDealsCallOfFlameBonusDamage confirms the multiplier
// reaches actual dealt damage for both the direct hit and the DoT tick,
// not just the registered SpellConfig field: two sims share the same
// RandomSeed, so the only difference in outcome is Call of Flame's own
// multiplier (the same isolation TestLavaBurstFlameShockInteraction
// uses above).
func TestFlameShockDealsCallOfFlameBonusDamage(t *testing.T) {
	simWithout, withoutShaman := newCallOfFlameShaman(t, callOfFlameTalentsString(0))
	targetWithout := simWithout.Encounter.TargetUnits[0]
	flameShockWithout := withoutShaman.FlameShock[len(withoutShaman.FlameShock)-1]
	flameShockWithout.ApplyEffects(simWithout, targetWithout, flameShockWithout)
	baseDamage := flameShockWithout.SpellMetrics[targetWithout.UnitIndex].TotalDamage
	if baseDamage <= 0 {
		t.Fatalf("Flame Shock without Call of Flame dealt %v initial damage, want > 0", baseDamage)
	}
	dotWithout := flameShockWithout.Dot(targetWithout)
	if !dotWithout.IsActive() {
		t.Fatal("Flame Shock without Call of Flame did not apply its DoT")
	}
	baseTick := dotWithout.SnapshotBaseDamage * dotWithout.SnapshotAttackerMultiplier

	simWith, withShaman := newCallOfFlameShaman(t, callOfFlameTalentsString(3))
	targetWith := simWith.Encounter.TargetUnits[0]
	flameShockWith := withShaman.FlameShock[len(withShaman.FlameShock)-1]
	flameShockWith.ApplyEffects(simWith, targetWith, flameShockWith)
	bonusDamage := flameShockWith.SpellMetrics[targetWith.UnitIndex].TotalDamage

	wantBonusDamage := baseDamage * 1.15
	if diff := bonusDamage - wantBonusDamage; diff > 0.01 || diff < -0.01 {
		t.Errorf("Flame Shock initial hit with Call of Flame 3/3 dealt %v damage, want %v (+15%% of %v)", bonusDamage, wantBonusDamage, baseDamage)
	}

	dotWith := flameShockWith.Dot(targetWith)
	if !dotWith.IsActive() {
		t.Fatal("Flame Shock with Call of Flame did not apply its DoT")
	}
	bonusTick := dotWith.SnapshotBaseDamage * dotWith.SnapshotAttackerMultiplier
	wantBonusTick := baseTick * 1.15
	if diff := bonusTick - wantBonusTick; diff > 0.01 || diff < -0.01 {
		t.Errorf("Flame Shock DoT tick with Call of Flame 3/3 snapshots %v damage, want %v (+15%% of %v)", bonusTick, wantBonusTick, baseTick)
	}
}
