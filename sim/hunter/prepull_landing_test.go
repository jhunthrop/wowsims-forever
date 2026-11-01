package hunter

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// prepullOnlyRotation casts one spell before the pull at doAt and does
// nothing else, so the spell's whole damage metric is the pre-pull cast.
func prepullOnlyRotation(spellID int32, doAt string) *proto.APLRotation {
	return &proto.APLRotation{
		Type: proto.APLRotation_TypeAPL,
		PrepullActions: []*proto.APLPrepullAction{{
			Action: &proto.APLAction{Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{
				SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: spellID}},
			}}},
			DoAtValue: &proto.APLValue{Value: &proto.APLValue_Const{Const: &proto.APLValueConst{Val: doAt}}},
		}},
	}
}

func spellDamage(result *proto.RaidSimResult, spellID int32) (damage float64, hits int32) {
	for _, action := range result.RaidMetrics.Parties[0].Players[0].Actions {
		if action.Id.GetSpellId() != spellID {
			continue
		}
		for _, target := range action.Targets {
			damage += target.Damage
			hits += target.Hits + target.Crits
		}
	}
	return damage, hits
}

// A damaging cast started before the pull is the pull: a player times it
// to land as the fight starts, and the engine counts no damage before 0.
// Aimed Shot cast at -3.5 s (2 s cast) used to land at about -1.5 s, so
// its damage went unrecorded while it still spent the shared 6 s
// cooldown. The engine now holds a damaging pre-pull cast so it lands at
// 0, and the damage counts.
func TestPrepullAimedShotLandsAtThePullAndCounts(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              core.FullBuffs.Player,
			DistanceFromTarget: 25,
			Rotation:           prepullOnlyRotation(20904, "-3.5s"),
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 30, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{Iterations: 50, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %v", result.Error.Message)
	}
	damage, hits := spellDamage(result, 20904)
	if hits == 0 {
		t.Fatal("the pre-pull Aimed Shot never hit")
	}
	if damage <= 0 {
		t.Errorf("pre-pull Aimed Shot damage = %.0f over %d hits, want > 0: it must land at the pull, not before it", damage, hits)
	}
}
