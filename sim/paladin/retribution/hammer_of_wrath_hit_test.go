package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Precision (20189) carries aura 54 (melee and ranged hit, 698111) and,
// new in Forever, aura 55 (spell hit, 1324040), with no equipped-item
// restriction (1.60.1.70707); Wowhead Forever: "Improves your chance to hit
// by 1%". Hammer of Wrath (24239, a ranged attack) keeps it: no negative
// bonus hit takes Precision back.
func TestHammerOfWrathKeepsPrecisionsHit(t *testing.T) {
	player := utilitySealPaladin(60, `{"type": "TypeAPL"}`)
	player.TalentsString = talentString(t, map[string]int{"precision": 3})
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	unit := &sim.Raid.Parties[0].Players[0].GetCharacter().Unit
	hammer := unit.GetSpell(core.ActionID{SpellID: 24239})
	if hammer == nil {
		t.Fatal("Hammer of Wrath 24239 is not registered at 60")
	}
	if got := hammer.BonusHitRating; got != 0 {
		t.Errorf("Hammer of Wrath's bonus hit with Precision 3/3 is %v, want 0", got)
	}
}
