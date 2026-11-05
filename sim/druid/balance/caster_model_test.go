package balance

import (
	"github.com/wowsims/classic/sim/druid"
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	googleProto "google.golang.org/protobuf/proto"
)

// Balance proto field numbers (proto/druid.proto).
const (
	improvedWrathField    = 1
	moonglowField         = 3
	improvedMoonfireField = 4
	insectSwarmField      = 9
	naturesGraceField     = 13
	moonfuryField         = 15
	moonkinFormField      = 16
)

func near(got, want float64) bool { return math.Abs(got-want) < 1e-9 }

func currentCost(spell *core.Spell) float64 { return spell.Cost.GetCurrentCost() }

// TestMoonglowDiscountsEveryBalanceDamageSpellByTheClientsPercent: 8/17/25%
// off "your damaging spells" (spell 16845, class mask Wrath, Moonfire,
// Starfire, Insect Swarm, ...). The engine took 3 points a rank off three
// spells, and took Starfire's off twice.
func TestMoonglowDiscountsEveryBalanceDamageSpellByTheClientsPercent(t *testing.T) {
	bare, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{insectSwarmField: 1}))

	for rank, discount := range map[int]float64{1: 0.08, 2: 0.17, 3: 0.25} {
		talented, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{insectSwarmField: 1, moonglowField: rank}))
		spells := map[string][2]*core.Spell{
			"Wrath":        {bare.Wrath[8].Spell, talented.Wrath[8].Spell},
			"Starfire":     {bare.Starfire[core.MaxTrainerRank(druid.StarfireRanks)].Spell, talented.Starfire[core.MaxTrainerRank(druid.StarfireRanks)].Spell},
			"Moonfire":     {bare.Moonfire[len(bare.Moonfire)-1].Spell, talented.Moonfire[len(talented.Moonfire)-1].Spell},
			"Insect Swarm": {bare.InsectSwarm[5].Spell, talented.InsectSwarm[5].Spell},
		}
		for name, pair := range spells {
			want := currentCost(pair[0]) * (1 - discount)
			if got := currentCost(pair[1]); !near(got, want) {
				t.Errorf("Moonglow %d/3 %s costs %v, want %v (%.0f%% off %v)", rank, name, got, want, discount*100, currentCost(pair[0]))
			}
		}
	}
}

// TestImprovedWrathCutsManaAsWellAsCastTime: "Reduces the cast time of
// your Wrath spell by 0.1 sec and its Mana cost by 10%" per rank.
func TestImprovedWrathCutsManaAsWellAsCastTime(t *testing.T) {
	bare, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, nil))
	talented, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{improvedWrathField: 5}))

	if got, want := currentCost(talented.Wrath[8].Spell), currentCost(bare.Wrath[8].Spell)*0.5; !near(got, want) {
		t.Errorf("Wrath costs %v with Improved Wrath 5/5, want %v (50%% off)", got, want)
	}
	if got, want := talented.Wrath[8].DefaultCast.CastTime, bare.Wrath[8].DefaultCast.CastTime-500*time.Millisecond; got != want {
		t.Errorf("Wrath cast time %v with Improved Wrath 5/5, want %v", got, want)
	}
}

// TestImprovedMoonfireIsFivePercentPerRank: "Increases the damage and
// critical strike chance of your Moonfire spell by 5%" per rank (the
// client's trait curve gives 5 and 10 on all three effects).
func TestImprovedMoonfireIsFivePercentPerRank(t *testing.T) {
	bare, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, nil))
	talented, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{improvedMoonfireField: 2}))
	top := len(bare.Moonfire) - 1

	if got, want := talented.Moonfire[top].BaseDamageMultiplierAdditive-bare.Moonfire[top].BaseDamageMultiplierAdditive, 0.10; !near(got, want) {
		t.Errorf("Moonfire damage bonus +%v with Improved Moonfire 2/2, want +%v", got, want)
	}
	if got, want := talented.Moonfire[top].BonusCritRating-bare.Moonfire[top].BonusCritRating, 10*float64(core.CritRatingPerCritChance); !near(got, want) {
		t.Errorf("Moonfire crit rating bonus +%v with Improved Moonfire 2/2, want +%v (10%%)", got, want)
	}
}

