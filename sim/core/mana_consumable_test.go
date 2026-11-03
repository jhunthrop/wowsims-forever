package core

import (
	"testing"

	googleProto "google.golang.org/protobuf/proto"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// manaConsumableSim is a level-60 Elemental shaman holding consumes, reset
// and ready to use them.
func manaConsumableSim(t *testing.T, consumes *proto.Consumes) (*Simulation, *Character) {
	t.Helper()
	target := googleProto.Clone(DefaultTargetProtoLvl60).(*proto.Target)
	env, _, _ := NewEnvironment(
		SinglePlayerRaidProto(&proto.Player{
			Name:      "Mana Consumable Test",
			Race:      proto.Race_RaceHuman,
			Class:     proto.Class_ClassShaman,
			Spec:      &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{}},
			Equipment: &proto.EquipmentSpec{},
			Consumes:  consumes,
		}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		&proto.Encounter{Duration: 180, Targets: []*proto.Target{target}},
		false,
	)
	sim := newSimWithEnv(env, &proto.SimOptions{RandomSeed: 1}, simsignals.CreateSignals())
	sim.reset()
	character := env.Raid.Parties[0].Players[0].GetCharacter()
	// Room for the largest roll, so no gain is cut at maximum mana.
	character.AddStatDynamic(sim, stats.Mana, 10000)
	return sim, character
}

// Every mana potion and mana-restoring conjured item restores the client's
// amount: SpellEffect energize (30) EffectBasePointsF with its Variance as
// the full width, so base x (1 -/+ Variance/2), 1.60.1.70753:
//   - Lesser Mana Potion 3385 -> 438: 320, variance 0.25 (280 to 360)
//   - Mana Potion 3827 -> 2023: 520, 0.25 (455 to 585)
//   - Greater Mana Potion 6149 -> 11903: 800, 0.25 (700 to 900)
//   - Superior Mana Potion 13443 -> 17530: 1200, 0.5 (900 to 1500)
//   - Major Mana Potion 13444 -> 17531: 1800, 0.5 (1350 to 2250)
//   - Minor Recombobulator 4381 -> 18805: 128 mana, 0.5 (96 to 160)
//   - Demonic Rune 12662 -> 16666: 1200, 0.5 (900 to 1500)
//
// The engine had no row for the Lesser Mana Potion (it restored nothing)
// and Era's 150 to 250 for the Recombobulator.
func TestManaConsumablesRestoreTheClientsAmount(t *testing.T) {
	cases := []struct {
		name     string
		consumes *proto.Consumes
		itemID   int32
		min, max float64
	}{
		{"Lesser Mana Potion", &proto.Consumes{DefaultPotion: proto.Potions_LesserManaPotion}, 3385, 280, 360},
		{"Mana Potion", &proto.Consumes{DefaultPotion: proto.Potions_ManaPotion}, 3827, 455, 585},
		{"Greater Mana Potion", &proto.Consumes{DefaultPotion: proto.Potions_GreaterManaPotion}, 6149, 700, 900},
		{"Superior Mana Potion", &proto.Consumes{DefaultPotion: proto.Potions_SuperiorManaPotion}, 13443, 900, 1500},
		{"Major Mana Potion", &proto.Consumes{DefaultPotion: proto.Potions_MajorManaPotion}, 13444, 1350, 2250},
		{"Minor Recombobulator", &proto.Consumes{DefaultConjured: proto.Conjured_ConjuredMinorRecombobulator}, 4381, 96, 160},
		{"Demonic Rune", &proto.Consumes{DefaultConjured: proto.Conjured_ConjuredDemonicRune}, 12662, 900, 1500},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sim, character := manaConsumableSim(t, c.consumes)
			spell := character.GetSpell(ActionID{ItemID: c.itemID})
			if spell == nil {
				t.Fatalf("item %d is not registered", c.itemID)
			}
			lo, hi := c.max, c.min
			for i := 0; i < 400; i++ {
				character.currentMana = 0
				spell.ApplyEffects(sim, &character.Unit, spell)
				got := character.CurrentMana()
				lo, hi = min(lo, got), max(hi, got)
			}
			if lo < c.min-1e-9 || hi > c.max+1e-9 || hi-lo < 0.8*(c.max-c.min) {
				t.Errorf("restored %.0f to %.0f mana over 400 uses, want the client's %.0f to %.0f", lo, hi, c.min, c.max)
			}
		})
	}
}

// The healthstones heal the client's flat amounts (1.60.1.70753 SpellEffect,
// variance 0): Healthstone 5509 -> 5720 600, Greater Healthstone 5510 ->
// 5723 960, Major Healthstone 9421 -> 11732 1440 (server text "Instantly
// restores 1440 life"). The engine had Era's 500, 800 and 1200.
func TestHealthstonesHealTheClientsAmount(t *testing.T) {
	cases := []struct {
		conjured proto.Conjured
		itemID   int32
		want     float64
	}{
		{proto.Conjured_ConjuredHealthstone, 5509, 600},
		{proto.Conjured_ConjuredGreaterHealthstone, 5510, 960},
		{proto.Conjured_ConjuredMajorHealthstone, 9421, 1440},
	}
	for _, c := range cases {
		sim, character := manaConsumableSim(t, &proto.Consumes{DefaultConjured: c.conjured})
		character.AddStatDynamic(sim, stats.Health, 10000)
		spell := character.GetSpell(ActionID{ItemID: c.itemID})
		if spell == nil {
			t.Fatalf("item %d is not registered", c.itemID)
		}
		character.RemoveHealth(sim, 5000)
		before := character.CurrentHealth()
		spell.ApplyEffects(sim, &character.Unit, spell)
		if got := character.CurrentHealth() - before; got != c.want {
			t.Errorf("item %d healed %v, want %v", c.itemID, got, c.want)
		}
	}
}
