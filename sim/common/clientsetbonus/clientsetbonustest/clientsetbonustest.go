// Package clientsetbonustest holds the assertion every Tier 1 set test
// shares: wearing n pieces adds exactly what the client's flat bonus rows
// state. It imports testing, so it stays out of the simulator binaries.
package clientsetbonustest

import (
	"slices"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// flatBonusesWorn is every flat bonus row of the set at or below pieces.
func flatBonusesWorn(t *testing.T, setID int32, pieces int) []core.ClientFlatBonus {
	t.Helper()
	row, ok := core.ClientSetRow(setID)
	if !ok {
		t.Fatalf("no client item set %d", setID)
	}
	var worn []core.ClientFlatBonus
	for _, bonus := range row.Bonuses {
		if int(bonus.Threshold) > pieces {
			continue
		}
		if flat, isFlat := core.DecodeClientFlatBonus(core.MustClientSpellRow(bonus.SpellID)); isFlat {
			worn = append(worn, flat)
		}
	}
	return worn
}

func expectedStats(worn []core.ClientFlatBonus) stats.Stats {
	var total stats.Stats
	for _, flat := range worn {
		total = total.Add(flat.Stats)
	}
	// Every character turns healing power into a third as much spell damage.
	total[stats.SpellDamage] += total[stats.HealingPower] * core.HealingToSpellDamageRatio
	return total
}

func expectedVs(worn []core.ClientFlatBonus, pick func(core.ClientFlatBonus) []core.MobTypeBonus, mobType proto.MobType) float64 {
	var total float64
	for _, flat := range worn {
		for _, bonus := range pick(flat) {
			if slices.Contains(bonus.MobTypes, mobType) {
				total += bonus.Amount
			}
		}
	}
	return total
}

func attackPowerVs(flat core.ClientFlatBonus) []core.MobTypeBonus { return flat.AttackPowerVs }
func spellDamageVs(flat core.ClientFlatBonus) []core.MobTypeBonus { return flat.SpellDamageVs }

// AssertAutomaticTotals fails unless worn has, over bare, the stats the rows
// name and the attack power and spell damage against its first target that the
// set's flat bonus rows give at the number of pieces worn. Both characters
// must be built the same way except for the gear, and past Reset.
func AssertAutomaticTotals(t *testing.T, setID int32, pieces int, bare, worn *core.Character) {
	t.Helper()
	rows := flatBonusesWorn(t, setID, pieces)

	want := expectedStats(rows)
	got := worn.GetStats().Subtract(bare.GetStats())
	for stat := range want {
		if want[stat] == 0 {
			continue // derived stats (defense gives block, dodge and parry) follow from the rows
		}
		if diff := got[stat] - want[stat]; diff > 1e-9 || diff < -1e-9 {
			t.Errorf("%d pieces of set %d change %s by %v, the rows say %v", pieces, setID, stats.Stat(stat).StatName(), got[stat], want[stat])
		}
	}

	target := worn.Env.Encounter.AllTargetUnits[0]
	assertVs(t, "attack power", expectedVs(rows, attackPowerVs, target.MobType),
		attackTableOf(worn, target).BonusAttackPowerTaken-attackTableOf(bare, target).BonusAttackPowerTaken)
	assertVs(t, "spell damage", expectedVs(rows, spellDamageVs, target.MobType),
		attackTableOf(worn, target).BonusSpellDamageTaken-attackTableOf(bare, target).BonusSpellDamageTaken)
}

func attackTableOf(character *core.Character, target *core.Unit) *core.AttackTable {
	return character.AttackTables[target.UnitIndex][proto.CastType_CastTypeMainHand]
}

func assertVs(t *testing.T, what string, want, got float64) {
	t.Helper()
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("bonus %s against the target is %v, the rows say %v", what, got, want)
	}
}
