package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// The mana oils state their mana per five seconds and healing in the
// client's item tooltips (20745, 20747, 20748) and in the enchant spells
// those apply (25114, 25115, 25116: aura 85 and aura 135). Vanilla's 4, 8
// and 12 mana with 0, 0 and 25 healing are not this build's.
func TestManaOilsMatchTheClient(t *testing.T) {
	want := map[proto.WeaponImbue]stats.Stats{
		proto.WeaponImbue_MinorManaOil:     {stats.MP5: 5, stats.HealingPower: 10},
		proto.WeaponImbue_LesserManaOil:    {stats.MP5: 10, stats.HealingPower: 20},
		proto.WeaponImbue_BrilliantManaOil: {stats.MP5: 15, stats.HealingPower: 30},
	}
	for imbue, expected := range want {
		if got := manaOilStats[imbue]; got != expected {
			t.Errorf("%v gives %v, want %v", imbue, got, expected)
		}
	}
}

// The wizard oils' enchant spells state damage AND healing (25109, 25110,
// 25111, 25113: aura 13 and aura 135 at 8/16/24/36, school mask 126), and
// Brilliant Wizard Oil adds aura 57, 1% spell crit. The engine gave the
// spell damage only (sim audit round 8, items lane).
func TestWizardOilsMatchTheClient(t *testing.T) {
	want := map[proto.WeaponImbue]stats.Stats{
		proto.WeaponImbue_MinorWizardOil:     {stats.SpellPower: 8, stats.HealingPower: 8},
		proto.WeaponImbue_LesserWizardOil:    {stats.SpellPower: 16, stats.HealingPower: 16},
		proto.WeaponImbue_WizardOil:          {stats.SpellPower: 24, stats.HealingPower: 24},
		proto.WeaponImbue_BrilliantWizardOil: {stats.SpellPower: 36, stats.HealingPower: 36, stats.Crit: 1 * CritRatingPerCritChance},
	}
	for imbue, expected := range want {
		if got := wizardOilStats[imbue]; got != expected {
			t.Errorf("%v gives %v, want %v", imbue, got, expected)
		}
	}
}

// Nightfin Soup's well-fed buff is 22 spell damage (spell 1249513 states
// it through an aura 227 effect of 22); the client gives it no mana.
func TestNightfinSoupMatchesTheClient(t *testing.T) {
	want := stats.Stats{stats.SpellDamage: 22}
	if nightfinSoupStats != want {
		t.Errorf("Nightfin Soup gives %v, want %v", nightfinSoupStats, want)
	}
	if nightfinSoupStats[stats.MP5] != 0 {
		t.Error("Nightfin Soup must give no mana per five seconds")
	}
}

// The healer elixirs and Sage's Tea state their amounts in the client's
// item and aura rows (see the comments on the tables).
func TestHealerConsumablesMatchTheClient(t *testing.T) {
	checks := []struct {
		name      string
		got, want stats.Stats
	}{
		{"Mageblood Elixir", manaRegenElixirStats[proto.ManaRegenElixir_MagebloodPotion], stats.Stats{stats.MP5: 12}},
		{"Greater Mageblood Elixir", manaRegenElixirStats[proto.ManaRegenElixir_GreaterMagebloodElixir], stats.Stats{stats.MP5: 20}},
		{"Cleric's Elixir", healingPowerBuffStats[proto.HealingPowerBuff_ClericsElixir], stats.Stats{stats.HealingPower: 30}},
		{"Greater Cleric's Elixir", healingPowerBuffStats[proto.HealingPowerBuff_GreaterClericsElixir], stats.Stats{stats.HealingPower: 40}},
		{"Elixir of Sages", spiritElixirStats[proto.SpiritElixir_ElixirOfSages], stats.Stats{stats.Spirit: 25, stats.Crit: 2 * CritRatingPerCritChance}},
		{"Sage's Tea", sagesTeaStats, stats.Stats{stats.HealingPower: 44}},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s gives %v, want %v", c.name, c.got, c.want)
		}
	}
}

// The consumables reach a character through its Consumes message.
func TestHealerConsumablesApply(t *testing.T) {
	character := newGnomeEurekaTestCaster(proto.Class_ClassPriest)
	applyFoodConsumes(character, &proto.Consumes{Food: proto.Food_FoodSagesTea})
	applySpellBuffConsumes(character, &proto.Consumes{
		ManaRegenElixir:  proto.ManaRegenElixir_GreaterMagebloodElixir,
		HealingPowerBuff: proto.HealingPowerBuff_GreaterClericsElixir,
		SpiritElixir:     proto.SpiritElixir_ElixirOfSages,
	})
	for stat, want := range map[stats.Stat]float64{
		stats.HealingPower: 84, stats.MP5: 20, stats.Spirit: 25, stats.Crit: 2 * CritRatingPerCritChance,
	} {
		if got := character.GetStat(stat); got != want {
			t.Errorf("stat %v is %v, want %v", stat, got, want)
		}
	}
}
