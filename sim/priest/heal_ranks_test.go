package priest

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/spellconst"
)

const (
	clientPriestPath     = "../core/testdata/conformance/client/priest.json"
	clientTrainablesPath = "../core/testdata/conformance/client/trainables/priest.json"

	// The client's effect codes the ladders read.
	effectSchoolDamage = 2
	effectHeal         = 10
	effectDummy        = 3
	auraAbsorb         = 69
	auraPeriodicHeal   = 8
	auraArmor          = 22
)

// healingLadder is one table of heal ranks and the client effect it
// states.
type healingLadder struct {
	name   string
	table  []healRank
	effect int32
	aura   int32
}

func healingLadders() []healingLadder {
	return []healingLadder{
		{"Lesser Heal", lesserHealRanks, effectHeal, 0},
		{"Heal", healRanks, effectHeal, 0},
		{"Flash Heal", flashHealRanks, effectHeal, 0},
		{"Greater Heal", greaterHealRanks, effectHeal, 0},
		{"Binding Heal", bindingHealRanks, effectHeal, 0},
		{"Desperate Prayer", desperatePrayerRanks, effectHeal, 0},
		{"Prayer of Healing", prayerOfHealingRanks, effectHeal, 0},
		{"Renew", renewRanks, 6, auraPeriodicHeal},
		{"Power Word: Shield", powerWordShieldRanks, 6, auraAbsorb},
		{"Holy Nova damage", holyNovaDamageRanks, effectSchoolDamage, 0},
		{"Holy Nova heal", holyNovaHealRanks, effectHeal, 0},
		{"Penance", penanceRanks, effectHeal, 0},
		{"Prayer of Mending", prayerOfMendingRanks, effectDummy, 0},
		{"Inner Fire", innerFireRanks, 6, auraArmor},
	}
}

func loadClient(t *testing.T) spellconst.Class {
	t.Helper()
	client, err := spellconst.Load(clientPriestPath)
	if err != nil {
		t.Fatalf("loading the client's priest spellconst: %v", err)
	}
	return client
}

// effectOf is the client effect a ladder describes on the spell that
// carries it.
func effectOf(t *testing.T, ladder healingLadder, spell spellconst.Spell) spellconst.Effect {
	t.Helper()
	for _, effect := range spell.Effects {
		if effect.Effect == ladder.effect && effect.Aura == ladder.aura {
			return effect
		}
	}
	t.Fatalf("%s (%d): the client states no effect %d aura %d", ladder.name, spell.ID, ladder.effect, ladder.aura)
	return spellconst.Effect{}
}

// TestHealingLaddersMatchTheClient pins every rank of every healing table
// to the vendored client file: level, cost, cast time, coefficient and the
// amount at several caster levels (so growth per level and its cap are
// both exercised).
func TestHealingLaddersMatchTheClient(t *testing.T) {
	client := loadClient(t)
	for _, ladder := range healingLadders() {
		for i, rank := range ladder.table {
			cast, ok := client.ByID(rank.spellID)
			if !ok {
				t.Fatalf("%s rank %d: spell %d is not in the client table", ladder.name, i+1, rank.spellID)
			}
			if cast.SpellLevel != rank.level {
				t.Errorf("%s rank %d (%d): level %d, client %d", ladder.name, i+1, rank.spellID, rank.level, cast.SpellLevel)
			}
			if math.Abs(cast.Cost-rank.manaCost) > 0.5 {
				t.Errorf("%s rank %d (%d): mana %v, client %v", ladder.name, i+1, rank.spellID, rank.manaCost, cast.Cost)
			}
			if ladder.name != "Holy Nova heal" && cast.CastTimeMS != rank.castMS {
				t.Errorf("%s rank %d (%d): cast %d ms, client %d ms", ladder.name, i+1, rank.spellID, rank.castMS, cast.CastTimeMS)
			}
			assertAmount(t, client, ladder, i+1, rank)
		}
	}
}

func assertAmount(t *testing.T, client spellconst.Class, ladder healingLadder, rankNumber int, rank healRank) {
	t.Helper()
	carrierID := rank.spellID
	if rank.effectSpellID != 0 {
		carrierID = rank.effectSpellID
	}
	carrier, ok := client.ByID(carrierID)
	if !ok {
		t.Fatalf("%s rank %d: spell %d is not in the client table", ladder.name, rankNumber, carrierID)
	}
	effect := effectOf(t, ladder, carrier)
	if effect.CoefficientSource == "table" && math.Abs(effect.ResolvedSPCoefficient-rank.coefficient) > 0.002 {
		t.Errorf("%s rank %d (%d): coefficient %v, client %v", ladder.name, rankNumber, carrierID, rank.coefficient, effect.ResolvedSPCoefficient)
	}
	for _, casterLevel := range []int{1, 10, 20, 30, 40, 50, 55, 60} {
		wantLow, wantHigh, _ := carrier.DamageRange(effect.Index, casterLevel)
		got := rank.effect.Range(casterLevel)
		if math.Abs(got[0]-wantLow) > 0.01 || math.Abs(got[1]-wantHigh) > 0.01 {
			t.Errorf("%s rank %d (%d) at level %d: rolls %.3f-%.3f, client %.3f-%.3f", ladder.name, rankNumber, carrierID, casterLevel, got[0], got[1], wantLow, wantHigh)
		}
	}
}

