package paladin

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/spellconst"
)

type blessingClientCase struct {
	name   string
	spells []blessingSpell
	// ranked blessings are numbered 1..n in the client.
	ranked bool
	// effectSign turns the client's effect amount into the engine's
	// positive amount (Salvation states -30).
	effectSign float64
}

func blessingClientCases() []blessingClientCase {
	return []blessingClientCase{
		{"Blessing of Might", blessingOfMightSpells, true, 1},
		{"Greater Blessing of Might", greaterBlessingOfMightSpells, true, 1},
		{"Blessing of Wisdom", blessingOfWisdomSpells, true, 1},
		{"Greater Blessing of Wisdom", greaterBlessingOfWisdomSpells, true, 1},
		{"Blessing of Kings", blessingOfKingsSpells, false, 1},
		{"Greater Blessing of Kings", greaterBlessingOfKingsSpells, false, 1},
		{"Blessing of Salvation", blessingOfSalvationSpells, false, -1},
		{"Greater Blessing of Salvation", greaterBlessingOfSalvationSpells, false, -1},
	}
}

func assertBlessingSpell(t *testing.T, client spellconst.Class, name string, spell blessingSpell, wantRank int, effectSign float64) {
	t.Helper()
	row, ok := client.ByID(spell.spellID)
	if !ok {
		t.Fatalf("%s: spell %d is not in the client table", name, spell.spellID)
	}
	if row.SpellLevel != spell.level {
		t.Errorf("%s (%d): engine level %d, client %d", name, spell.spellID, spell.level, row.SpellLevel)
	}
	if row.Cost != spell.manaCost || row.CostPct != spell.baseManaPct {
		t.Errorf("%s (%d): engine cost %v / %v%%, client %v / %v%%", name, spell.spellID,
			spell.manaCost, spell.baseManaPct, row.Cost, row.CostPct)
	}
	if row.DurationMS != int32(blessingDuration.Milliseconds()) {
		t.Errorf("%s (%d): engine duration %v, client %dms", name, spell.spellID, blessingDuration, row.DurationMS)
	}
	if row.Rank != wantRank {
		t.Errorf("%s (%d): engine rank %d, client %d", name, spell.spellID, wantRank, row.Rank)
	}
	if got := row.Effects[0].Amount * effectSign; math.Abs(got-spell.amount) > 1e-9 {
		t.Errorf("%s (%d): engine amount %v, client %v", name, spell.spellID, spell.amount, got)
	}
}

func TestBlessingsMatchClient(t *testing.T) {
	client := clientdamagetest.Load(t, clientPaladinSpellconst)
	for _, c := range blessingClientCases() {
		for i, spell := range c.spells {
			wantRank := 0
			if c.ranked {
				wantRank = i + 1
			}
			assertBlessingSpell(t, client, c.name, spell, wantRank, c.effectSign)
		}
	}
}

func TestBlessingRankCounts(t *testing.T) {
	counts := map[string]int{
		"Blessing of Might": 7, "Greater Blessing of Might": 2,
		"Blessing of Wisdom": 6, "Greater Blessing of Wisdom": 2,
		"Blessing of Kings": 1, "Greater Blessing of Kings": 1,
		"Blessing of Salvation": 1, "Greater Blessing of Salvation": 1,
	}
	for _, c := range blessingClientCases() {
		if len(c.spells) != counts[c.name] {
			t.Errorf("%s has %d spells, the client teaches %d", c.name, len(c.spells), counts[c.name])
		}
	}
}

func TestBlessingManaCostOptions(t *testing.T) {
	flat := blessingSpell{manaCost: 130}.manaCostOptions()
	if flat.FlatCost != 130 || flat.BaseCost != 0 {
		t.Errorf("flat cost options %+v", flat)
	}
	percent := blessingSpell{baseManaPct: 8}.manaCostOptions()
	if percent.FlatCost != 0 || math.Abs(percent.BaseCost-0.08) > 1e-12 {
		t.Errorf("percent cost options %+v", percent)
	}
}

func TestRighteousFuryMatchesClient(t *testing.T) {
	client := clientdamagetest.Load(t, clientPaladinSpellconst)
	row, ok := client.ByID(righteousFuryActionID)
	if !ok {
		t.Fatalf("Righteous Fury %d is not in the client table", righteousFuryActionID)
	}
	if row.SpellLevel != righteousFuryLevel {
		t.Errorf("engine level %d, client %d", righteousFuryLevel, row.SpellLevel)
	}
	if want := righteousFuryBaseManaCost * 100; math.Abs(row.CostPct-want) > 1e-9 {
		t.Errorf("engine cost %v%% of base mana, client %v%%", want, row.CostPct)
	}
	if row.DurationMS != int32(righteousFuryDuration.Milliseconds()) {
		t.Errorf("engine duration %v, client %dms", righteousFuryDuration, row.DurationMS)
	}
}

func TestInstrumentOfLawPenaltyIsLiftedWhileRighteousFuryIsActive(t *testing.T) {
	unit := &core.Unit{}
	unit.PseudoStats.ThreatMultiplier = 1
	penalty := &instrumentOfLawPenalty{unit: unit, multiplier: 0.8}

	penalty.set(true)
	penalty.set(true)
	if got := unit.PseudoStats.ThreatMultiplier; math.Abs(got-0.8) > 1e-12 {
		t.Errorf("penalty applied twice or not at all: threat multiplier %v, want 0.8", got)
	}
	penalty.set(false)
	penalty.set(false)
	if got := unit.PseudoStats.ThreatMultiplier; math.Abs(got-1) > 1e-12 {
		t.Errorf("penalty not lifted: threat multiplier %v, want 1", got)
	}
}