// TestMoonfuryBoostsEveryArcaneAndNatureSpell: "Increases the damage done
// by your Arcane and Nature spells by 2%" per rank (spell 16896 is a
// school-damage aura, multiplicative), so Insect Swarm is covered as well
// as Wrath, Starfire and Moonfire.
func TestMoonfuryBoostsEveryArcaneAndNatureSpell(t *testing.T) {
	built, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{moonfuryField: 5, insectSwarmField: 1}))

	for name, school := range map[string]stats.SchoolIndex{"Arcane": stats.SchoolIndexArcane, "Nature": stats.SchoolIndexNature} {
		if got := built.PseudoStats.SchoolDamageDealtMultiplier[school]; !near(got, 1.10) {
			t.Errorf("%s damage multiplier %v with Moonfury 5/5, want 1.10", name, got)
		}
	}
	for name, school := range map[string]stats.SchoolIndex{"Fire": stats.SchoolIndexFire, "Frost": stats.SchoolIndexFrost, "Physical": stats.SchoolIndexPhysical} {
		if got := built.PseudoStats.SchoolDamageDealtMultiplier[school]; !near(got, 1) {
			t.Errorf("%s damage multiplier %v with Moonfury, want the untouched 1", name, got)
		}
	}
}

// TestNaturesGraceIsTenPercentHasteAndGcdForThreeSeconds: "All
// non-periodic spell criticals grace you with a blessing of nature,
// increasing your spellcasting speed and reducing your global cooldown by
// 10% for 3 sec" (16880 -> 16886: casting speed +10, global cooldown
// -10%, 3 s). The engine gave a flat 0.5 s off cast times for 15 s, lost
// on the next cast.
func TestNaturesGraceIsTenPercentHasteAndGcdForThreeSeconds(t *testing.T) {
	built, sim, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{naturesGraceField: 1}))
	proc := built.NaturesGraceProcAura
	if proc == nil {
		t.Fatal("no Nature's Grace proc aura with the talent learned")
	}
	if proc.Duration != 3*time.Second {
		t.Errorf("Nature's Grace lasts %v, want 3s", proc.Duration)
	}

	castSpeedBefore := built.CastSpeed
	wrathCastBefore := built.Wrath[8].DefaultCast.CastTime
	wrathGcdBefore := built.Wrath[8].DefaultCast.GCD
	moonfireGcdBefore := built.Moonfire[len(built.Moonfire)-1].DefaultCast.GCD

	proc.Activate(sim)

	if got, want := built.CastSpeed, castSpeedBefore/1.1; !near(got, want) {
		t.Errorf("cast speed factor %v during Nature's Grace, want %v (10%% faster)", got, want)
	}
	if got := built.Wrath[8].DefaultCast.CastTime; got != wrathCastBefore {
		t.Errorf("Wrath base cast time %v during Nature's Grace, want the unchanged %v (the haste is a speed multiplier, not a flat cut)", got, wrathCastBefore)
	}
	for name, pair := range map[string][2]time.Duration{
		"Wrath":    {wrathGcdBefore, built.Wrath[8].DefaultCast.GCD},
		"Moonfire": {moonfireGcdBefore, built.Moonfire[len(built.Moonfire)-1].DefaultCast.GCD},
	} {
		if got, want := pair[1], pair[0]*9/10; got != want {
			t.Errorf("%s global cooldown %v during Nature's Grace, want %v (10%% shorter)", name, got, want)
		}
	}

	proc.Deactivate(sim)

	if !near(built.CastSpeed, castSpeedBefore) || built.Wrath[8].DefaultCast.GCD != wrathGcdBefore {
		t.Errorf("Nature's Grace left cast speed %v and Wrath GCD %v after fading, want %v and %v", built.CastSpeed, built.Wrath[8].DefaultCast.GCD, castSpeedBefore, wrathGcdBefore)
	}
}

