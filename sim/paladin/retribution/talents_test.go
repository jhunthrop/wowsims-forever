package retribution

import (
	"math"
	"slices"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// newTalentTestUnit builds a level-60 Retribution Paladin carrying only
// the given talent string and returns its Unit for direct Spell/stat
// inspection -- no rotation runs, so this is for structural assertions
// (a spell's registered DamageMultiplier, Cost, CastTime, CD, or a stat
// dependency's effect on GetStat), not for proc-rate or metrics
// assertions, which need an actual sim (see TestSanctifiedJudgement... /
// TestTwistOfLightEcho... below).
func newTalentTestUnit(t *testing.T, talents string) (*RetributionPaladin, *core.Unit) {
	t.Helper()
	return newTalentTestUnitWith(t, talents, &proto.EquipmentSpec{})
}

// newTalentTestUnitWith is newTalentTestUnit wearing the given equipment.
func newTalentTestUnitWith(t *testing.T, talents string, equipment *proto.EquipmentSpec) (*RetributionPaladin, *core.Unit) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassPaladin,
			Race:          proto.Race_RaceHuman,
			Level:         60,
			Equipment:     equipment,
			Buffs:         core.FullBuffs.Player,
			TalentsString: talents,
		},
		&proto.Player_RetributionPaladin{
			RetributionPaladin: &proto.RetributionPaladin{
				Options: &proto.PaladinOptions{},
			},
		},
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	encounter := &proto.Encounter{
		Duration: 10,
		Targets:  []*proto.Target{core.NewDefaultTarget()},
	}

	env, _, _ := core.NewEnvironment(raid, encounter, true)
	agent := env.Raid.Parties[0].Players[0]
	ret, ok := agent.(*RetributionPaladin)
	if !ok {
		t.Fatalf("player agent is %T, want *RetributionPaladin", agent)
	}
	return ret, &ret.Paladin.Character.Unit
}

// Level 60 top-rank spell ids this file checks against (see sor.go,
// soc.go, consecration.go, holy_wrath.go, exorcism.go, hammer_of_wrath.go
// rank tables).
const (
	judgementOfRighteousnessRank8SpellID = 20286
	sealOfRighteousnessProcRank8SpellID  = 25713
	consecrationRank5SpellID             = 20924
	holyWrathRank2SpellID                = 10318
	exorcismRank6SpellID                 = 10314
	hammerOfWrathRank3SpellID            = 24239
)

func TestImprovedSealsIncreasesSealAndJudgementDamage(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"improved_seals": 5}))

	baseJudge := baseUnit.GetSpell(core.ActionID{SpellID: judgementOfRighteousnessRank8SpellID})
	talentedJudge := talentedUnit.GetSpell(core.ActionID{SpellID: judgementOfRighteousnessRank8SpellID})
	if baseJudge == nil || talentedJudge == nil {
		t.Fatalf("Judgement of Righteousness rank 8 not registered: base=%v talented=%v", baseJudge, talentedJudge)
	}
	wantMultiplier := baseJudge.DamageMultiplier * 1.15
	if talentedJudge.DamageMultiplier < wantMultiplier-1e-9 || talentedJudge.DamageMultiplier > wantMultiplier+1e-9 {
		t.Errorf("Judgement of Righteousness DamageMultiplier = %v, want %v (base %v * 1.15)", talentedJudge.DamageMultiplier, wantMultiplier, baseJudge.DamageMultiplier)
	}

	baseProc := baseUnit.GetSpell(core.ActionID{SpellID: sealOfRighteousnessProcRank8SpellID})
	talentedProc := talentedUnit.GetSpell(core.ActionID{SpellID: sealOfRighteousnessProcRank8SpellID})
	if baseProc == nil || talentedProc == nil {
		t.Fatalf("Seal of Righteousness proc rank 8 not registered: base=%v talented=%v", baseProc, talentedProc)
	}
	wantProcMultiplier := baseProc.DamageMultiplier * 1.15
	if talentedProc.DamageMultiplier < wantProcMultiplier-1e-9 || talentedProc.DamageMultiplier > wantProcMultiplier+1e-9 {
		t.Errorf("Seal of Righteousness proc DamageMultiplier = %v, want %v", talentedProc.DamageMultiplier, wantProcMultiplier)
	}
}

