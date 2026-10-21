package core

import (
	"slices"
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// restoreSetRegistry puts the global set registries back after a test that
// registers a set.
func restoreSetRegistry(t *testing.T) {
	t.Helper()
	savedSets := slices.Clone(sets)
	savedModels := map[int32][]ClientBonusModel{}
	for id, models := range clientSetModels {
		savedModels[id] = models
	}
	t.Cleanup(func() {
		sets = savedSets
		clientSetModels = savedModels
	})
}

func TestClientRowsAreSelfContained(t *testing.T) {
	for id, set := range clientSetRows {
		if set.Name == "" || len(set.Bonuses) == 0 {
			t.Errorf("set %d has no name or bonuses", id)
		}
		for _, bonus := range set.Bonuses {
			spell, ok := clientSpellRows[bonus.SpellID]
			if !ok {
				t.Errorf("set %d (%s) names spell %d, which has no row", id, set.Name, bonus.SpellID)
				continue
			}
			for _, effect := range spell.Effects {
				if _, ok := clientSpellRows[effect.Trigger]; effect.Trigger != 0 && !ok {
					t.Errorf("spell %d (%s) triggers %d, which has no row", bonus.SpellID, spell.Name, effect.Trigger)
				}
			}
		}
	}
}

func flatStats(t *testing.T, spellID int32) ClientFlatBonus {
	t.Helper()
	flat, ok := DecodeClientFlatBonus(MustClientSpellRow(spellID))
	if !ok {
		t.Fatalf("spell %d does not decode as a flat bonus", spellID)
	}
	return flat
}

func TestDecodeFlatBonus(t *testing.T) {
	tests := []struct {
		name    string
		spellID int32
		want    stats.Stats
	}{
		{"Attack Power 30 is melee and ranged", 9336, stats.Stats{stats.AttackPower: 30, stats.RangedAttackPower: 30}},
		{"equal damage and healing is spell power", 9346, stats.Stats{stats.SpellPower: 18}},
		{"healing alone", 9318, stats.Stats{stats.HealingPower: 33}},
		{"hybrid healing and damage", 467550, stats.Stats{stats.HealingPower: 44, stats.SpellDamage: 15}},
		{"hit is one stat, not melee plus spell", 1300947, stats.Stats{stats.Hit: 1 * HitRatingPerHitChance}},
		{"haste is melee and spell", 1300968, stats.Stats{stats.MeleeHaste: HasteRatingPerHastePercent, stats.SpellHaste: HasteRatingPerHastePercent}},
		{"spirit", 1300943, stats.Stats{stats.Spirit: 10}},
		{"defense skill", 1300949, stats.Stats{stats.Defense: 7 * DefenseRatingPerDefense}},
		{"armor", 14803, stats.Stats{stats.Armor: 200}},
		{"all resistances", 18679, stats.Stats{
			stats.ArcaneResistance: 8, stats.FireResistance: 8, stats.FrostResistance: 8,
			stats.NatureResistance: 8, stats.ShadowResistance: 8,
		}},
		{"one resistance", 14673, stats.Stats{stats.ShadowResistance: 10}},
		{"mana per five", 18378, stats.Stats{stats.MP5: 8}},
		{"expertise from dodge and parry reduction", 1213289, stats.Stats{stats.Expertise: 2 * ExpertiseRatingPerExpertiseChance}},
		{"expertise from a rating", 1301083, stats.Stats{stats.Expertise: 1.2 * ExpertiseRatingPerExpertiseChance}},
		{"crit from every source", 1251991, stats.Stats{stats.Crit: 1 * CritRatingPerCritChance}},
		{"holy damage only", 21518, stats.Stats{stats.HolyPower: 29}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := flatStats(t, tt.spellID).Stats; got != tt.want {
				t.Errorf("spell %d decodes to %v, want %v", tt.spellID, got, tt.want)
			}
		})
	}
}