// trainableIDs reads the ids a priest's trainer teaches, per ability name.
func trainableIDs(t *testing.T) map[int32]bool {
	t.Helper()
	raw, err := os.ReadFile(clientTrainablesPath)
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Trainables []struct {
			Ranks []struct {
				ID int32 `json:"id"`
			} `json:"ranks"`
		} `json:"trainables"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	ids := map[int32]bool{}
	for _, trainable := range file.Trainables {
		for _, rank := range trainable.Ranks {
			ids[rank.ID] = true
		}
	}
	return ids
}

// notTrainerRanks are the ladders whose ids are not trainer rows: the heal
// half of Holy Nova is a helper spell its cast spell triggers, and
// Prayer of Mending's ranks come with its talent spell.
var notTrainerRanks = map[string]bool{"Holy Nova heal": true, "Prayer of Mending": true}

// TestLearnableHealsAreTheOnesTheTrainerTeaches checks that every ranked
// heal a level-60 priest casts is a spell the trainer lists. Lesser Heal
// rank 1 is the one exception: priests start knowing it, so it has no
// trainer row.
func TestLearnableHealsAreTheOnesTheTrainerTeaches(t *testing.T) {
	taught := trainableIDs(t)
	const startingSpell = int32(2050)
	for _, ladder := range healingLadders() {
		if notTrainerRanks[ladder.name] {
			continue
		}
		for i, rank := range ladder.table {
			if rank.spellID != startingSpell && !taught[rank.spellID] {
				t.Errorf("%s rank %d: spell %d is not taught by the trainer", ladder.name, i+1, rank.spellID)
			}
		}
	}
}

// TestHotAndShieldDurationsFollowTheClient pins the constants the spells
// hard-code to the client's durations, periods and cooldowns.
func TestHotAndShieldDurationsFollowTheClient(t *testing.T) {
	client := loadClient(t)
	renew, _ := client.ByID(renewRanks[len(renewRanks)-1].spellID)
	if got := time.Duration(renew.DurationMS) / time.Duration(renew.Effects[0].PeriodMS); int(got) != renewTicks {
		t.Errorf("Renew ticks %d, client duration/period %d", renewTicks, got)
	}
	if time.Duration(renew.Effects[0].PeriodMS)*time.Millisecond != renewTickLength {
		t.Errorf("Renew tick length %v, client %d ms", renewTickLength, renew.Effects[0].PeriodMS)
	}
	shield, _ := client.ByID(powerWordShieldRanks[0].spellID)
	if time.Duration(shield.DurationMS)*time.Millisecond != powerWordShieldDuration {
		t.Errorf("Power Word: Shield lasts %v, client %d ms", powerWordShieldDuration, shield.DurationMS)
	}
	if time.Duration(shield.CategoryCooldownMS)*time.Millisecond != powerWordShieldCooldown {
		t.Errorf("Power Word: Shield cooldown %v, client %d ms", powerWordShieldCooldown, shield.CategoryCooldownMS)
	}
	weakened, _ := client.ByID(weakenedSoulSpellID)
	if time.Duration(weakened.DurationMS)*time.Millisecond != weakenedSoulDuration {
		t.Errorf("Weakened Soul lasts %v, client %d ms", weakenedSoulDuration, weakened.DurationMS)
	}
	prayer, _ := client.ByID(desperatePrayerRanks[0].spellID)
	if time.Duration(prayer.CategoryCooldownMS)*time.Millisecond != desperatePrayerCooldown {
		t.Errorf("Desperate Prayer cooldown %v, client %d ms", desperatePrayerCooldown, prayer.CategoryCooldownMS)
	}
	penance, _ := client.ByID(penanceRanks[0].spellID)
	if time.Duration(penance.CategoryCooldownMS)*time.Millisecond != penanceCooldown {
		t.Errorf("Penance cooldown %v, client %d ms", penanceCooldown, penance.CategoryCooldownMS)
	}
	mending, _ := client.ByID(prayerOfMendingRanks[0].spellID)
	if time.Duration(mending.CategoryCooldownMS)*time.Millisecond != prayerOfMendingCooldown {
		t.Errorf("Prayer of Mending cooldown %v, client %d ms", prayerOfMendingCooldown, mending.CategoryCooldownMS)
	}
	if got := int(mending.Effects[1].Amount); got != prayerOfMendingCharges {
		t.Errorf("Prayer of Mending charges %d, client %d", prayerOfMendingCharges, got)
	}
	innerFire, _ := client.ByID(innerFireRanks[0].spellID)
	if time.Duration(innerFire.DurationMS)*time.Millisecond != innerFireDuration {
		t.Errorf("Inner Fire lasts %v, client %d ms", innerFireDuration, innerFire.DurationMS)
	}
}
