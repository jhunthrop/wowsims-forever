package shaman

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
	"github.com/wowsims/classic/sim/core/spellconst"
)

// healCheckLevels are the caster levels every healing rank's roll is
// compared at: below, at and above each rank's own level, to the cap.
var healCheckLevels = []int{1, 10, 20, 30, 40, 50, 60}

const (
	directHealEffect = 0
	riptideTickIndex = 1
	coefficientSlack = 0.002
)

// assertRanksMatchClient checks one family's ranks against the vendored
// client constants: the spell exists, and its learn level, mana cost, cast
// time, coefficient and healing roll at every checked level agree.
func assertRanksMatchClient(t *testing.T, class spellconst.Class, name string, ranks []healingRank) {
	t.Helper()

	for _, rank := range ranks {
		client, ok := class.ByID(rank.spellID)
		if !ok {
			t.Fatalf("%s rank %d (%d) is not in the client table", name, rank.rank, rank.spellID)
		}
		if client.SpellLevel != rank.level {
			t.Errorf("%s rank %d: level %d, client %d", name, rank.rank, rank.level, client.SpellLevel)
		}
		if client.Cost != rank.manaCost {
			t.Errorf("%s rank %d: cost %v, client %v", name, rank.rank, rank.manaCost, client.Cost)
		}
		if got := time.Duration(client.CastTimeMS) * time.Millisecond; got != rank.castTime {
			t.Errorf("%s rank %d: cast time %v, client %v", name, rank.rank, rank.castTime, got)
		}
		if got := client.Effects[directHealEffect].SPCoefficient; got-rank.coefficient > coefficientSlack || rank.coefficient-got > coefficientSlack {
			t.Errorf("%s rank %d: coefficient %v, client %v", name, rank.rank, rank.coefficient, got)
		}
		for _, level := range healCheckLevels {
			clientdamagetest.AssertRoll(t, class, name, rank.spellID, directHealEffect, level, rank.healing.Range(level))
		}
	}
}

func TestDirectHealTablesMatchTheClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientShamanSpellconst)

	assertRanksMatchClient(t, class, "Healing Wave", healingWaveRanks())
	assertRanksMatchClient(t, class, "Lesser Healing Wave", lesserHealingWaveRanks())
	assertRanksMatchClient(t, class, "Chain Heal", chainHealRanks())
	assertRanksMatchClient(t, class, "Riptide", riptideRanks())
}

func TestRiptideTickTableMatchesTheClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientShamanSpellconst)

	for rank := 1; rank <= RiptideRanks; rank++ {
		id := RiptideSpellId[rank]
		for _, level := range healCheckLevels {
			clientdamagetest.AssertRoll(t, class, "Riptide tick", id, riptideTickIndex, level, riptideTickHealing[rank].Range(level))
		}
		client, _ := class.ByID(id)
		tick := client.Effects[riptideTickIndex]
		if tick.Aura != 8 || tick.PeriodMS != int32(riptideTickGap/time.Millisecond) {
			t.Errorf("Riptide rank %d effect 1 is aura %d every %d ms, want a periodic heal every %v", rank, tick.Aura, tick.PeriodMS, riptideTickGap)
		}
		if tick.SPCoefficient-riptideTickCoefficient > coefficientSlack || riptideTickCoefficient-tick.SPCoefficient > coefficientSlack {
			t.Errorf("Riptide rank %d tick coefficient = %v, client %v", rank, riptideTickCoefficient, tick.SPCoefficient)
		}
		if got := time.Duration(client.DurationMS) * time.Millisecond; got != riptideTicks*riptideTickGap {
			t.Errorf("Riptide rank %d lasts %v, client %v", rank, riptideTicks*riptideTickGap, got)
		}
		bonus := client.Effects[2]
		if bonus.Amount != riptideChainHealBonusPercent {
			t.Errorf("Riptide rank %d Chain Heal bonus = %v, client %v", rank, float64(riptideChainHealBonusPercent), bonus.Amount)
		}
	}
}

func TestLesserHealingWaveRankSixIsTheLearnableSpell(t *testing.T) {
	class := clientdamagetest.Load(t, clientShamanSpellconst)

	ranks := lesserHealingWaveRanks()
	top := ranks[len(ranks)-1]
	if top.spellID != lesserHealingWaveTopRankSpellID {
		t.Errorf("rank 6 is %d, want %d", top.spellID, lesserHealingWaveTopRankSpellID)
	}
	other, ok := class.ByID(LesserHealingWaveSpellId[LesserHealingWaveRanks])
	if !ok || other.Rank != lesserHealingWaveTopRank {
		t.Errorf("the generated table's rank 6 should be the other client rank-6 spell, got %+v", other)
	}
}

func TestHealingStreamAndManaTideTablesMatchTheClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientShamanSpellconst)

	for rank := 1; rank <= HealingStreamTotemRanks; rank++ {
		totem, _ := class.ByID(HealingStreamTotemSpellId[rank])
		heal, ok := class.ByID(HealingStreamSpellId[rank])
		if !ok {
			t.Fatalf("Healing Stream rank %d (%d) is not in the client table", rank, HealingStreamSpellId[rank])
		}
		if totem.Cost != HealingStreamTotemManaCost[rank] || totem.SpellLevel != HealingStreamTotemLevel[rank] {
			t.Errorf("Healing Stream Totem rank %d: cost %v level %d, client %v and %d", rank, HealingStreamTotemManaCost[rank], HealingStreamTotemLevel[rank], totem.Cost, totem.SpellLevel)
		}
		if got := time.Duration(totem.DurationMS) * time.Millisecond; got != healingStreamTotemDuration {
			t.Errorf("Healing Stream Totem rank %d lasts %v, client %v", rank, healingStreamTotemDuration, got)
		}
		effect := heal.Effects[0]
		if effect.Amount != HealingStreamBaseDamage[rank][0] || time.Duration(effect.PeriodMS)*time.Millisecond != healingStreamTotemTickGap {
			t.Errorf("Healing Stream rank %d heals %v every %d ms, client %v every %v", rank, HealingStreamBaseDamage[rank][0], effect.PeriodMS, effect.Amount, healingStreamTotemTickGap)
		}
	}

	for rank := 1; rank <= ManaTideTotemRanks; rank++ {
		totem, ok := class.ByID(ManaTideTotemSpellId[rank])
		if !ok {
			t.Fatalf("Mana Tide Totem rank %d is not in the client table", rank)
		}
		if got := time.Duration(totem.DurationMS) * time.Millisecond; got != manaTideTotemDuration {
			t.Errorf("Mana Tide Totem rank %d lasts %v, client %v", rank, manaTideTotemDuration, got)
		}
		if got := time.Duration(totem.EffectiveCooldownMS()) * time.Millisecond; got != manaTideTotemCooldown {
			t.Errorf("Mana Tide Totem rank %d cooldown %v, client %v", rank, manaTideTotemCooldown, got)
		}
		tick, _ := class.ByID(ManaTideSpellId[rank])
		if got := tick.Effects[0]; got.Amount != ManaTideBaseDamage[rank][0] || time.Duration(got.PeriodMS)*time.Millisecond != manaTideTotemTickGap {
			t.Errorf("Mana Tide rank %d restores %v every %d ms, client says %v every %v", rank, ManaTideBaseDamage[rank][0], got.PeriodMS, got.Amount, manaTideTotemTickGap)
		}
	}
}
