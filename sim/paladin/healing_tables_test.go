package paladin

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
	"github.com/wowsims/classic/sim/core/spellconst"
)

// healEffectIndex is the heal effect's index in every client row below.
const healEffectIndex = 0

var clientLevels = []int{1, 10, 20, 30, 38, 40, 50, 60}

func assertHealRank(t *testing.T, client spellconst.Class, label string, castID, healID int32, rank healRank) {
	t.Helper()
	cast, ok := client.ByID(castID)
	if !ok {
		t.Fatalf("%s: cast spell %d is not in the client table", label, castID)
	}
	if cast.SpellLevel != rank.level {
		t.Errorf("%s (%d): engine level %d, client %d", label, castID, rank.level, cast.SpellLevel)
	}
	if cast.Cost != rank.manaCost {
		t.Errorf("%s (%d): engine mana cost %v, client %v", label, castID, rank.manaCost, cast.Cost)
	}
	healSpell, ok := client.ByID(healID)
	if !ok {
		t.Fatalf("%s: heal spell %d is not in the client table", label, healID)
	}
	if healSpell.SpellLevel != rank.heal.SpellLevel || healSpell.MaxLevel != rank.heal.MaxLevel {
		t.Errorf("%s (%d): engine heal levels %d/%d, client %d/%d", label, healID,
			rank.heal.SpellLevel, rank.heal.MaxLevel, healSpell.SpellLevel, healSpell.MaxLevel)
	}
	for _, level := range clientLevels {
		clientdamagetest.AssertRoll(t, client, label, healID, healEffectIndex, level, rank.heal.Range(level))
	}
}

func TestHolyLightMatchesClient(t *testing.T) {
	client := clientdamagetest.Load(t, clientPaladinSpellconst)
	for _, rank := range holyLightRanks {
		assertHealRank(t, client, "Holy Light", rank.spellID, rank.spellID, rank)
		spell, _ := client.ByID(rank.spellID)
		if spell.CastTimeMS != int32(holyLightCastTime.Milliseconds()) {
			t.Errorf("Holy Light %d: engine cast time %v, client %dms", rank.spellID, holyLightCastTime, spell.CastTimeMS)
		}
		if got := spell.Effects[healEffectIndex].ResolvedSPCoefficient; got-holyLightCoefficient > 0.002 || holyLightCoefficient-got > 0.002 {
			t.Errorf("Holy Light %d: engine coefficient %v, client %v", rank.spellID, holyLightCoefficient, got)
		}
	}
	if len(holyLightRanks) != 9 {
		t.Errorf("Holy Light has %d ranks, the client teaches 9", len(holyLightRanks))
	}
}

func TestFlashOfLightMatchesClient(t *testing.T) {
	client := clientdamagetest.Load(t, clientPaladinSpellconst)
	for _, rank := range flashOfLightRanks {
		assertHealRank(t, client, "Flash of Light", rank.spellID, rank.spellID, rank)
		spell, _ := client.ByID(rank.spellID)
		if spell.CastTimeMS != int32(flashOfLightCastTime.Milliseconds()) {
			t.Errorf("Flash of Light %d: engine cast time %v, client %dms", rank.spellID, flashOfLightCastTime, spell.CastTimeMS)
		}
		if got := spell.Effects[healEffectIndex].ResolvedSPCoefficient; got-flashOfLightCoefficient > 0.002 || flashOfLightCoefficient-got > 0.002 {
			t.Errorf("Flash of Light %d: engine coefficient %v, client %v", rank.spellID, flashOfLightCoefficient, got)
		}
	}
	if len(flashOfLightRanks) != 6 {
		t.Errorf("Flash of Light has %d ranks, the client teaches 6", len(flashOfLightRanks))
	}
}

func TestHolyShockHealMatchesClient(t *testing.T) {
	client := clientdamagetest.Load(t, clientPaladinSpellconst)
	for _, rank := range holyShockRanks {
		assertHealRank(t, client, "Holy Shock", rank.castID, rank.healID, healRank{
			level: rank.level, manaCost: rank.manaCost, heal: rank.heal,
		})
		heal, _ := client.ByID(rank.healID)
		if got := heal.Effects[healEffectIndex].ResolvedSPCoefficient; got-holyShockHealCoefficient > 0.002 || holyShockHealCoefficient-got > 0.002 {
			t.Errorf("Holy Shock heal %d: engine coefficient %v, client %v", rank.healID, holyShockHealCoefficient, got)
		}
		cast, _ := client.ByID(rank.castID)
		if cast.EffectiveCooldownMS() != int32(holyShockCooldown.Milliseconds()) {
			t.Errorf("Holy Shock %d: engine cooldown %v, client %dms", rank.castID, holyShockCooldown, cast.EffectiveCooldownMS())
		}
	}
}

func TestLightsVigilHealMatchesClient(t *testing.T) {
	client := clientdamagetest.Load(t, clientPaladinSpellconst)
	for _, rank := range lightsVigilRanks {
		assertHealRank(t, client, "Light's Vigil", rank.castID, rank.healID, healRank{
			level: rank.level, manaCost: rank.manaCost, heal: rank.heal,
		})
		cast, _ := client.ByID(rank.castID)
		if cast.CastTimeMS != int32(lightsVigilCastTime.Milliseconds()) || cast.EffectiveCooldownMS() != int32(lightsVigilCooldown.Milliseconds()) {
			t.Errorf("Light's Vigil %d: engine cast %v / cooldown %v, client %dms / %dms", rank.castID,
				lightsVigilCastTime, lightsVigilCooldown, cast.CastTimeMS, cast.EffectiveCooldownMS())
		}
		heal, _ := client.ByID(rank.healID)
		if got := heal.Effects[healEffectIndex].ResolvedSPCoefficient; got-lightsVigilCoefficient > 0.002 || lightsVigilCoefficient-got > 0.002 {
			t.Errorf("Light's Vigil heal %d: engine coefficient %v, client %v", rank.healID, lightsVigilCoefficient, got)
		}
	}
}

func TestBlessingOfLightMatchesClient(t *testing.T) {
	client := clientdamagetest.Load(t, clientPaladinSpellconst)
	for _, rank := range append(append([]blessingOfLightRank{}, blessingOfLightRanks...), greaterBlessingOfLight) {
		spell, ok := client.ByID(rank.spellID)
		if !ok {
			t.Fatalf("Blessing of Light %d is not in the client table", rank.spellID)
		}
		if spell.SpellLevel != rank.level || spell.Cost != rank.manaCost {
			t.Errorf("Blessing of Light %d: engine level %d cost %v, client %d / %v", rank.spellID, rank.level, rank.manaCost, spell.SpellLevel, spell.Cost)
		}
		for heal, effect := range spell.Effects[:blessingOfLightHealKinds] {
			if float64(effect.Amount) != rank.bonus[heal] {
				t.Errorf("Blessing of Light %d effect %d: engine %v, client %v", rank.spellID, heal, rank.bonus[heal], effect.Amount)
			}
		}
		if spell.DurationMS != int32(blessingOfLightDuration.Milliseconds()) {
			t.Errorf("Blessing of Light %d: engine duration %v, client %dms", rank.spellID, blessingOfLightDuration, spell.DurationMS)
		}
	}
}
