package elemental

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Elemental tree node positions in the engine's talent string: the proto's
// field order (proto/shaman.proto, field number - 1), which is not the
// client JSON's tier-then-column order (Elemental Fury is field 8).
const (
	concussionNode           = 1
	elementalDevastationNode = 5
	lightningOverloadNode    = 12
	elementalFuryNode        = 7
)

// elementalTalentsWith is a talent string with only the named Elemental
// nodes set, the isolate-one-talent convention of callOfFlameTalentsString.
func elementalTalentsWith(points map[int]int) string {
	nodes := [16]byte{}
	for i := range nodes {
		nodes[i] = '0'
	}
	for node, rank := range points {
		nodes[node] = byte('0' + rank)
	}
	return string(nodes[:]) + "--"
}

func near(got, want float64) bool { return math.Abs(got-want) < 1e-9 }

// TestConcussionBoostsOnlyTheSpellsItNames: the client's text and class
// mask (spell 16035) name Lightning Bolt, Chain Lightning and Earth
// Shock. Flame Shock and Frost Shock must not draw the bonus.
func TestConcussionBoostsOnlyTheSpellsItNames(t *testing.T) {
	_, bare := newCallOfFlameShaman(t, elementalTalentsWith(nil))
	_, talented := newCallOfFlameShaman(t, elementalTalentsWith(map[int]int{concussionNode: 5}))

	const bonus = 0.05
	named := map[string][2]*core.Spell{
		"Lightning Bolt":  {bare.LightningBolt[10], talented.LightningBolt[10]},
		"Chain Lightning": {bare.ChainLightning[4], talented.ChainLightning[4]},
		"Earth Shock":     {bare.EarthShock[7], talented.EarthShock[7]},
	}
	for name, pair := range named {
		if got, want := pair[1].DamageMultiplierAdditive, pair[0].DamageMultiplierAdditive+bonus; !near(got, want) {
			t.Errorf("%s additive multiplier = %v, want %v with Concussion 5/5", name, got, want)
		}
	}

	unnamed := map[string][2]*core.Spell{
		"Flame Shock": {bare.FlameShock[6], talented.FlameShock[6]},
		"Frost Shock": {bare.FrostShock[4], talented.FrostShock[4]},
	}
	for name, pair := range unnamed {
		if got, want := pair[1].DamageMultiplierAdditive, pair[0].DamageMultiplierAdditive; !near(got, want) {
			t.Errorf("%s additive multiplier = %v with Concussion, want the untalented %v: the client's text does not name it", name, got, want)
		}
	}
}

// TestElementalFuryScalesWithItsRank: 20% more crit damage bonus per
// rank, five ranks. The Classic port gave every rank the full +100%.
func TestElementalFuryScalesWithItsRank(t *testing.T) {
	_, bare := newCallOfFlameShaman(t, elementalTalentsWith(nil))
	baseline := bare.LightningBolt[10].CritDamageBonus

	for rank := 1; rank <= 5; rank++ {
		_, talented := newCallOfFlameShaman(t, elementalTalentsWith(map[int]int{elementalFuryNode: rank}))
		got := talented.LightningBolt[10].CritDamageBonus - baseline
		if want := 0.2 * float64(rank); !near(got, want) {
			t.Errorf("Elemental Fury %d/5: Lightning Bolt crit damage bonus +%v, want +%v", rank, got, want)
		}
	}
}

// TestElementalDevastationIsMeleeOnly: "your offensive spell critical
// strikes will increase your chance to get a critical strike with MELEE
// attacks". Forever's Crit stat is unified, so the proc must raise the
// stat (melee crit) while taking the same amount back off the shaman's
// own spell crit.
func TestElementalDevastationIsMeleeOnly(t *testing.T) {
	sim, built := newCallOfFlameShaman(t, elementalTalentsWith(map[int]int{elementalDevastationNode: 3}))
	target := sim.Encounter.TargetUnits[0]
	lightningBolt := built.LightningBolt[10]

	procAura := built.GetAura("Elemental Devastation Proc")
	if procAura == nil {
		t.Fatal("no Elemental Devastation Proc aura registered")
	}

	spellCritBefore := lightningBolt.SpellCritChance(target)
	meleeCritBefore := built.GetStat(stats.Crit)

	procAura.Activate(sim)

	if got := lightningBolt.SpellCritChance(target); !near(got, spellCritBefore) {
		t.Errorf("spell crit chance %v while Elemental Devastation is up, want the unchanged %v", got, spellCritBefore)
	}
	wantMeleeBonus := 9.0 * core.CritRatingPerCritChance
	if got := built.GetStat(stats.Crit) - meleeCritBefore; !near(got, wantMeleeBonus) {
		t.Errorf("melee crit rating gained %v, want %v (9%% at 3/3)", got, wantMeleeBonus)
	}

	procAura.Deactivate(sim)
	if got := lightningBolt.SpellCritChance(target); !near(got, spellCritBefore) {
		t.Errorf("spell crit chance %v after the proc fades, want %v", got, spellCritBefore)
	}
}

// TestLightningOverloadHalvesTheWholeHit: the client's overload spells
// carry half the amount AND half the spell-power coefficient (408477:
// 98 at 0.357 against 196 at 0.714), so the second hit is the whole
// primary at half the damage multiplier, not half of its base damage
// alone. The scale helper must halve the multiplier for the hit it wraps
// and restore it after.
func TestLightningOverloadHalvesTheWholeHit(t *testing.T) {
	_, built := newCallOfFlameShaman(t, elementalTalentsWith(map[int]int{lightningOverloadNode: 3}))
	lightningBolt := built.LightningBolt[10]

	full := lightningBolt.DamageMultiplier
	var scaled float64
	built.AtLightningOverloadScale(lightningBolt, func() {
		scaled = lightningBolt.DamageMultiplier
	})

	if !near(scaled, full/2) {
		t.Errorf("overload hit damage multiplier %v, want half of %v", scaled, full)
	}
	if !near(lightningBolt.DamageMultiplier, full) {
		t.Errorf("damage multiplier %v after the overload, want the restored %v", lightningBolt.DamageMultiplier, full)
	}
}
