package shadow

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get caster sets included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
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

// Shadow Weaving "can no longer fail to apply" (Blizzard's 1 October 2026
// notes): every Shadow spell that lands adds a stack, at any rank - the
// pre-hotfix engine rolled the rank's proc chance and a resist on top.
func TestShadowWeavingAlwaysApplies(t *testing.T) {
	sim, built, target := newShadowPriestSimWithTalents(t, map[string]int{"shadow_weaving": 1})

	for i := 1; i <= 5; i++ {
		built.AddShadowWeavingStack(sim, target)
		if got := built.ShadowWeavingAuras.Get(target).GetStacks(); got != int32(i) {
			t.Fatalf("after %d applications the target has %d Shadow Weaving stacks, want %d", i, got, i)
		}
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
