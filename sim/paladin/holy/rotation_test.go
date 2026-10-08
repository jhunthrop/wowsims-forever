package holy

import (
	"os"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/healsim"
)

const (
	// defaultRotationFile is the rotation apl-sync writes from the site's
	// data/curated/apl/paladin-holy.json.
	defaultRotationFile = "../../../ui/holy_paladin/apls/forever_holy.apl.json"

	rotationSeconds    = 180
	rotationIterations = 60
	// maxOverheal is the overheal share the default rotation must stay under
	// against the test profile.
	maxOverheal = 0.8
	// manaSpentTolerance is how far the mana the sim took may sit from the
	// cost of the casts it counted.
	manaSpentTolerance = 0.02

	greaterBlessingOfLight int32 = 25890
)

// A healer with some gear: enough that the rotation's heals are the ones a
// level 60 raid healer would cast.
var raidGear = stats.Stats{
	stats.Intellect:    250,
	stats.Spirit:       100,
	stats.HealingPower: 500,
	stats.MP5:          20,
	stats.Crit:         5 * core.CritRatingPerCritChance,
}

// manaToSpare is a pool large enough that the rotation's paced lines open.
const manaToSpare = 15000

var holyBuild = map[string]int{
	"divine_intellect": 5, "healing_light": 3, "spiritual_focus": 2, "reverence": 3,
	"infusion_of_light": 2, "illumination": 5, "divine_favor": 1, "holy_shock": 1,
	"holy_power": 5, "lights_vigil": 1,
}

func defaultRotation(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(defaultRotationFile)
	if err != nil {
		t.Fatalf("the default rotation is missing (make apl-sync in the site worktree writes it): %v", err)
	}
	return string(raw)
}

func rotationFight(t *testing.T, extraMana float64) fight {
	bonus := raidGear
	bonus[stats.Mana] += extraMana
	return fight{
		rotation: defaultRotation(t),
		talents:  holyBuild,
		bonus:    bonus,
		duration: rotationSeconds,
		model:    healsim.TestProfile(),
		realMana: true,
	}
}

// manaSpent is the mana the healer paid for a spell, summed over the run.
func manaSpent(result *proto.RaidSimResult, spellID int32) float64 {
	var spent float64
	for _, resource := range result.RaidMetrics.Parties[0].Players[healsim.HealerIndex].Resources {
		if resource.Id.GetSpellId() == spellID && resource.Type == proto.ResourceType_ResourceTypeMana && resource.Gain < 0 {
			spent -= resource.Gain
		}
	}
	return spent
}

func TestDefaultRotationHealsTheFakeRaid(t *testing.T) {
	result := rotationFight(t, 0).run(t, rotationIterations)
	player := result.RaidMetrics.Parties[0].Players[healsim.HealerIndex]

	if player.EffectiveHps.Avg <= 0 {
		t.Fatalf("effective HPS %.1f, want healing that landed", player.EffectiveHps.Avg)
	}
	overheal := 1 - player.EffectiveHps.Avg/player.Hps.Avg
	if overheal < 0 || overheal >= maxOverheal {
		t.Errorf("overheal share %.2f, want 0 to %.1f", overheal, maxOverheal)
	}
	for name, id := range map[string]int32{
		"Flash of Light": flashOfLightTopRank, "Greater Blessing of Light": greaterBlessingOfLight,
	} {
		if totalsOf(result, id).casts == 0 {
			t.Errorf("the default rotation never cast %s", name)
		}
	}
}

// heavyProfile is the site's heal profile (data/curated/heal-profile.json,
// "onyxia-sized"): a tank hit for 1150 every other second and raid pulses
// of 450 on three members every four seconds.
func heavyProfile() *proto.RaidDamageModel {
	return &proto.RaidDamageModel{
		Profile: "onyxia-sized", TankHealth: 9500, MemberHealth: 5000,
		TankHitDamage: 1150, TankSwingSeconds: 2, DamageSpread: 0.25,
		PulseDamage: 450, PulseIntervalSeconds: 4, PulseMembers: 3,
	}
}

func TestDefaultRotationUsesDivineFavorAndHolyLightWhenTheTankIsInDanger(t *testing.T) {
	f := rotationFight(t, 0)
	f.model = heavyProfile()
	result := f.run(t, rotationIterations)

	for name, id := range map[string]int32{"Holy Light": holyLightTopRank, "Divine Favor": divineFavorSpell} {
		if totalsOf(result, id).casts == 0 {
			t.Errorf("the default rotation never cast %s against a tank taking %v every swing", name, f.model.TankHitDamage)
		}
	}
}

func TestDefaultRotationSpendsTheManaItsCastsCost(t *testing.T) {
	result := rotationFight(t, 0).run(t, rotationIterations)
	costs := map[int32]float64{
		flashOfLightTopRank: 140, holyLightTopRank: 660, greaterBlessingOfLight: 260,
	}
	var want, got float64
	for id, cost := range costs {
		want += float64(totalsOf(result, id).casts) * cost
		got += manaSpent(result, id)
	}
	if want == 0 || !within(got, want, manaSpentTolerance) {
		t.Errorf("mana spent %.0f, want about %.0f (casts times cost)", got, want)
	}
}

func TestDefaultRotationSpendsSpareManaOnHolyShockAndHolyLight(t *testing.T) {
	short := rotationFight(t, 0).run(t, rotationIterations)
	spare := rotationFight(t, manaToSpare).run(t, rotationIterations)

	if casts := totalsOf(short, holyShockTopCast).casts; casts != 0 {
		t.Errorf("a healer short of mana cast Holy Shock %d times; the paced lines should not open", casts)
	}
	if totalsOf(spare, holyShockTopCast).casts == 0 {
		t.Errorf("a healer with spare mana never cast Holy Shock")
	}
	shortHPS := short.RaidMetrics.Parties[0].Players[healsim.HealerIndex].EffectiveHps.Avg
	spareHPS := spare.RaidMetrics.Parties[0].Players[healsim.HealerIndex].EffectiveHps.Avg
	if spareHPS <= shortHPS {
		t.Errorf("spare mana did not raise effective HPS: %.0f against %.0f", spareHPS, shortHPS)
	}
}
