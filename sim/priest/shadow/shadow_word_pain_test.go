package shadow

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/priest"
)

// TestShadowWordPainSpellCoefficientMatchesClient guards the
// rotation-accuracy program's engine-numbers fix (2026-09-28,
// audit-priest): the Forever client's spellconst (1.60.1.70009,
// priest.json spells 589/594/970/992/2767/10892/10893/10894, effect 0's
// sp_coefficient) is a flat 0.2 on every one of Shadow Word: Pain's
// eight ranks. This file used to carry vanilla Classic's own
// escalating-then-0.167 table (0.067 at rank 1 up to 0.167 by rank 4),
// a value this build's own client data does not carry for any rank.
func TestShadowWordPainSpellCoefficientMatchesClient(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassPriest,
		Race:          proto.Race_RaceUndead,
		Level:         60,
		Equipment:     &proto.EquipmentSpec{},
		Buffs:         core.FullBuffs.Player,
		TalentsString: P1Talents,
	}, PlayerOptionsBasic)

	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	encounter := &proto.Encounter{
		Duration: 10,
		Targets:  []*proto.Target{core.NewDefaultTarget()},
	}

	env, _, _ := core.NewEnvironment(raid, encounter, true)
	agent := env.Raid.Parties[0].Players[0]
	spriest, ok := agent.(*ShadowPriest)
	if !ok {
		t.Fatalf("player agent is %T, want *ShadowPriest", agent)
	}

	target := env.Encounter.TargetUnits[0]
	for rank := 1; rank <= priest.ShadowWordPainRanks; rank++ {
		spell := spriest.ShadowWordPain[rank]
		if spell == nil {
			continue // not registered below this level's rank floor.
		}
		dot := spell.Dot(target)
		if dot == nil {
			t.Fatalf("Shadow Word: Pain rank %d has no Dot template on the default target", rank)
		}
		if got, want := dot.BonusCoefficient, 0.2; got != want {
			t.Errorf("Shadow Word: Pain rank %d BonusCoefficient = %v, want %v (client's flat sp_coefficient)", rank, got, want)
		}
	}
}
