package sim

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

func runSampleSim(t *testing.T, player *proto.Player) *proto.SampleIteration {
	t.Helper()
	result := core.RunSim(&proto.RaidSimRequest{
		Raid:      core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: parityEncounter(),
		SimOptions: &proto.SimOptions{
			Iterations:      parityIterations,
			IsTest:          true,
			RandomSeed:      1,
			SampleIteration: true,
		},
	}, nil, simsignals.CreateSignals())
	if result.Error != nil {
		t.Fatalf("sim failed: %s", result.Error.Message)
	}
	if result.SampleIteration == nil {
		t.Fatal("SampleIteration = nil, want a log")
	}
	return result.SampleIteration
}

// A fury warrior's log reads rage after every cast, and never mana.
func TestFuryWarriorSampleLogReadsRage(t *testing.T) {
	sample := runSampleSim(t, furyWarriorPlayer())

	sawRage := false
	for _, cast := range sample.Casts {
		if _, ok := cast.Resources["rage"]; ok {
			sawRage = true
		}
		if _, ok := cast.Resources["mana"]; ok {
			t.Fatalf("a warrior's cast at %dms reported mana", cast.AtMs)
		}
		if cast.Resources["rage"] < 0 || cast.Resources["rage"] > 100 {
			t.Fatalf("rage %d at %dms is outside the bar", cast.Resources["rage"], cast.AtMs)
		}
	}
	if !sawRage {
		t.Error("no cast in the warrior's sample log reported rage")
	}
}

// A frost mage's log reads mana after every cast, and never rage.
func TestFrostMageSampleLogReadsMana(t *testing.T) {
	sample := runSampleSim(t, frostMagePlayer())

	sawMana := false
	for _, cast := range sample.Casts {
		if _, ok := cast.Resources["mana"]; ok {
			sawMana = true
		}
		if _, ok := cast.Resources["rage"]; ok {
			t.Fatalf("a mage's cast at %dms reported rage", cast.AtMs)
		}
	}
	if !sawMana {
		t.Error("no cast in the mage's sample log reported mana")
	}
}

// magePrepullAction grafts a Fire Blast at -1s onto the frost mage
// fixture's rotation. ui/mage/apls/forever_frost.apl.json has no
// prepullActions at all - "Frostbolt is the whole rotation," per the
// file's own note - so runSampleSim would never see a negative-time cast
// for the mage without one. The brief's suggested lever for this case is
// player.Cooldowns, but that field only feeds the SimpleRotation cooldown
// list (proto/apl.proto's SimpleRotation.cooldowns); it has no effect on
// a TypeAPL rotation such as this fixture's, so it cannot add a pre-pull
// cast here. Appending directly to player.Rotation.PrepullActions is the
// APL-rotation equivalent: a real cast grafted onto the rotation, not a
// weakened assertion.
//
// It is Fire Blast (spellId 10199, rank 7), not Frostbolt: Frostbolt's
// ~2.5s cast time means a cast started at -1s does not complete, and so
// is not recorded, until after the pull - recordSampleCast logs a cast
// at the time its effects apply, not the time it started, so that cast
// would land at a positive AtMs and this test would never observe a
// negative one. Fire Blast is baseline for every mage spec and has no
// cast time (see sim/mage/fire_blast.go: CastConfig sets only GCD), so
// it completes, and is recorded, at exactly -1000ms.
func magePrepullAction() *proto.APLPrepullAction {
	return &proto.APLPrepullAction{
		Action: &proto.APLAction{
			Action: &proto.APLAction_CastSpell{
				CastSpell: &proto.APLActionCastSpell{
					SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: 10199}, Rank: 7},
				},
			},
		},
		DoAtValue: &proto.APLValue{
			Value: &proto.APLValue_Const{Const: &proto.APLValueConst{Val: "-1s"}},
		},
	}
}

// Pre-pull casts are what the report separates out, and they are
// identified by a negative time. Both reference rotations open with one.
func TestSampleLogSeparatesThePrePull(t *testing.T) {
	for _, tc := range []struct {
		name   string
		player func() *proto.Player
	}{
		{"fury warrior", furyWarriorPlayer},
		{"frost mage", func() *proto.Player {
			player := frostMagePlayer()
			if len(player.Rotation.PrepullActions) == 0 {
				player.Rotation.PrepullActions = append(player.Rotation.PrepullActions, magePrepullAction())
			}
			return player
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sample := runSampleSim(t, tc.player())

			prePull := 0
			for i, cast := range sample.Casts {
				if cast.AtMs < 0 {
					prePull++
				}
				if i > 0 && cast.AtMs < sample.Casts[i-1].AtMs {
					t.Fatalf("cast %d at %dms comes before cast %d at %dms; the log is not in time order", i, cast.AtMs, i-1, sample.Casts[i-1].AtMs)
				}
			}
			if prePull == 0 {
				t.Errorf("no pre-pull cast in the log; the reference rotation opens with one, so a negative time is never produced")
			}
			if len(sample.Casts) > 0 && sample.Casts[0].AtMs >= 0 && prePull > 0 {
				t.Error("a pre-pull cast is not first in the log; the log is not in time order")
			}
		})
	}
}

// Every cast names a target and a spell, so the report can render a row.
func TestSampleLogCastsAreRenderable(t *testing.T) {
	sample := runSampleSim(t, furyWarriorPlayer())
	for _, cast := range sample.Casts {
		if cast.ActionId == nil {
			t.Fatalf("a cast at %dms has no ActionId", cast.AtMs)
		}
		if cast.ActionId.GetSpellId() == 0 && cast.ActionId.GetItemId() == 0 && cast.ActionId.GetOtherId() == 0 {
			t.Fatalf("a cast at %dms has an empty ActionId", cast.AtMs)
		}
	}
}
