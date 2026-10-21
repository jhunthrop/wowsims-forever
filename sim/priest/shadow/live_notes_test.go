package shadow

import (
	"math"
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get caster sets included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/priest"
)

// newShadowPriestSimWithTalents stands up a level-60 Undead shadow priest
// in a real simulation (so spells can be cast and auras activated) with
// exactly the named talents spent, by generated proto field name.
func newShadowPriestSimWithTalents(t *testing.T, ranks map[string]int) (*core.Simulation, *ShadowPriest, *core.Unit) {
	t.Helper()

	talents, err := core.TalentsStringFromRanks((&proto.PriestTalents{}).ProtoReflect(), priest.TalentTreeSizes, ranks)
	if err != nil {
		t.Fatal(err)
	}
	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassPriest,
		Race:          proto.Race_RaceUndead,
		Level:         60,
		Equipment:     &proto.EquipmentSpec{},
		Buffs:         core.FullBuffs.Player,
		TalentsString: talents,
	}, PlayerOptionsBasic)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 3600,
			Targets:  []*proto.Target{core.NewDefaultTarget()},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	built, ok := sim.Raid.Parties[0].Players[0].(*ShadowPriest)
	if !ok {
		t.Fatal("the raid's first player is not a *ShadowPriest")
	}
	return sim, built, sim.Encounter.TargetUnits[0]
}

// Devouring Plague can critically strike: Blizzard's 1 October 2026
// notes ("Devouring Plague can crit"), enabled per dot through
// DotConfig.CanCrit.
func TestDevouringPlagueTicksCanCrit(t *testing.T) {
	_, built, target := newShadowPriestSimWithTalents(t, nil)

	topRank := built.DevouringPlague[priest.DevouringPlagueRanks]
	if topRank == nil {
		t.Fatalf("DevouringPlague[%d] not registered at level 60", priest.DevouringPlagueRanks)
	}
	if !topRank.Dot(target).CanCrit {
		t.Error("Devouring Plague's dot does not roll periodic critical strikes")
	}
}

// Forever's Shadow Weaving is a self buff (live talent text, build
// 1.60.1.70009 with the Wowhead overlay): "increase the Shadow damage you
// deal by 2% for 15 sec, stacking up to 5 times", at a 100% chance with
// three points, and Blizzard's 1 October 2026 note removed the proc's own
// hit roll. Every landed Shadow spell therefore adds one stack on the
// priest, each worth 2% of the priest's own Shadow damage, and the target
// carries nothing.
func TestShadowWeavingStacksOnThePriestAtTwoPercent(t *testing.T) {
	sim, built, target := newShadowPriestSimWithTalents(t, map[string]int{"shadow_weaving": 3})

	base := built.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow]
	// The harness's raid debuffs (Curse of Shadow) already sit on the
	// target; the stacks must not add to them.
	targetBefore := target.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexShadow]
	for i := 1; i <= 5; i++ {
		built.AddShadowWeavingStack(sim, target)
		if got := built.ShadowWeavingAura.GetStacks(); got != int32(i) {
			t.Fatalf("after %d applications the priest has %d Shadow Weaving stacks, want %d", i, got, i)
		}
		want := base * (1 + 0.02*float64(i))
		if got := built.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow]; math.Abs(got-want) > 1e-9 {
			t.Fatalf("after %d stacks the priest's Shadow damage multiplier is %v, want %v (2%% a stack)", i, got, want)
		}
	}
	// A sixth application stays at the cap.
	built.AddShadowWeavingStack(sim, target)
	if got := built.ShadowWeavingAura.GetStacks(); got != 5 {
		t.Fatalf("Shadow Weaving stacks past its cap: %d", got)
	}
	if got := target.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexShadow]; got != targetBefore {
		t.Errorf("the target's Shadow damage taken multiplier moved from %v to %v; Forever's Shadow Weaving is the priest's own buff", targetBefore, got)
	}
}

// Inner Focus's critical bonus no longer applies to periodic effects
// (Blizzard's 1 October 2026 notes): the 25% reaches a direct spell
// and stays off Devouring Plague, Shadow Word: Pain and channels.
func TestInnerFocusCritBonusSkipsPeriodicSpells(t *testing.T) {
	sim, built, _ := newShadowPriestSimWithTalents(t, map[string]int{"inner_focus": 1})

	direct := built.MindBlast[len(built.MindBlast)-1]
	periodic := built.DevouringPlague[priest.DevouringPlagueRanks]
	directBefore, periodicBefore := direct.BonusCritRating, periodic.BonusCritRating

	built.InnerFocusAura.Activate(sim)

	if got, want := direct.BonusCritRating-directBefore, 25*float64(core.CritRatingPerCritChance); got != want {
		t.Errorf("Inner Focus raised Mind Blast's crit rating by %v, want %v", got, want)
	}
	if got := periodic.BonusCritRating - periodicBefore; got != 0 {
		t.Errorf("Inner Focus raised Devouring Plague's crit rating by %v, want 0 (periodic)", got)
	}
}