func TestReverenceIncreasesSpiritRegenWhileCasting(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"reverence": 2}))

	if baseUnit.PseudoStats.SpiritRegenRateCasting != 0 {
		t.Errorf("base SpiritRegenRateCasting = %v, want 0", baseUnit.PseudoStats.SpiritRegenRateCasting)
	}
	const want = 0.30
	got := talentedUnit.PseudoStats.SpiritRegenRateCasting
	if got < want-1e-9 || got > want+1e-9 {
		t.Errorf("rank 2 Reverence SpiritRegenRateCasting = %v, want %v", got, want)
	}
}

func TestPurifyingPowerReducesExorcismAndHolyWrathCooldown(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"purifying_power": 2}))

	for _, spellID := range []int32{exorcismRank6SpellID, holyWrathRank2SpellID} {
		base := baseUnit.GetSpell(core.ActionID{SpellID: spellID})
		talented := talentedUnit.GetSpell(core.ActionID{SpellID: spellID})
		if base == nil || talented == nil {
			t.Fatalf("spell %d not registered: base=%v talented=%v", spellID, base, talented)
		}
		wantCD := base.CD.Duration * 67 / 100 // rank 2: -33%
		if talented.CD.Duration != wantCD {
			t.Errorf("spell %d CD = %v, want %v (base %v - 33%%)", spellID, talented.CD.Duration, wantCD, base.CD.Duration)
		}
	}
}

func TestDivinePrecisionAddsHolySpellHit(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"divine_precision": 2}))

	base := baseUnit.GetSpell(core.ActionID{SpellID: exorcismRank6SpellID})
	talented := talentedUnit.GetSpell(core.ActionID{SpellID: exorcismRank6SpellID})
	if base == nil || talented == nil {
		t.Fatalf("Exorcism rank 6 not registered: base=%v talented=%v", base, talented)
	}
	wantBonus := 8.0 * core.HitRatingPerHitChance // 1.60.1.70707: 4% a rank, 2 ranks
	gotBonus := talented.BonusHitRating - base.BonusHitRating
	if gotBonus < wantBonus-1e-6 || gotBonus > wantBonus+1e-6 {
		t.Errorf("rank 2 Divine Precision BonusHitRating delta on Exorcism = %v, want %v (8%% hit)", gotBonus, wantBonus)
	}
}

func TestConsecratedGroundIncreasesConsecrationDamage(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"consecrated_ground": 2}))

	base := baseUnit.GetSpell(core.ActionID{SpellID: consecrationRank5SpellID})
	talented := talentedUnit.GetSpell(core.ActionID{SpellID: consecrationRank5SpellID})
	if base == nil || talented == nil {
		t.Fatalf("Consecration rank 5 not registered: base=%v talented=%v", base, talented)
	}
	wantMultiplier := base.DamageMultiplier * 1.10
	if talented.DamageMultiplier < wantMultiplier-1e-9 || talented.DamageMultiplier > wantMultiplier+1e-9 {
		t.Errorf("rank 2 Consecrated Ground Consecration DamageMultiplier = %v, want %v", talented.DamageMultiplier, wantMultiplier)
	}
}

func TestHolyConduitReducesRotationManaCosts(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"holy_conduit": 2}))

	for _, spellID := range []int32{consecrationRank5SpellID, holyWrathRank2SpellID, exorcismRank6SpellID, hammerOfWrathRank3SpellID} {
		base := baseUnit.GetSpell(core.ActionID{SpellID: spellID})
		talented := talentedUnit.GetSpell(core.ActionID{SpellID: spellID})
		if base == nil || talented == nil {
			t.Fatalf("spell %d not registered: base=%v talented=%v", spellID, base, talented)
		}
		baseCost := base.Cost.GetCurrentCost()
		talentedCost := talented.Cost.GetCurrentCost()
		wantCost := baseCost * 0.6 // rank 2: -40%
		if talentedCost < wantCost-1e-6 || talentedCost > wantCost+1e-6 {
			t.Errorf("spell %d mana cost = %v, want %v (base %v - 40%%)", spellID, talentedCost, wantCost, baseCost)
		}
	}
}