// TestMoonkinFormRaisesTheDruidsOwnSpellCrit: the form's text gives "all
// party members within 45 yards ... critical strike chance increased by
// 3%" (aura 24907, which includes the druid). The engine applied the form
// and gave it no effect at all, so a build spending its capstone point on
// it was credited zero.
func TestMoonkinFormRaisesTheDruidsOwnSpellCrit(t *testing.T) {
	alone := googleProto.Clone(core.FullBuffs.Raid).(*proto.RaidBuffs)
	alone.MoonkinAura = false     // the only Moonkin is this druid (TestMoonkinCritIsGrantedOnce)
	alone.LeaderOfThePack = false // exclusive with the form (TestMoonkinFormAddsNoCritBesideLeaderOfThePack)
	built, sim, target := newBalanceDruidSimInRaid(t, druidTalentsString(t, map[int]int{moonkinFormField: 1}), &proto.EquipmentSpec{}, alone)
	wrath := built.Wrath[8]

	if !built.MoonkinFormAura.IsActive() {
		t.Fatal("a Balance druid with the talent is not in Moonkin Form at the pull")
	}
	inForm := wrath.SpellCritChance(target)

	built.MoonkinFormAura.Deactivate(sim)
	outOfForm := wrath.SpellCritChance(target)

	if !near(inForm-outOfForm, 0.03) {
		t.Errorf("Moonkin Form adds %v spell crit, want 0.03", inForm-outOfForm)
	}
}

// moonkinCritFromTalent is the spell crit the Moonkin Form talent adds to
// the druid's Wrath under the given raid buffs.
func moonkinCritFromTalent(t *testing.T, raidBuffs *proto.RaidBuffs) float64 {
	t.Helper()
	talented, _, target := newBalanceDruidSimInRaid(t, druidTalentsString(t, map[int]int{moonkinFormField: 1}), &proto.EquipmentSpec{}, raidBuffs)
	plain, _, plainTarget := newBalanceDruidSimInRaid(t, druidTalentsString(t, nil), &proto.EquipmentSpec{}, raidBuffs)
	return talented.Wrath[8].SpellCritChance(target) - plain.Wrath[8].SpellCritChance(plainTarget)
}

// Moonkin Form's crit is one aura, 24907 (aura 290, 3), and the client's
// form spell 24858 has no crit effect of its own. The druid in the form is
// inside its own aura, which the raid buffs carry as MoonkinAura; an aura
// does not stack with itself, so another Moonkin in the raid adds nothing
// either. The druid gets the 3% once.
func TestMoonkinCritIsGrantedOnce(t *testing.T) {
	withAura := googleProto.Clone(core.FullBuffs.Raid).(*proto.RaidBuffs)
	withAura.MoonkinAura = true
	withoutAura := googleProto.Clone(core.FullBuffs.Raid).(*proto.RaidBuffs)
	withoutAura.MoonkinAura = false
	withoutAura.LeaderOfThePack = false

	if got := moonkinCritFromTalent(t, withoutAura); !near(got, 0.03) {
		t.Errorf("with no other Moonkin in the raid, the form adds %v spell crit, want 0.03", got)
	}
	if got := moonkinCritFromTalent(t, withAura); !near(got, 0) {
		t.Errorf("with the raid's Moonkin Aura already up, the form adds %v spell crit, want 0 (one aura)", got)
	}
}

