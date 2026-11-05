package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// talentGatedSacrifice is Affliction's raid prepull (sim-accuracy round 22):
// summon the Imp and sacrifice it, both gated on knowing Demonic Sacrifice,
// so the gate names the talent and a build without it keeps its demon.
func talentGatedSacrifice() *proto.APLRotation {
	known := &proto.APLValue{Value: &proto.APLValue_AuraIsKnown{AuraIsKnown: &proto.APLValueAuraIsKnown{AuraId: spellIDProto(demonicSacrificeSpellID)}}}
	cast := func(id int32, doAt string) *proto.APLPrepullAction {
		return &proto.APLPrepullAction{
			Action: &proto.APLAction{
				Condition: known,
				Action:    &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{SpellId: spellIDProto(id)}},
			},
			DoAtValue: &proto.APLValue{Value: &proto.APLValue_Const{Const: &proto.APLValueConst{Val: doAt}}},
		}
	}
	return &proto.APLRotation{
		Type:           proto.APLRotation_TypeAPL,
		PrepullActions: []*proto.APLPrepullAction{cast(summonImpSpellID, "-60s"), cast(demonicSacrificeSpellID, "-45s")},
	}
}

func runWithRotation(t *testing.T, talents string, rot *proto.APLRotation) *proto.UnitMetrics {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class: proto.Class_ClassWarlock, Race: proto.Race_RaceOrc, Level: 60,
			Equipment: &proto.EquipmentSpec{}, TalentsString: talents, Rotation: rot,
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

// Demonic Sacrifice is readable as a known aura under its talent spell
// (18788), as Demonic Pact is (425464), so a card can gate on the talent
// itself rather than on the buff it grants. Demonology field 10.
func TestDemonicSacrificeTalentIsAKnownAura(t *testing.T) {
	without := runWithRotation(t, "-", talentGatedSacrifice())
	if got := castsOf(without, demonicSacrificeSpellID); got != 0 {
		t.Errorf("without the talent the prepull cast Demonic Sacrifice %d times, want 0", got)
	}
	with := runWithRotation(t, "-0000000001", talentGatedSacrifice())
	if got := castsOf(with, demonicSacrificeSpellID); got == 0 {
		t.Error("with the talent the gated prepull never cast Demonic Sacrifice")
	}
	if got := uptimeOf(with, burningWishSpellID); got < 29 {
		t.Errorf("with the talent Burning Shadow is up %.1f s of a 30 s fight, want all of it", got)
	}
}

// Nightfall is readable as a known aura under its talent spell (18094), so
// a card can gate its Shadow Trance bolt on the talent. Affliction field 12.
func TestNightfallTalentIsAKnownAura(t *testing.T) {
	known := &proto.APLValue{Value: &proto.APLValue_AuraIsKnown{AuraIsKnown: &proto.APLValueAuraIsKnown{AuraId: spellIDProto(18094)}}}
	rot := &proto.APLRotation{
		Type: proto.APLRotation_TypeAPL,
		PriorityList: []*proto.APLListItem{{Action: &proto.APLAction{
			Condition: known,
			Action:    &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{SpellId: spellIDProto(686)}},
		}}},
	}
	if got := castsOf(runWithRotation(t, "", rot), 686); got != 0 {
		t.Errorf("without Nightfall the gated Shadow Bolt was cast %d times, want 0", got)
	}
	if got := castsOf(runWithRotation(t, "000000000002", rot), 686); got == 0 {
		t.Error("with Nightfall the gated Shadow Bolt was never cast")
	}
}
