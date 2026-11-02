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

// Forever's foods are "Nutritious Food": an aura 227 effect whose value is
// passed to the Well Fed it triggers, and the Well Fed's aura names the
// stat (1.60.1.70707 SpellEffect; Wowhead Forever's item tooltips read the
// same). Grilled Squid 13928 is 1% critical strike (Well Fed 1249523, aura
// 290), not Era's 10 Agility; Tender Wolf Steak 18045 and Hot Wolf Ribs
// 13851 are 15 and 10 Agility (1248420, aura 29 misc 1), not Stamina and
// Spirit; Runn Tum Tuber Surprise 18254 is 15 Intellect (1248421), not 10;
// Smoked Sagefish 21072 and Sagefish Delight 21217 are 4 and 7 spell damage
// (1249520, aura 13), not mana per five seconds.
func TestFoodsMatchTheClient(t *testing.T) {
	want := map[proto.Food]stats.Stats{
		proto.Food_FoodHotWolfRibs:              {stats.Agility: 10},
		proto.Food_FoodSmokedSagefish:           {stats.SpellDamage: 4},
		proto.Food_FoodSagefishDelight:          {stats.SpellDamage: 7},
		proto.Food_FoodTenderWolfSteak:          {stats.Agility: 15},
		proto.Food_FoodGrilledSquid:             {stats.Crit: 1 * CritRatingPerCritChance},
		proto.Food_FoodSmokedDesertDumpling:     {stats.Strength: 20},
		proto.Food_FoodNightfinSoup:             {stats.SpellDamage: 22},
		proto.Food_FoodSagesTea:                 {stats.HealingPower: 44},
		proto.Food_FoodRunnTumTuberSurprise:     {stats.Intellect: 15},
		proto.Food_FoodDirgesKickChimaerokChops: {stats.Stamina: 25},
		proto.Food_FoodBlessedSunfruitJuice:     {stats.Spirit: 10},
		proto.Food_FoodBlessSunfruit:            {stats.Strength: 10},
	}
	for food, expected := range want {
		if got := foodStats[food]; got != expected {
			t.Errorf("%v gives %v, want %v", food, got, expected)
		}
	}
}

// Elixir of Greater Defense 13445 is 450 armor in Forever (spell 11348,
// aura 674; Wowhead Forever: "Increases armor by 450"), Era's 250 raised,
// and Elixir of Defense 8951 is 250 (Era's 150);
// Elixir of Fortitude is item 250334, 200 maximum health (spell 1250928,
// aura 34), where Era's 3825 gave 120.
func TestDefensiveElixirsMatchTheClient(t *testing.T) {
	if got := armorElixirStats[proto.ArmorElixir_ElixirOfGreaterDefense]; got != (stats.Stats{stats.BonusArmor: 450}) {
		t.Errorf("Elixir of Greater Defense gives %v, want 450 armor", got)
	}
	if got := armorElixirStats[proto.ArmorElixir_ElixirOfDefense]; got != (stats.Stats{stats.BonusArmor: 250}) {
		t.Errorf("Elixir of Defense gives %v, want 250 armor", got)
	}
	if got := healthElixirStats[proto.HealthElixir_ElixirOfFortitude]; got != (stats.Stats{stats.Health: 200}) {
		t.Errorf("Elixir of Fortitude gives %v, want 200 health", got)
	}
}
