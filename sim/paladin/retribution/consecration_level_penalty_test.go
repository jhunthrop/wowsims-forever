package retribution

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Consecration's first-targets spell-power share (0.042 a tick) takes
// Forever's level-gap rule: a rank keeps its whole coefficient until the
// caster is 17 levels past the rank's learn level, then loses 5% a level
// (core.LevelPenalty). Rank 1 is learned at 20, rank 4 at 50.
func TestConsecrationCoefficientFollowsTheLevelGap(t *testing.T) {
	const fullCoefficient = 0.042
	cases := []struct {
		level   int32
		spellID int32
		want    float64
	}{
		{37, 26573, fullCoefficient},
		{40, 26573, fullCoefficient * 0.85},
		{50, 26573, fullCoefficient * 0.35},
		{60, 26573, 0},
		{60, 20923, fullCoefficient},
	}
	for _, c := range cases {
		env, _, _ := core.NewEnvironment(
			core.SinglePlayerRaidProto(core.WithSpec(&proto.Player{
				Class: proto.Class_ClassPaladin, Race: proto.Race_RaceHuman, Level: c.level,
				Equipment: &proto.EquipmentSpec{}, TalentsString: Phase45RetTalents,
				Rotation: consecrationRotation,
			}, PlayerOptionsSealofRighteousness), &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			&proto.Encounter{Duration: 30, Targets: []*proto.Target{core.NewDefaultTarget()}},
			false,
		)
		paladin := env.Raid.Parties[0].Players[0].GetCharacter()
		spell := paladin.GetSpell(core.ActionID{SpellID: c.spellID})
		if spell == nil {
			t.Fatalf("level %d: Consecration %d not registered", c.level, c.spellID)
		}
		if got := spell.AOEDot().BonusCoefficient; math.Abs(got-c.want) > 1e-12 {
			t.Errorf("level %d, spell %d: coefficient %v, want %v", c.level, c.spellID, got, c.want)
		}
	}
}