func TestDecodeFlatBonusAgainstCreatureTypes(t *testing.T) {
	humanoids := flatStats(t, 1301088)
	if len(humanoids.AttackPowerVs) != 1 || humanoids.AttackPowerVs[0].Amount != 36 ||
		!slices.Equal(humanoids.AttackPowerVs[0].MobTypes, []proto.MobType{proto.MobType_MobTypeHumanoid}) {
		t.Errorf("Rogue 4P decodes to %+v, want 36 attack power against humanoids", humanoids.AttackPowerVs)
	}
	if humanoids.Stats != (stats.Stats{}) {
		t.Errorf("an against-creature bonus also decoded stats: %v", humanoids.Stats)
	}
	demons := flatStats(t, 1301094)
	if len(demons.SpellDamageVs) != 1 || demons.SpellDamageVs[0].Amount != 21 ||
		!slices.Equal(demons.SpellDamageVs[0].MobTypes, []proto.MobType{proto.MobType_MobTypeDemon}) {
		t.Errorf("Warlock 4P decodes to %+v, want 21 spell damage against demons", demons.SpellDamageVs)
	}
}

func TestDecodeRefusesWhatItDoesNotRead(t *testing.T) {
	for _, tt := range []struct {
		name    string
		spellID int32
	}{
		{"a cooldown modifier", 1301013},
		{"a proc trigger", 1301123},
		{"a dummy", 1301715},
		{"a skill other than defense", 7534},
	} {
		if _, ok := DecodeClientFlatBonus(MustClientSpellRow(tt.spellID)); ok {
			t.Errorf("%s (%d) decoded as a flat bonus", tt.name, tt.spellID)
		}
	}
}

// Tier 1 Mage (2098) is a real row with a stat, a modifier, a creature-type
// bonus and a dummy: the four shapes a model has to account for.
func TestNewClientItemSetRequiresEveryBonusToBeAccountedFor(t *testing.T) {
	restoreSetRegistry(t)
	const id, hit, counterspell, vsElementals, frostfire = 2098, 1300947, 1301013, 1301079, 1301488

	mustPanic := func(name, wantSubstring string, model ClientSetModel) {
		t.Helper()
		defer func() {
			got := recover()
			if got == nil {
				t.Errorf("%s: no panic", name)
				return
			}
			if message, _ := got.(string); !strings.Contains(message, wantSubstring) {
				t.Errorf("%s: panic %q does not mention %q", name, message, wantSubstring)
			}
		}()
		NewClientItemSet(model)
	}

	noop := func(Agent) {}
	mustPanic("an unaccounted modifier", "not a flat stat", ClientSetModel{ID: id})
	mustPanic("an Effects key the set lacks", "has no bonus spell 1", ClientSetModel{ID: id, Effects: map[int32]ApplyEffect{1: noop}})
	mustPanic("a NoSim key the set lacks", "has no bonus spell 1", ClientSetModel{ID: id, NoSim: map[int32]string{1: "x"}})
	mustPanic("a NoSim without a reason", "without a reason", ClientSetModel{ID: id, NoSim: map[int32]string{counterspell: ""}})
	mustPanic("both modelled and NoSim", "both modelled and NoSim", ClientSetModel{
		ID: id, Effects: map[int32]ApplyEffect{counterspell: noop}, NoSim: map[int32]string{counterspell: "x"},
	})
	mustPanic("a set the client table lacks", "is not in client_sets_gen.go", ClientSetModel{ID: 999999})

	set := NewClientItemSet(ClientSetModel{
		ID:      id,
		Effects: map[int32]ApplyEffect{counterspell: noop, frostfire: noop},
		NoSim:   map[int32]string{vsElementals: "the sim fights no elemental"},
	})
	if set.Name != "Manaflare Regalia" || set.ID != id {
		t.Errorf("set is %q (%d)", set.Name, set.ID)
	}
	if got := len(set.Bonuses); got != 4 {
		t.Errorf("set carries %d thresholds, want 4", got)
	}

	want := []ClientBonusModel{
		{Threshold: 2, SpellID: hit, Kind: ClientBonusStat},
		{Threshold: 3, SpellID: counterspell, Kind: ClientBonusModelled},
		{Threshold: 4, SpellID: vsElementals, Kind: ClientBonusNoSim, Reason: "the sim fights no elemental"},
		{Threshold: 5, SpellID: frostfire, Kind: ClientBonusModelled},
	}
	if got := ClientSetModels()[id]; !slices.Equal(got, want) {
		t.Errorf("recorded models %+v, want %+v", got, want)
	}
}