// Champion of the Light is 12% of Intellect a rank, 60% at 5/5 (client
// 1.60.1.70707; 20/40/60% over 3 ranks before).
func TestChampionOfTheLightAddsSpellPowerFromIntellect(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"champion_of_the_light": 5}))

	baseIntellect := baseUnit.GetStat(stats.Intellect)
	if baseIntellect <= 0 {
		t.Fatalf("base Intellect = %v, want > 0 (test needs a nonzero base to observe the dependency)", baseIntellect)
	}

	wantSpellPower := baseUnit.GetStat(stats.SpellPower) + talentedUnit.GetStat(stats.Intellect)*0.60 // "up to 60%" at 5/5
	gotSpellPower := talentedUnit.GetStat(stats.SpellPower)
	if gotSpellPower < wantSpellPower-1e-6 || gotSpellPower > wantSpellPower+1e-6 {
		t.Errorf("rank 5 Champion of the Light SpellPower = %v, want %v (60%% of %v Intellect)", gotSpellPower, wantSpellPower, talentedUnit.GetStat(stats.Intellect))
	}
}

// Champion of the Light also takes 0.2 sec a rank off Hammer of Wrath's
// cast (client 1.60.1.70707, the removed Instrument of Law's half): 0.6 sec
// at rank 3, an instant at rank 5.
func TestChampionOfTheLightShortensHammerOfWrath(t *testing.T) {
	castTime := func(rank int) time.Duration {
		_, unit := newTalentTestUnit(t, talentString(t, map[string]int{"champion_of_the_light": rank}))
		spell := unit.GetSpell(core.ActionID{SpellID: hammerOfWrathRank3SpellID})
		if spell == nil {
			t.Fatalf("Hammer of Wrath rank 3 not registered")
		}
		return spell.DefaultCast.CastTime
	}
	for rank, want := range map[int]time.Duration{0: time.Second, 3: 400 * time.Millisecond, 5: 0} {
		if got := castTime(rank); got != want {
			t.Errorf("Hammer of Wrath cast time with Champion of the Light %d = %v, want %v", rank, got, want)
		}
	}
}

// Conviction (client 1.60.1.70707): 2% crit with all spells and attacks a
// rank, 6% at 3/3 on every table (aura 290 after the push; 1% melee a
// rank over 5 ranks before).
func TestConvictionAddsTwoPercentCritARank(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"conviction": 3}))
	if got, want := talentedUnit.GetStat(stats.Crit)-baseUnit.GetStat(stats.Crit), 6.0*core.CritRatingPerCritChance; got < want-1e-6 || got > want+1e-6 {
		t.Errorf("Conviction 3 adds %v crit rating, want %v", got, want)
	}
}

// Vindication (client 1.60.1.70707) no longer raises the paladin's attack
// power: the proc aura that did is gone.
func TestVindicationNoLongerRaisesAttackPower(t *testing.T) {
	_, unit := newTalentTestUnit(t, talentString(t, map[string]int{"vindication": 3}))
	if aura := unit.GetAuraByID(core.ActionID{SpellID: 26021}); aura != nil {
		t.Errorf("Vindication registers %q, its attack power buff", aura.Label)
	}
}

// mainHandOnly is an equipment spec with only the main hand filled.
func mainHandOnly(t *testing.T, handType proto.HandType) *proto.EquipmentSpec {
	t.Helper()
	var ids []int32
	for id, item := range core.ItemsByID {
		if item.Type == proto.ItemType_ItemTypeWeapon && item.HandType == handType && item.WeaponDamageMax > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		t.Fatalf("no weapon with hand type %v in the item database; this test needs --tags=with_db", handType)
	}
	slices.Sort(ids)
	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotMainHand] = &proto.ItemSpec{Id: ids[0]}
	return &proto.EquipmentSpec{Items: items}
}

