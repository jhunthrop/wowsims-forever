package druid

import (
	"fmt"
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/spellconst"
)

// healingTable pairs a table with the client name and the index of the
// client effects its heal and its tick read.
type healingTable struct {
	name       string
	ranks      []healingRank
	healEffect int
	tickEffect int
}

var healingTables = []healingTable{
	{"Healing Touch", healingTouchTable, 0, -1},
	{"Regrowth", regrowthTable, 0, 1},
	{"Rejuvenation", rejuvenationTable, -1, 0},
	{"Tranquility", tranquilityTable, -1, 0},
	{"Wild Growth", wildGrowthTable, -1, 0},
}

// checkLevels are the caster levels every rank is compared at.
var checkLevels = []int{1, 10, 20, 30, 40, 50, 60}

const (
	rollTolerance        = 0.05
	coefficientTolerance = 0.002
)

func assertEffect(t *testing.T, label string, client spellconst.Spell, index int, got healingEffect) {
	t.Helper()
	if got.roll.SpellLevel != client.SpellLevel || got.roll.MaxLevel != client.MaxLevel {
		t.Errorf("%s: spell level %d / max %d, client %d / %d", label, got.roll.SpellLevel, got.roll.MaxLevel, client.SpellLevel, client.MaxLevel)
	}
	if math.Abs(got.coefficient-client.Effects[index].ResolvedSPCoefficient) > coefficientTolerance {
		t.Errorf("%s: coefficient %v, client %v", label, got.coefficient, client.Effects[index].ResolvedSPCoefficient)
	}
	for _, level := range checkLevels {
		wantMin, wantMax, _ := client.DamageRange(index, level)
		have := got.roll.Range(level)
		if math.Abs(have[0]-wantMin) > rollTolerance || math.Abs(have[1]-wantMax) > rollTolerance {
			t.Errorf("%s at level %d: rolls %.2f-%.2f, client %.2f-%.2f", label, level, have[0], have[1], wantMin, wantMax)
		}
	}
}

func TestHealingTablesMatchTheClient(t *testing.T) {
	class, err := spellconst.Load("../core/testdata/conformance/client/druid.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range healingTables {
		for index, rank := range table.ranks {
			client, ok := class.ByID(rank.spellID)
			if !ok {
				t.Fatalf("%s rank %d: spell %d is not in the client table", table.name, index+1, rank.spellID)
			}
			label := fmt.Sprintf("%s rank %d", table.name, index+1)
			if client.Name != table.name || client.Rank != index+1 {
				t.Errorf("%s: client spell %d is %s rank %d", label, rank.spellID, client.Name, client.Rank)
			}
			if rank.manaCost != client.Cost || rank.level() != client.SpellLevel {
				t.Errorf("%s: cost %v level %d, client %v / %d", label, rank.manaCost, rank.level(), client.Cost, client.SpellLevel)
			}
			if got, want := rank.castTime.Milliseconds(), int64(client.CastTimeMS); got != want {
				t.Errorf("%s: cast time %dms, client %dms", label, got, want)
			}
			if table.healEffect >= 0 {
				assertEffect(t, label+" heal", client, table.healEffect, rank.heal)
			}
			if table.tickEffect >= 0 {
				assertEffect(t, label+" tick", client, table.tickEffect, rank.tick)
			}
		}
	}
}

// TestHealingTablesLeaveOutTheReissues: the client's second Regrowth and
// Rejuvenation ids carry no SkillLineAbility row, so no trainer teaches them.
func TestHealingTablesLeaveOutTheReissues(t *testing.T) {
	reissued := map[int32]bool{}
	for id := int32(436937); id <= 436946; id++ {
		reissued[id] = true
	}
	for id := int32(417057); id <= 417068; id++ {
		reissued[id] = true
	}
	for _, table := range healingTables {
		for _, rank := range table.ranks {
			if reissued[rank.spellID] {
				t.Errorf("%s has the unlearnable reissue %d", table.name, rank.spellID)
			}
		}
	}
}
