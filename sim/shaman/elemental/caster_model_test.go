package elemental

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/shaman"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// elementalNode is a talent's position in the Elemental segment of the
// engine's talent string: the generated proto field's number - 1, looked
// up by name so a tree layout change (the live tree moved Elemental
// Alacrity and Elemental Fury) cannot leave a typed index pointing at
// another talent. The Elemental tree is the first segment, so the field
// position is the index.
func elementalNode(field string) int {
	fd := (&proto.ShamanTalents{}).ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(field))
	if fd == nil {
		panic("ShamanTalents has no field " + field)
	}
	return int(fd.Number()) - 1
}

var (
	concussionNode           = elementalNode("concussion")
	elementalDevastationNode = elementalNode("elemental_devastation")
	lightningOverloadNode    = elementalNode("lightning_overload")
	elementalFuryNode        = elementalNode("elemental_fury")
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
// attacks". Spell 30165 is aura 52, the melee and ranged table: the proc
// raises melee crit and leaves the shaman's spell crit alone.
func TestElementalDevastationIsMeleeOnly(t *testing.T) {
	sim, built := newCallOfFlameShaman(t, elementalTalentsWith(map[int]int{elementalDevastationNode: 3}))
	target := sim.Encounter.TargetUnits[0]
	lightningBolt := built.LightningBolt[10]

	procAura := built.GetAura("Elemental Devastation Proc")
	if procAura == nil {
		t.Fatal("no Elemental Devastation Proc aura registered")
	}

	spellCritBefore := lightningBolt.SpellCritChance(target)
	meleeCritBefore := built.GetStat(stats.MeleeCrit)

	procAura.Activate(sim)

	if got := lightningBolt.SpellCritChance(target); !near(got, spellCritBefore) {
		t.Errorf("spell crit chance %v while Elemental Devastation is up, want the unchanged %v", got, spellCritBefore)
	}
	wantMeleeBonus := 9.0 * core.CritRatingPerCritChance
	if got := built.GetStat(stats.MeleeCrit) - meleeCritBefore; !near(got, wantMeleeBonus) {
		t.Errorf("melee crit rating gained %v, want %v (9%% at 3/3)", got, wantMeleeBonus)
	}

	procAura.Deactivate(sim)
	if got := lightningBolt.SpellCritChance(target); !near(got, spellCritBefore) {
		t.Errorf("spell crit chance %v after the proc fades, want %v", got, spellCritBefore)
	}
}

// TestLightningOverloadUsesItsOwnCoefficient: the overload is the client's
// own per-rank spell (408477 for Lightning Bolt rank 10: 98 + 0.6 a level
// at 0.357), which client 1.60.1.70755 left alone while it raised the bolt
// to 353 + 2.1 at 0.714. The helper must swap in the overload's
// coefficient for the hit it wraps, leave the damage multiplier alone, and
// restore the bolt's coefficient after.
func TestLightningOverloadUsesItsOwnCoefficient(t *testing.T) {
	_, built := newCallOfFlameShaman(t, elementalTalentsWith(map[int]int{lightningOverloadNode: 3}))
	lightningBolt := built.LightningBolt[10]

	coefficient, multiplier := lightningBolt.BonusCoefficient, lightningBolt.DamageMultiplier
	var during, multiplierDuring float64
	built.AtLightningOverloadCoefficient(lightningBolt, shaman.LightningBoltOverloadSpellCoef[10], func() {
		during, multiplierDuring = lightningBolt.BonusCoefficient, lightningBolt.DamageMultiplier
	})

	if !near(during, 0.357) {
		t.Errorf("overload hit coefficient %v, want the client's 0.357", during)
	}
	if !near(multiplierDuring, multiplier) {
		t.Errorf("overload hit damage multiplier %v, want the primary's %v", multiplierDuring, multiplier)
	}
	if !near(lightningBolt.BonusCoefficient, coefficient) {
		t.Errorf("coefficient %v after the overload, want the restored %v", lightningBolt.BonusCoefficient, coefficient)
	}
}

// spellByID is the shaman's registered spell with the given client id.
func spellByID(t *testing.T, built *shaman.Shaman, id int32) *core.Spell {
	t.Helper()
	spell := built.GetSpell(core.ActionID{SpellID: id})
	if spell == nil {
		t.Fatalf("no spell %d registered", id)
	}
	return spell
}

// TestElementalFuryReachesTheTotemsDamage: the client's text and class mask
// (spell 16089) name "your Searing and Magma Totems": the Searing Totem's
// Attack spells (3606 .. 10436) and Magma Totem's pulse (8187 .. 10581)
// carry the mask bit (0x40000000). The damage is dealt by those child
// spells, not by the totem-drop spell, so the bonus has to reach them.
func TestElementalFuryReachesTheTotemsDamage(t *testing.T) {
	_, bare := newCallOfFlameShaman(t, elementalTalentsWith(nil))

	for rank := 1; rank <= 5; rank++ {
		_, talented := newCallOfFlameShaman(t, elementalTalentsWith(map[int]int{elementalFuryNode: rank}))
		want := 0.2 * float64(rank)

		for name, id := range map[string]int32{
			"Searing Totem attack": shaman.SearingTotemAttackSpellId[shaman.SearingTotemRanks],
			"Magma Totem pulse":    shaman.MagmaTotemAoeSpellId[shaman.MagmaTotemRanks],
		} {
			got := spellByID(t, talented, id).CritDamageBonus - spellByID(t, bare, id).CritDamageBonus
			if !near(got, want) {
				t.Errorf("Elemental Fury %d/5: %s crit damage bonus +%v, want +%v", rank, name, got, want)
			}
		}
	}
}

// TestADownrankedBoltsOverloadTakesTheLevelGapShare: the overload is the
// cast rank's own spell, so the swapped-in coefficient takes Forever's
// level-gap rule (core.LevelPenalty) at the cast rank's learn level, as the
// bolt's own coefficient does. Lightning Bolt rank 4 (learned at 20) cast at
// 60 is 40 levels behind and keeps none of it; rank 10 keeps all.
func TestADownrankedBoltsOverloadTakesTheLevelGapShare(t *testing.T) {
	_, built := newCallOfFlameShaman(t, elementalTalentsWith(map[int]int{lightningOverloadNode: 3}))
	for _, c := range []struct {
		rank int
		want float64
	}{{4, 0}, {10, 0.357}} {
		bolt := built.LightningBolt[c.rank]
		var during float64
		built.AtLightningOverloadCoefficient(bolt, shaman.LightningBoltOverloadSpellCoef[c.rank], func() { during = bolt.BonusCoefficient })
		want := shaman.LightningBoltOverloadSpellCoef[c.rank] * core.LevelPenalty(bolt.RequiredLevel, int(built.Level))
		if !near(during, want) || (c.want == 0 && during != 0) || (c.want > 0 && !near(during, c.want)) {
			t.Errorf("rank %d overload coefficient %v, want %v", c.rank, during, want)
		}
	}
}
