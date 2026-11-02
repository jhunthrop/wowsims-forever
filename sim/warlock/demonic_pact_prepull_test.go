package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

const (
	summonImpSpellID        = 688
	summonSuccubusSpellID   = 712
	demonicSacrificeSpellID = 18788
	burningWishSpellID      = 18789
	demonicPactSpellID      = 425464
)

func spellIDProto(id int32) *proto.ActionID {
	return &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: id}}
}

// pactGatedPrepull is Demonology's prepull: the Imp and its sacrifice only
// under Demonic Pact, then the Succubus. Without Pact the Succubus summon
// would cancel the sacrifice, so the rotation keeps the Succubus instead.
func pactGatedPrepull() *proto.APLRotation {
	pactKnown := &proto.APLValue{Value: &proto.APLValue_AuraIsKnown{AuraIsKnown: &proto.APLValueAuraIsKnown{AuraId: spellIDProto(demonicPactSpellID)}}}
	cast := func(id int32, condition *proto.APLValue, doAt string) *proto.APLPrepullAction {
		return &proto.APLPrepullAction{
			Action: &proto.APLAction{
				Condition: condition,
				Action:    &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{SpellId: spellIDProto(id)}},
			},
			DoAtValue: &proto.APLValue{Value: &proto.APLValue_Const{Const: &proto.APLValueConst{Val: doAt}}},
		}
	}
	return &proto.APLRotation{
		Type: proto.APLRotation_TypeAPL,
		PrepullActions: []*proto.APLPrepullAction{
			cast(summonImpSpellID, pactKnown, "-60s"),
			cast(demonicSacrificeSpellID, pactKnown, "-45s"),
			cast(summonSuccubusSpellID, nil, "-44s"),
		},
	}
}

func runPrepullOnly(t *testing.T, talents string) *proto.UnitMetrics {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassWarlock,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			TalentsString: talents,
			Rotation:      pactGatedPrepull(),
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

func castsOf(player *proto.UnitMetrics, id int32) int32 {
	var casts int32
	for _, action := range player.Actions {
		if action.Id.GetSpellId() == id {
			for _, target := range action.Targets {
				casts += target.Casts
			}
		}
	}
	return casts
}

func uptimeOf(player *proto.UnitMetrics, id int32) float64 {
	for _, aura := range player.Auras {
		if aura.Id.GetSpellId() == id {
			return aura.UptimeSecondsAvg
		}
	}
	return 0
}

// A prepull action's condition decides whether it runs, and Demonic Pact
// is visible to the rotation as a known aura, so a Demonology prepull can
// sacrifice the Imp only when the following summon will not cancel it.
// Demonology field 10 is Demonic Sacrifice, field 19 Demonic Pact.
func TestPrepullSacrificesOnlyUnderDemonicPact(t *testing.T) {
	withoutPact := runPrepullOnly(t, "-0000000001")
	if got := castsOf(withoutPact, demonicSacrificeSpellID); got != 0 {
		t.Errorf("without Demonic Pact the prepull cast Demonic Sacrifice %d times, want 0: the Succubus summon cancels it", got)
	}
	if got := castsOf(withoutPact, summonImpSpellID); got != 0 {
		t.Errorf("without Demonic Pact the prepull summoned the Imp %d times, want 0: it is only summoned to be sacrificed", got)
	}
	if got := castsOf(withoutPact, summonSuccubusSpellID); got == 0 {
		t.Error("without Demonic Pact the prepull never summoned the Succubus")
	}

	withPact := runPrepullOnly(t, "-0000000001000000001")
	if got := castsOf(withPact, demonicSacrificeSpellID); got == 0 {
		t.Error("with Demonic Pact the prepull never cast Demonic Sacrifice")
	}
	if got := uptimeOf(withPact, burningWishSpellID); got < 29 {
		t.Errorf("with Demonic Pact Burning Wish is up %.1f s of a 30 s fight, want all of it", got)
	}
}