// TestBalanceDruidsMoonkinAuraIsCountedOnce: the Balance spec started every
// druid in Moonkin Form whether or not it had the talent (a leveling Balance
// druid below 40 has not learned it), and a druid in the form raised the
// raid's Moonkin Aura on top of the form's own aura. 24907 (aura 290, 3%)
// is one aura and does not stack with itself, so:
//   - without the talent, the druid is in caster form and has no Moonkin
//     Aura (it had the raid's 3%);
//   - with it, the form's aura is the druid's only 3% (it had 6% whenever
//     the raid buffs did not already carry the aura: every bare band).
func TestBalanceDruidsMoonkinAuraIsCountedOnce(t *testing.T) {
	for _, tc := range []struct {
		name     string
		points   map[int]int
		inForm   bool
		formCrit float64
	}{
		{"without the talent", map[int]int{insectSwarmField: 1}, false, 0},
		{"with the talent", map[int]int{insectSwarmField: 1, moonkinFormField: 1}, true, 0.03},
	} {
		t.Run(tc.name, func(t *testing.T) {
			built, sim, target := newBalanceDruidSimInRaid(t, druidTalentsString(t, tc.points), &proto.EquipmentSpec{}, &proto.RaidBuffs{})
			if got := built.InForm(druid.Moonkin); got != tc.inForm {
				t.Errorf("in Moonkin Form %v, want %v", got, tc.inForm)
			}
			raidBuffs := &proto.RaidBuffs{}
			built.AddRaidBuffs(raidBuffs)
			if raidBuffs.MoonkinAura {
				t.Error("the druid raises the raid's Moonkin Aura on top of its own form's aura")
			}
			wrath := built.Wrath[8]
			withForm := wrath.SpellCritChance(target)
			if built.MoonkinFormAura != nil {
				built.MoonkinFormAura.Deactivate(sim)
			}
			if got := withForm - wrath.SpellCritChance(target); !near(got, tc.formCrit) {
				t.Errorf("Moonkin Form gives %v spell crit, want %v", got, tc.formCrit)
			}
		})
	}
}

// wrathCritUnder is a talentless druid's Wrath crit chance under the given
// raid buffs.
func wrathCritUnder(t *testing.T, raidBuffs *proto.RaidBuffs) float64 {
	t.Helper()
	built, _, target := newBalanceDruidSimInRaid(t, druidTalentsString(t, nil), &proto.EquipmentSpec{}, raidBuffs)
	return built.Wrath[8].SpellCritChance(target)
}

// Blizzard's Forever notes of 3 November 2026: "Moonkin Form: The critical
// strike bonus from this talent is again an increase to all types of
// critical strikes and is again exclusive with Leader of the Pack" and
// "Leader of the Pack: ... is again exclusive with Moonkin Aura." Both are
// aura 290 at 3% (24907, 24932), so a raid with both gives 3%, not 6%.
func TestLeaderOfThePackAndMoonkinAuraAreExclusive(t *testing.T) {
	both := googleProto.Clone(core.FullBuffs.Raid).(*proto.RaidBuffs)
	both.MoonkinAura, both.LeaderOfThePack = true, true
	moonkinOnly := googleProto.Clone(both).(*proto.RaidBuffs)
	moonkinOnly.LeaderOfThePack = false
	packOnly := googleProto.Clone(both).(*proto.RaidBuffs)
	packOnly.MoonkinAura = false
	neither := googleProto.Clone(packOnly).(*proto.RaidBuffs)
	neither.LeaderOfThePack = false

	base := wrathCritUnder(t, neither)
	for name, buffs := range map[string]*proto.RaidBuffs{"both": both, "Moonkin Aura": moonkinOnly, "Leader of the Pack": packOnly} {
		if got := wrathCritUnder(t, buffs) - base; !near(got, 0.03) {
			t.Errorf("%s gives %v crit, want 0.03", name, got)
		}
	}
}

// The druid's own Moonkin Form aura is exclusive with a raid Leader of the
// Pack as well: in a raid with a feral druid the form adds no crit.
func TestMoonkinFormAddsNoCritBesideLeaderOfThePack(t *testing.T) {
	pack := googleProto.Clone(core.FullBuffs.Raid).(*proto.RaidBuffs)
	pack.MoonkinAura, pack.LeaderOfThePack = false, true
	if got := moonkinCritFromTalent(t, pack); !near(got, 0) {
		t.Errorf("beside the raid's Leader of the Pack the form adds %v crit, want 0 (exclusive)", got)
	}
}
