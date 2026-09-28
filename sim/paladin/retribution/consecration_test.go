package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// consecrationSpellID is spell 20924, Consecration rank 5 (learned at 60;
// spellconst/paladin.json).
const consecrationSpellID = 20924

// consecrationRotation casts Consecration on cooldown -- the smallest
// rotation that exercises the spell's registration and its periodic AOE
// damage.
var consecrationRotation = &proto.APLRotation{
	Type: proto.APLRotation_TypeAPL,
	PriorityList: []*proto.APLListItem{
		{
			Action: &proto.APLAction{
				Action: &proto.APLAction_CastSpell{
					CastSpell: &proto.APLActionCastSpell{
						SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: consecrationSpellID}},
					},
				},
			},
		},
	},
}

// TestConsecrationTicks is engine-fixes brief 1's test for item 2:
// Consecration is baseline in Forever (re-enabled in
// sim/paladin/consecration.go, no talent gate), so a level 60 paladin
// casting it should see its periodic AOE damage actually tick.
func TestConsecrationTicks(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassPaladin,
		Race:               proto.Race_RaceHuman,
		Level:              60,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              core.FullBuffs.Player,
		TalentsString:      Phase45RetTalents,
		Rotation:           consecrationRotation,
		DistanceFromTarget: 5,
	}, PlayerOptionsSealofRighteousness)

	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 30,
			Targets:  []*proto.Target{core.NewDefaultTarget()},
		},
		SimOptions: &proto.SimOptions{
			Iterations: 1,
			RandomSeed: 1,
			IsTest:     true,
		},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}

	metrics := findActionMetrics(result, consecrationSpellID)
	if metrics == nil {
		t.Fatalf("Consecration (spell %d) never appears in the action metrics -- it did not cast", consecrationSpellID)
	}

	var casts, ticks int32
	var damage float64
	for _, target := range metrics.Targets {
		casts += target.Casts
		ticks += target.Ticks
		damage += target.Damage
	}

	if casts == 0 {
		t.Errorf("Consecration casts = 0, want > 0")
	}
	if ticks == 0 {
		t.Errorf("Consecration ticks = 0, want > 0 (periodic AOE damage never landed)")
	}
	if damage <= 0 {
		t.Errorf("Consecration damage = %v, want > 0", damage)
	}
}

func findActionMetrics(result *proto.RaidSimResult, spellID int32) *proto.ActionMetrics {
	player := result.RaidMetrics.Parties[0].Players[0]
	for _, action := range player.Actions {
		if action.Id.GetSpellId() == spellID {
			return action
		}
	}
	return nil
}
