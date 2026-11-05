package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

func newSummonTestWarlock(t *testing.T, level int32, summon proto.WarlockOptions_Summon) *Warlock {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:     proto.Class_ClassWarlock,
			Race:      proto.Race_RaceOrc,
			Level:     level,
			Equipment: &proto.EquipmentSpec{},
			Buffs:     &proto.IndividualBuffs{},
			Rotation:  &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		},
		&proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.WarlockOptions{
			Armor:  proto.WarlockOptions_NoArmor,
			Summon: summon,
		}}},
	)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	agent, ok := sim.Raid.Parties[0].Players[0].(WarlockAgent)
	if !ok {
		t.Fatal("the raid's first player is not a warlock agent")
	}
	return agent.GetWarlock()
}

// Round 21 (platform): the trainer teaches Summon Voidwalker at 10, Summon
// Succubus at 20 and Summon Felhunter at 30 (trainables 697, 712, 691). The
// engine registered all four summons at every level, so the demonology
// card's pre-pull Summon Succubus fired at the ladder's level-10 rung.
func TestSummonsAreNotKnownBelowTheirTrainerLevel(t *testing.T) {
	for _, tc := range []struct {
		level int32
		id    int32
		known bool
	}{
		{9, 697, false}, {10, 697, true},
		{19, 712, false}, {20, 712, true},
		{29, 691, false}, {30, 691, true},
		{1, 688, true},
	} {
		warlock := newSummonTestWarlock(t, tc.level, proto.WarlockOptions_Imp)
		if got := warlock.GetSpell(core.ActionID{SpellID: tc.id}) != nil; got != tc.known {
			t.Errorf("level %d: summon %d registered = %v, want %v", tc.level, tc.id, got, tc.known)
		}
	}
}

// Every warlock spec's default demon is the Succubus, which a warlock below
// 20 has not learned. Below its trainer level the default falls back to the
// demon a leveling warlock has out: the Voidwalker from 10 (the class quest
// at 10, demonology.md), the Imp before that.
func TestTheDefaultDemonIsOneTheWarlockHasLearned(t *testing.T) {
	for _, tc := range []struct {
		level int32
		want  string
	}{{9, "Imp"}, {15, "Voidwalker"}, {19, "Voidwalker"}, {20, "Succubus"}, {60, "Succubus"}} {
		warlock := newSummonTestWarlock(t, tc.level, proto.WarlockOptions_Succubus)
		if warlock.ActivePet == nil || warlock.ActivePet.Name != tc.want {
			got := "none"
			if warlock.ActivePet != nil {
				got = warlock.ActivePet.Name
			}
			t.Errorf("level %d: default demon %s, want %s", tc.level, got, tc.want)
		}
	}
}