// Two-Handed Weapon Specialization (client 1.60.1.70707): "Increases the
// physical damage you deal by 6%, and reduces all threat you generate by
// 20% while a two-handed melee weapon is equipped" at 3/3. It reaches
// physical damage only, so Holy Strike (a Holy weapon attack) no longer
// gains it, and a one-hander gets neither half.
func TestTwoHandedWeaponSpecializationIsPhysicalDamageAndLessThreat(t *testing.T) {
	talents := talentString(t, map[string]int{"two_handed_weapon_specialization": 3})
	for _, tc := range []struct {
		hand           proto.HandType
		damage, threat float64
	}{
		{proto.HandType_HandTypeTwoHand, 1.06, 0.80},
		{proto.HandType_HandTypeMainHand, 1, 1},
	} {
		_, base := newTalentTestUnitWith(t, "", mainHandOnly(t, tc.hand))
		_, talented := newTalentTestUnitWith(t, talents, mainHandOnly(t, tc.hand))
		physical := talented.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] / base.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical]
		if math.Abs(physical-tc.damage) > 1e-9 {
			t.Errorf("%v: physical damage x%v, want x%v", tc.hand, physical, tc.damage)
		}
		if threat := talented.PseudoStats.ThreatMultiplier / base.PseudoStats.ThreatMultiplier; math.Abs(threat-tc.threat) > 1e-9 {
			t.Errorf("%v: threat x%v, want x%v", tc.hand, threat, tc.threat)
		}
		holyStrike := func(unit *core.Unit) float64 {
			return unit.GetSpell(core.ActionID{SpellID: 10333}).DamageMultiplier
		}
		if got, want := holyStrike(talented), holyStrike(base); math.Abs(got-want) > 1e-9 {
			t.Errorf("%v: Holy Strike's multiplier is %v with the talent, %v without", tc.hand, got, want)
		}
	}
}

func TestSealOfCommandRequiresTalent(t *testing.T) {
	_, unit := newTalentTestUnit(t, "")

	if spell := unit.GetSpell(core.ActionID{SpellID: 20920}); spell != nil {
		t.Errorf("Seal of Command rank 5 (20920) registered without the seal_of_command talent")
	}
	if spell := unit.GetSpell(core.ActionID{SpellID: 20966}); spell != nil {
		t.Errorf("Judgement of Command rank 5 (20966) registered without the seal_of_command talent")
	}
}

// TestSanctifiedJudgementRefundsMana uses rank 3 (100% proc chance) for
// determinism: every Judgement cast must return mana, so a single cast
// is enough to prove the hook fires and AddMana is reached, without
// depending on sim.Proc's RNG draw.
func TestSanctifiedJudgementRefundsMana(t *testing.T) {
	rotation := core.APLRotationFromJsonString(`{
		"type": "TypeAPL",
		"prepullActions": [
			{"action": {"castSpell": {"spellId": {"spellId": 20293}}}, "doAtValue": {"const": {"val": "-1.5s"}}}
		],
		"priorityList": [
			{"action": {"castSpell": {"spellId": {"spellId": 20271}}}}
		]
	}`)

	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassPaladin,
		Race:               proto.Race_RaceHuman,
		Level:              60,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              core.FullBuffs.Player,
		TalentsString:      talentString(t, map[string]int{"sanctified_judgement": 3}),
		Rotation:           rotation,
		DistanceFromTarget: 5,
	}, PlayerOptionsSealofRighteousness)

	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 15,
			Targets:  []*proto.Target{core.NewDefaultTarget()},
		},
		SimOptions: &proto.SimOptions{
			Iterations: 1,
			RandomSeed: 1,
			IsTest:     true,
		},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}

	judgeMetrics := findActionMetrics(result, 20271)
	if judgeMetrics == nil {
		t.Fatalf("Judgement never appears in the action metrics -- it did not cast")
	}
	var judgeCasts int32
	for _, target := range judgeMetrics.Targets {
		judgeCasts += target.Casts
	}
	if judgeCasts == 0 {
		t.Fatalf("Judgement casts = 0, want > 0")
	}

	player0 := result.RaidMetrics.Parties[0].Players[0]
	var gain float64
	found := false
	for _, resource := range player0.Resources {
		if resource.Id.GetSpellId() == 1311074 { // Sanctified Judgement (sim/paladin/talents.go's sanctifiedJudgementActionID)
			gain = resource.Gain
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Sanctified Judgement never appears in the resource metrics -- it never refunded mana")
	}
	if gain <= 0 {
		t.Errorf("Sanctified Judgement mana gain = %v, want > 0", gain)
	}
}
