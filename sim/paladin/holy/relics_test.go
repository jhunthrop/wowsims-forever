package holy

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/paladin"
)

func holyRelicFingerprints(t *testing.T, relicID int32) map[string]core.SpellFingerprint {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassPaladin,
		Race:          proto.Race_RaceHuman,
		Level:         60,
		Equipment:     core.RelicEquipment(relicID),
		TalentsString: paladinTalents(t, map[string]int{"holy_shock": 1}),
	}, &proto.Player_HolyPaladin{HolyPaladin: &proto.HolyPaladin{Options: &proto.PaladinOptions{
		PrimarySeal: proto.PaladinSeal_Righteousness,
	}}})
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)
	return core.SpellFingerprints(&env.Raid.Parties[0].Players[0].GetCharacter().Unit)
}

// Libram of Infusion 279248 (client spell 1306429, aura 107 op 7, families
// 0x200000 / 0x10000): +6% crit chance for Holy Shock. The heal 1311605
// carries both bits (the cast 1311606 only the first), so a Holy paladin's
// Holy Shock heals crit 6% more often, not just its damage.
func TestLibramOfInfusionRaisesTheHolyShockHealsCrit(t *testing.T) {
	changed := core.ChangedSpells(holyRelicFingerprints(t, 0), holyRelicFingerprints(t, paladin.LibramOfInfusion))
	heals := 0
	for _, change := range changed {
		if change.After.ClassSpellMask&(paladin.PaladinSpellMaskHolyShock|paladin.PaladinSpellMaskHolyShockHeal) == 0 {
			t.Errorf("%s changed but is not Holy Shock: %+v", change.Key, change.After)
		}
		if got := change.After.BonusCritRating - change.Before.BonusCritRating; got != 6*core.CritRatingPerCritChance {
			t.Errorf("%s bonus crit moved by %v, want 6%%", change.Key, got)
		}
		if change.After.ClassSpellMask&paladin.PaladinSpellMaskHolyShockHeal != 0 {
			heals++
		}
	}
	if heals == 0 {
		t.Error("no Holy Shock heal gained the libram's crit")
	}
}
