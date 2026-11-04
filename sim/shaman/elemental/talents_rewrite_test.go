package elemental

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/shaman"
)

// elementalTalentString sets exactly one of the Elemental tree's 16
// nodes to the given rank, the same "isolate the one talent this lane
// added" convention flame_shock_call_of_flame_test.go's
// callOfFlameTalentsString uses - node indices come from the generated
// proto.ShamanTalents's field order by name (elementalNode), which is
// the live tree's tier-then-column order since the proto was regenerated
// from build 1.60.1.70009.
func elementalTalentString(index, rank int) string {
	nodes := [16]byte{}
	for i := range nodes {
		nodes[i] = '0'
	}
	nodes[index] = byte('0' + rank)
	return string(nodes[:]) + "--"
}

var (
	nodeEyeOfTheStorm     = elementalNode("eye_of_the_storm")
	nodeLightningOverload = elementalNode("lightning_overload")
	nodeElementalAlacrity = elementalNode("elemental_alacrity")
)

func newIsolatedTalentShaman(t *testing.T, talentsString string) (*core.Simulation, *shaman.Shaman) {
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
			// Long enough that TestLightningOverloadExtraHits's 300
			// direct ApplyEffects calls, each flushed with a few
			// sim.Step()s to land its travel-delayed damage, cannot run
			// the clock past end of combat and have some of those hits
			// silently stop counting.
			Duration: 36000,
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

// TestElementalAlacrityCastTime guards shaman.applyElementalAlacrity:
// node 104771, three ranks (trait curve 85216: -170/-330/-500), flat
// 0.17/0.33/0.5 sec off Lightning Bolt's and Chain Lightning's cast time.
func TestElementalAlacrityCastTime(t *testing.T) {
	cases := []struct {
		rank      int
		reduction time.Duration
	}{
		{1, 170 * time.Millisecond},
		{2, 330 * time.Millisecond},
		{3, 500 * time.Millisecond},
	}

	_, baseline := newIsolatedTalentShaman(t, elementalTalentString(nodeElementalAlacrity, 0))
	baseCastTime := baseline.LightningBolt[len(baseline.LightningBolt)-1].DefaultCast.CastTime
	baseChainCastTime := baseline.ChainLightning[len(baseline.ChainLightning)-1].DefaultCast.CastTime

	for _, c := range cases {
		_, built := newIsolatedTalentShaman(t, elementalTalentString(nodeElementalAlacrity, c.rank))
		if got, want := built.Talents.ElementalAlacrity, int32(c.rank); got != want {
			t.Fatalf("elementalTalentString(nodeElementalAlacrity, %d) gave rank %d, want %d", c.rank, got, want)
		}
		maxRankLB := built.LightningBolt[len(built.LightningBolt)-1]
		if got, want := maxRankLB.DefaultCast.CastTime, baseCastTime-c.reduction; got != want {
			t.Errorf("Elemental Alacrity %d/3: Lightning Bolt cast time = %v, want %v", c.rank, got, want)
		}
		maxRankCL := built.ChainLightning[len(built.ChainLightning)-1]
		if got, want := maxRankCL.DefaultCast.CastTime, baseChainCastTime-c.reduction; got != want {
			t.Errorf("Elemental Alacrity %d/3: Chain Lightning cast time = %v, want %v", c.rank, got, want)
		}
	}
}

// TestEyeOfTheStormPushbackReduction guards shaman.applyEyeOfTheStorm:
// node 104763, three ranks (trait curve 85224), 23/47/70% pushback
// reduction on Lightning Bolt and Chain Lightning.
func TestEyeOfTheStormPushbackReduction(t *testing.T) {
	cases := []struct {
		rank int
		want float64
	}{
		{0, 0},
		{1, 0.23},
		{2, 0.47},
		{3, 0.70},
	}
	for _, c := range cases {
		_, built := newIsolatedTalentShaman(t, elementalTalentString(nodeEyeOfTheStorm, c.rank))
		maxRankLB := built.LightningBolt[len(built.LightningBolt)-1]
		if got := maxRankLB.PushbackReduction; got != c.want {
			t.Errorf("Eye of the Storm %d/3: Lightning Bolt PushbackReduction = %v, want %v", c.rank, got, c.want)
		}
		maxRankCL := built.ChainLightning[len(built.ChainLightning)-1]
		if got := maxRankCL.PushbackReduction; got != c.want {
			t.Errorf("Eye of the Storm %d/3: Chain Lightning PushbackReduction = %v, want %v", c.rank, got, c.want)
		}
	}
}

// flushTravelTime drains the sim's pending actions so every
// WaitTravelTime-deferred DealDamage (Lightning Bolt's own missile
// travel, MissileSpeed 20) lands and updates SpellMetrics before the
// test reads them; ApplyEffects called directly (bypassing Cast(), the
// same as this package's other direct-ApplyEffects tests) never
// advances sim time on its own. Draining all at once, after every cast
// has already been issued, rather than a few steps between each cast,
// avoids a growing backlog of not-yet-processed travel actions outrunning
// a small fixed per-cast step budget.
func flushTravelTime(sim *core.Simulation) {
	for i := 0; i < 5000; i++ {
		if sim.Step() {
			return
		}
	}
}

// TestLightningOverloadExtraDamage guards shaman.rollLightningOverload
// and lightning_bolt.go's use of it: node 104759, three ranks, a 10/20/
// 30% chance per Lightning Bolt cast (1.60.1.70755) to deal a second hit,
// the rank's own overload row (408477: 98 at 0.357 against the bolt's 353
// at 0.714), through the same spell object. Base miss and resist chance
// alone (no gear, no +hit) already keeps Hits below the cast count without
// this talent, so landed-hit count isn't a usable signal here; total
// damage across many casts is - at 30% per cast and a ~28% hit, roughly
// an extra 8% of damage over the same number of casts, and over 300 casts
// that is far outside noise.
func TestLightningOverloadExtraDamage(t *testing.T) {
	const casts = 300

	simWithout, without := newIsolatedTalentShaman(t, elementalTalentString(nodeLightningOverload, 0))
	targetWithout := simWithout.Encounter.TargetUnits[0]
	lbWithout := without.LightningBolt[len(without.LightningBolt)-1]
	for i := 0; i < casts; i++ {
		lbWithout.ApplyEffects(simWithout, targetWithout, lbWithout)
	}
	flushTravelTime(simWithout)
	damageWithout := lbWithout.SpellMetrics[targetWithout.UnitIndex].TotalDamage

	simWith, with := newIsolatedTalentShaman(t, elementalTalentString(nodeLightningOverload, 3))
	if got, want := with.Talents.LightningOverload, int32(3); got != want {
		t.Fatalf("elementalTalentString(nodeLightningOverload, 3) gave rank %d, want %d", got, want)
	}
	targetWith := simWith.Encounter.TargetUnits[0]
	lbWith := with.LightningBolt[len(with.LightningBolt)-1]
	for i := 0; i < casts; i++ {
		lbWith.ApplyEffects(simWith, targetWith, lbWith)
	}
	flushTravelTime(simWith)
	damageWith := lbWith.SpellMetrics[targetWith.UnitIndex].TotalDamage

	if damageWith <= damageWithout {
		t.Fatalf("Lightning Overload 3/3 (30%% per cast): %v total damage across %d casts, want more than the no-talent total %v", damageWith, casts, damageWithout)
	}
	// Expected uplift is close to 30%*98/353 = 8.3%; require at least
	// half of that (4%) so a weak or miswired proc doesn't pass by
	// accident, without being so tight that ordinary hit/crit variance
	// fails it - and at most 12%, so the old half-the-bolt overload (15%)
	// fails.
	if minWant := damageWithout * 1.04; damageWith < minWant {
		t.Errorf("Lightning Overload 3/3: total damage %v, want at least %v (>=4%% over the no-talent %v)", damageWith, minWant, damageWithout)
	}
	if maxWant := damageWithout * 1.12; damageWith > maxWant {
		t.Errorf("Lightning Overload 3/3: total damage %v, want at most %v (<=12%% over the no-talent %v)", damageWith, maxWant, damageWithout)
	}
}
