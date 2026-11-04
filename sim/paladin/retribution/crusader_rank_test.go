package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// A paladin's own Judgement of the Crusader gives the bonus of the rank he
// judges with, at 60 rank 6's 161 Holy damage taken (20303; spellconst
// paladin.json), at 50 rank 5's 127 (20302). The debuff is one aura on the
// target, so every rank must not share rank 1's 23.
func TestOwnCrusaderJudgementUsesTheTopRankBonus(t *testing.T) {
	for _, c := range []struct {
		level int32
		bonus float64
	}{{60, 161}, {50, 127}, {30, 58}} {
		env, _, _ := core.NewEnvironment(
			core.SinglePlayerRaidProto(core.WithSpec(&proto.Player{
				Class: proto.Class_ClassPaladin, Race: proto.Race_RaceHuman, Level: c.level,
				Equipment: &proto.EquipmentSpec{}, TalentsString: Phase45RetTalents,
				Rotation: consecrationRotation,
			}, PlayerOptionsSealofRighteousness), &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			&proto.Encounter{Duration: 30, Targets: []*proto.Target{core.NewDefaultTarget()}},
			false,
		)
		target := env.Encounter.TargetUnits[0]
		aura := target.GetAura("Judgement of the Crusader")
		if aura == nil {
			t.Fatalf("level %d: no Judgement of the Crusader aura on the target", c.level)
		}
		before := target.PseudoStats.SchoolBonusDamageTaken[stats.SchoolIndexHoly]
		aura.Activate(&core.Simulation{Environment: env})
		if got := target.PseudoStats.SchoolBonusDamageTaken[stats.SchoolIndexHoly] - before; got != c.bonus {
			t.Errorf("level %d: Judgement of the Crusader adds %v Holy damage taken, want %v", c.level, got, c.bonus)
		}
	}
}
