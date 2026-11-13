package shaman_test

import (
	"math"
	"testing"

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
