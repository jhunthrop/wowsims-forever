package rogue_test

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	dpsrogue "github.com/wowsims/classic/sim/rogue/dps_rogue"
)

// rogueStatsWith is a bare level-60 rogue's Strength and Agility under the
// given raid buffs and consumes.
func rogueStatsWith(t *testing.T, raidBuffs *proto.RaidBuffs, consumes *proto.Consumes) (float64, float64) {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class: proto.Class_ClassRogue, Race: proto.Race_RaceHuman, Level: 60,
		Equipment: &proto.EquipmentSpec{}, Consumes: consumes, DistanceFromTarget: 5,
	}, &proto.Player_Rogue{Rogue: &proto.Rogue{Options: &proto.RogueOptions{}}})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, raidBuffs, &proto.Debuffs{}),
		Encounter:  &proto.Encounter{Duration: 60},
		SimOptions: core.DefaultSimTestOptions,
	}, simsignals.CreateSignals())
	sim.Reset()
	r := sim.Raid.Parties[0].Players[0].(*dpsrogue.DpsRogue)
	s := r.GetStats()
	return s[stats.Strength], s[stats.Agility]
}

// The raid buffs' Scroll of Strength and Scroll of Agility (RaidBuffs
// scroll_of_strength, scroll_of_agility, which the site's buff list offers)
// did nothing: the engine dropped both when the totems became auras, so a
// scroll showed no stat change. A scroll gives its +17 when no totem of the
// same stat is up (the rule the engine keeps for the Intellect, Stamina and
// Spirit scrolls), and is not counted twice when the consume slot already
// holds the same scroll.
func TestRaidBuffScrollsOfStrengthAndAgilityApply(t *testing.T) {
	str0, agi0 := rogueStatsWith(t, &proto.RaidBuffs{}, &proto.Consumes{})
	str, agi := rogueStatsWith(t, &proto.RaidBuffs{ScrollOfStrength: true, ScrollOfAgility: true}, &proto.Consumes{})
	if str-str0 != 17 || agi-agi0 != 17 {
		t.Errorf("raid-buff scrolls: Strength +%v, Agility +%v, want +17 each", str-str0, agi-agi0)
	}

	strTotems, agiTotems := rogueStatsWith(t, &proto.RaidBuffs{
		StrengthOfEarthTotem: proto.TristateEffect_TristateEffectRegular, GraceOfAirTotem: proto.TristateEffect_TristateEffectRegular,
	}, &proto.Consumes{})
	strBoth, agiBoth := rogueStatsWith(t, &proto.RaidBuffs{
		StrengthOfEarthTotem: proto.TristateEffect_TristateEffectRegular, GraceOfAirTotem: proto.TristateEffect_TristateEffectRegular,
		ScrollOfStrength: true, ScrollOfAgility: true,
	}, &proto.Consumes{})
	if strBoth != strTotems || agiBoth != agiTotems {
		t.Errorf("a scroll under its totem: Strength %v vs %v, Agility %v vs %v; the totem should win", strBoth, strTotems, agiBoth, agiTotems)
	}

	consumes := &proto.Consumes{StrengthBuff: proto.StrengthBuff_ScrollOfStrength, AgilityElixir: proto.AgilityElixir_ScrollOfAgility}
	strC, agiC := rogueStatsWith(t, &proto.RaidBuffs{}, consumes)
	strCR, agiCR := rogueStatsWith(t, &proto.RaidBuffs{ScrollOfStrength: true, ScrollOfAgility: true}, consumes)
	if strCR != strC || agiCR != agiC {
		t.Errorf("the same scroll as a consume and a raid buff counted twice: Strength %v vs %v, Agility %v vs %v", strCR, strC, agiCR, agiC)
	}
}
