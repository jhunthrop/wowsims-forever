package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// manaSpentBy is the mana a spell took from the warlock over the run, per
// iteration.
func manaSpentBy(player *proto.UnitMetrics, id int32, iterations int) float64 {
	spent := 0.0
	for _, resource := range player.Resources {
		if resource.Type == proto.ResourceType_ResourceTypeMana && resource.Id.GetSpellId() == id {
			spent -= resource.ActualGain
		}
	}
	return spent / float64(iterations)
}

// summonInFight is a rotation with no prepull that summons the Succubus
// once the fight has started.
func summonInFight() *proto.APLRotation {
	return &proto.APLRotation{
		Type: proto.APLRotation_TypeAPL,
		PriorityList: []*proto.APLListItem{{
			Action: &proto.APLAction{Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{SpellId: spellIDProto(summonSuccubusSpellID)}}},
		}},
	}
}

func runRotation(t *testing.T, talents string, rotation *proto.APLRotation) *proto.UnitMetrics {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassWarlock,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			TalentsString: talents,
			Rotation:      rotation,
		},
		&proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.WarlockOptions{
			Armor: proto.WarlockOptions_NoArmor, Summon: proto.WarlockOptions_NoSummon, WeaponImbue: proto.WarlockOptions_NoWeaponImbue,
		}}},
	)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  &proto.Encounter{Duration: 30, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{Iterations: 5, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %v", result.Error.Message)
	}
	return result.RaidMetrics.Parties[0].Players[0]
}

// Sim-accuracy round 29 W2 (closed by the 2026-11-12 warlock lane): the
// pre-pull summons took their mana out of the fight's bar, so Master
// Summoner's only sim value was 40% of a cost the warlock never carries
// into the pull. In play the demons are summoned a minute out, the warlock
// Life Taps back to full and a healer tops the health up (demonology.md's
// pre-pull list), so the pull starts at full mana. A summon during the
// fight (a dead Succubus) pays the full cost, cut by Master Summoner.
// Demonology field 10 is Demonic Sacrifice, 11 Master Summoner, 19 Demonic
// Pact.
func TestSummonsBeforeThePullCostNoMana(t *testing.T) {
	player := runPrepullOnly(t, "-0000000001000000001")
	for _, id := range []int32{summonImpSpellID, summonSuccubusSpellID} {
		if castsOf(player, id) == 0 {
			t.Fatalf("the prepull never cast summon %d", id)
		}
		if spent := manaSpentBy(player, id, 5); spent != 0 {
			t.Errorf("summon %d took %.0f mana before the pull, want 0 (the warlock Life Taps back to full before the pull)", id, spent)
		}
	}
}

func TestSummonsDuringTheFightPayTheirCostCutByMasterSummoner(t *testing.T) {
	plain := runRotation(t, "-000000000100", summonInFight())
	if castsOf(plain, summonSuccubusSpellID) == 0 {
		t.Fatal("the rotation never summoned the Succubus in the fight")
	}
	full := manaSpentBy(plain, summonSuccubusSpellID, 5)
	if full <= 0 {
		t.Fatalf("a Succubus summoned in the fight took %.0f mana, want its full cost", full)
	}
	cut := manaSpentBy(runRotation(t, "-000000000120", summonInFight()), summonSuccubusSpellID, 5)
	if ratio := cut / full; ratio < 0.59 || ratio > 0.61 {
		t.Errorf("Master Summoner 2 cut the in-fight summon's mana to %.0f of %.0f (%.2f), want 0.60", cut, full, ratio)
	}
}
