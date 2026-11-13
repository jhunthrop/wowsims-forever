package shaman_test

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/shaman"
)

// Blizzard's 9 and 10 November hotfixes (us.forums.blizzard.com t/2403040,
// posts 5 and 6). Client 1.60.1.70888 and its push carry the Lightning
// Overload rows; the Enhancement changes are server side.

// withEnhancementRank sets one Enhancement talent's rank in the raid build.
func withEnhancementRank(index int, rank byte) string {
	trees := []byte(enhancementRaidTalents)
	trees[17+index] = '0' + rank // tree 1 starts after "0533001000000000-"
	return string(trees)
}

const (
	mentalDexterityIndex = 4
	maelstromWeaponIndex = 16
)

// "Mental Dexterity: Intellect to Attack Power conversion ratio reduced to
// 33/66/100% (was 50/100/150%)."
func TestMentalDexterityTenthNovemberRatios(t *testing.T) {
	for rank, want := range map[byte]float64{1: 0.33, 2: 0.66, 3: 1.0} {
		_, none := enhancementSimWith(t, withEnhancementRank(mentalDexterityIndex, 0))
		_, built := enhancementSimWith(t, withEnhancementRank(mentalDexterityIndex, rank))
		intellect := built.GetStat(stats.Intellect)
		got := (built.GetStat(stats.AttackPower) - none.GetStat(stats.AttackPower)) / intellect
		if math.Abs(got-want) > 0.005 {
			t.Errorf("Mental Dexterity %d/3 gives %.3f attack power per Intellect, want %.2f", rank, got, want)
		}
	}
}

// "Maelstrom Weapon now grants 20% reduced cast time and Mana cost per
// application at all ranks": five stacks make Lightning Bolt instant at 1/5.
func TestMaelstromWeaponTwentyPercentAStackAtEveryRank(t *testing.T) {
	for _, rank := range []byte{1, 5} {
		sim, built := enhancementSimWith(t, withEnhancementRank(maelstromWeaponIndex, rank))
		sim.Reset()
		sim.PrePull()
		aura := built.MaelstromWeaponAura
		aura.Activate(sim)
		aura.SetStacks(sim, 5)
		bolt := built.LightningBolt[len(built.LightningBolt)-1]
		if got := bolt.CastTimeMultiplier; math.Abs(got) > 1e-9 {
			t.Errorf("Maelstrom Weapon %d/5 at five stacks leaves Lightning Bolt's cast time at x%.2f, want x0", rank, got)
		}
	}
}

// "...activates 6/7/8/9/10 times per minute (was 20 times per minute
// regardless of rank)", on the main hand's speed: "5 points ... a 3.6
// speed weapon ... 60% chance to activate on swing".
func TestMaelstromWeaponProcsPerMinuteByRank(t *testing.T) {
	melee := &core.Spell{ProcMask: core.ProcMaskMeleeMHAuto}
	for rank, ppm := range map[byte]float64{1: 6, 3: 8, 5: 10} {
		sim, built := enhancementSimWith(t, withEnhancementRank(maelstromWeaponIndex, rank))
		sim.Reset()
		want := ppm * built.AutoAttacks.MH().SwingSpeed / 60
		if got := built.MaelstromWeaponChanceForTest(melee); math.Abs(got-want) > 1e-9 {
			t.Errorf("Maelstrom Weapon %d/5: %.3f a main-hand hit, want %.3f (%v a minute)", rank, got, want, ppm)
		}
	}
}

// Totem of the Storm (client spell 1291078): "Your Lightning Bolt ability can
// now also trigger the Maelstrom Weapon talent, but with a 50% reduced
// chance" - half a main-hand hit's; no other spell triggers it.
func TestMaelstromWeaponChanceForLightningBoltAndShocks(t *testing.T) {
	sim, built := enhancementSim(t)
	sim.Reset()
	bolt := &core.Spell{ProcMask: core.ProcMaskSpellDamage, SpellCode: shaman.SpellCode_ShamanLightningBolt}
	shock := &core.Spell{ProcMask: core.ProcMaskSpellDamage, SpellCode: shaman.SpellCode_ShamanEarthShock}
	melee := built.MaelstromWeaponChanceForTest(&core.Spell{ProcMask: core.ProcMaskMeleeMHAuto})
	if got := built.MaelstromWeaponChanceForTest(bolt); got != 0 {
		t.Errorf("Lightning Bolt without Totem of the Storm = %v, want 0", got)
	}
	built.LightningBoltMaelstromFactor = 0.5
	if got, want := built.MaelstromWeaponChanceForTest(bolt), melee/2; math.Abs(got-want) > 1e-12 || want <= 0 {
		t.Errorf("Lightning Bolt with Totem of the Storm = %v, want half a main-hand hit's (%v)", got, want)
	}
	if got := built.MaelstromWeaponChanceForTest(shock); got != 0 {
		t.Errorf("Earth Shock = %v, want 0", got)
	}
}

// "Lightning Overload now deals the correct amount of damage for Lightning
// Bolt" (9 November): client 1.60.1.70888 makes every overload row half its
// bolt rank's (408439-408477 and 408479-408484, SpellEffect). Read here, not
// from the vendored client file: 70938's spellconst carries the rows but
// rounds EffectBasePointsF to a whole number (36.5 reads 36).
func TestLightningOverloadRowsAreThe70888Client(t *testing.T) {
	bolt := []clientdamage.Effect{{},
		{Amount: 8, Variance: 0.142857, PerLevel: 0.05, SpellLevel: 1, MaxLevel: 6},
		{Amount: 15, Variance: 0.142857, PerLevel: 0.05, SpellLevel: 8, MaxLevel: 13},
		{Amount: 26, Variance: 0.163265, PerLevel: 0.15, SpellLevel: 14, MaxLevel: 19},
		{Amount: 36.5, Variance: 0.134831, PerLevel: 0.25, SpellLevel: 20, MaxLevel: 25},
		{Amount: 54.5, Variance: 0.134328, PerLevel: 0.35, SpellLevel: 26, MaxLevel: 31},
		{Amount: 74, Variance: 0.120879, PerLevel: 0.5, SpellLevel: 32, MaxLevel: 37},
		{Amount: 97, Variance: 0.125, PerLevel: 0.5, SpellLevel: 38, MaxLevel: 43},
		{Amount: 120, Variance: 0.114094, PerLevel: 0.75, SpellLevel: 44, MaxLevel: 49},
		{Amount: 147.5, Variance: 0.119565, PerLevel: 0.8, SpellLevel: 50, MaxLevel: 55},
		{Amount: 176.5, Variance: 0.113122, PerLevel: 1.05, SpellLevel: 56, MaxLevel: 61},
	}
	chain := []clientdamage.Effect{{},
		{Amount: 107.5, Variance: 0.137255, PerLevel: 0.7, SpellLevel: 32, MaxLevel: 37},
		{Amount: 174, Variance: 0.122449, PerLevel: 1.1, SpellLevel: 40, MaxLevel: 45},
		{Amount: 214, Variance: 0.115, PerLevel: 1.35, SpellLevel: 48, MaxLevel: 53},
		{Amount: 255, Variance: 0.114943, PerLevel: 1.65, SpellLevel: 56, MaxLevel: 61},
	}
	assertRows(t, "Lightning Bolt overload", shaman.LightningBoltOverloadDamage[:], bolt)
	assertRows(t, "Chain Lightning overload", shaman.ChainLightningOverloadDamage[:], chain)
	boltCoef := []float64{0, 0.2145, 0.2855, 0.357, 0.357, 0.357, 0.357, 0.357, 0.357, 0.357, 0.357}
	for i, want := range boltCoef {
		if got := shaman.LightningBoltOverloadSpellCoef[i]; math.Abs(got-want) > 1e-4 {
			t.Errorf("Lightning Bolt overload rank %d coefficient = %v, want %v", i, got, want)
		}
	}
	for i := 1; i < len(shaman.ChainLightningOverloadSpellCoef); i++ {
		if got := shaman.ChainLightningOverloadSpellCoef[i]; math.Abs(got-0.357) > 1e-4 {
			t.Errorf("Chain Lightning overload rank %d coefficient = %v, want 0.357", i, got)
		}
	}
}

func assertRows(t *testing.T, name string, got, want []clientdamage.Effect) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: %d ranks, want %d", name, len(got)-1, len(want)-1)
	}
	for i := range want {
		g, w := got[i], want[i]
		if math.Abs(g.Amount-w.Amount) > 1e-6 || math.Abs(g.Variance-w.Variance) > 1e-5 || math.Abs(g.PerLevel-w.PerLevel) > 1e-6 ||
			g.SpellLevel != w.SpellLevel || g.MaxLevel != w.MaxLevel {
			t.Errorf("%s rank %d = %+v, want %+v", name, i, g, w)
		}
	}
}

// "Weapon imbues and Weapon totems can now be used at the same time ... the
// Shaman can benefit from Windfury Weapon and Windfury Totem at the same
// time" (10 November). A shaman wearing Windfury Weapon on the main hand
// takes the raid's Windfury and Flametongue Totems.
func TestWeaponTotemsReachAShamanWearingTheirImbue(t *testing.T) {
	for _, imbue := range []proto.WeaponImbue{proto.WeaponImbue_WindfuryWeapon, proto.WeaponImbue_FlametongueWeapon} {
		player := core.WithSpec(
			&proto.Player{
				Class:         proto.Class_ClassShaman,
				Race:          proto.Race_RaceOrc,
				Level:         60,
				Equipment:     core.GetGearSet("../../ui/enhancement_shaman/gear_sets", "phase_5").GearSet,
				TalentsString: enhancementRaidTalents,
				Consumes:      &proto.Consumes{MainHandImbue: imbue},
			},
			&proto.Player_EnhancementShaman{EnhancementShaman: &proto.EnhancementShaman{Options: &proto.EnhancementShaman_Options{}}},
		)
		raid := core.SinglePlayerRaidProto(player, nil, &proto.RaidBuffs{WindfuryTotem: true, FlametongueTotem: true}, nil)
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid:       raid,
			Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{{Level: 63}}},
			SimOptions: &proto.SimOptions{RandomSeed: 1, Iterations: 1},
		}, simsignals.CreateSignals())
		built := sim.Raid.Parties[0].Players[0].(shaman.ShamanAgent).GetShaman()
		for _, label := range []string{core.WindfuryAuraLabel, core.FlametongueTotemAuraLabel} {
			if built.GetAura(label) == nil {
				t.Errorf("a shaman with %v on the main hand has no %q from the raid's totem", imbue, label)
			}
		}
	}
}
