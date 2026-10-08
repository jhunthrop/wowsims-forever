package healing

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Talent positions, 1-based within their tree, in the client's tier-then-
// column order (sim/priest/talents_auto_gen.go TalentNodeIDs).
const (
	discTree, holyTree = 0, 1

	twinDisciplines       = 3
	improvedPowerWordShld = 6
	mentalAgility         = 8
	innerFocus            = 9
	mentalStrength        = 12
	soulWarding           = 13
	renewedHope           = 16
	divineAegis           = 17
	powerInfusion         = 18

	improvedRenew     = 2
	divineFury        = 5
	inspiration       = 8
	improvedHealing   = 10
	litanyOfLight     = 13
	spiritualGuidance = 15
	spiritualHealing  = 16
)

// onePoint builds a talent string with rank points in one talent.
func onePoint(tree, position int, rank int) string {
	trees := []string{"", "", ""}
	trees[tree] = strings.Repeat("0", position-1) + string(rune('0'+rank))
	return strings.Join(trees[:tree+1], "-")
}

// tankHealed is what spell heals the tank for with no crit.
func tankHealed(t *testing.T, talents string, pick func(*HealingPriest) *core.Spell) float64 {
	t.Helper()
	sim, agent := healerSim(t, 60, talents)
	return healedBy(sim, pick(agent), agent.GetMainTarget())
}

func assertRatio(t *testing.T, label string, got, base, want float64) {
	t.Helper()
	if base == 0 || math.Abs(got/base-want) > 0.0005 {
		t.Errorf("%s: %.4f x the base (%.2f over %.2f), want %.4f x", label, got/base, got, base, want)
	}
}

// renewTotal is what a full Renew heals the tank for with no crit.
func renewTotal(t *testing.T, talents string) float64 {
	t.Helper()
	sim, agent := healerSim(t, 60, talents)
	target := agent.GetMainTarget()
	renew := agent.Renew[10]
	renew.ApplyEffects(sim, target, renew)
	start := sim.CurrentTime
	for sim.CurrentTime < start+16*time.Second && !sim.Step() {
	}
	return renew.SpellMetrics[target.UnitIndex].TotalHealing
}

func TestImprovedRenewAddsFivePercentARank(t *testing.T) {
	base := renewTotal(t, "")
	assertRatio(t, "Improved Renew 3/3", renewTotal(t, onePoint(holyTree, improvedRenew, 3)), base, 1.15)
	assertRatio(t, "Improved Renew 1/3", renewTotal(t, onePoint(holyTree, improvedRenew, 1)), base, 1.05)
}

func TestTwinDisciplinesBoostsInstantHealsButNotTheShield(t *testing.T) {
	talents := onePoint(discTree, twinDisciplines, 5)
	assertRatio(t, "Renew", renewTotal(t, talents), renewTotal(t, ""), 1.05)
	flash := func(a *HealingPriest) *core.Spell { return a.FlashHeal[7] }
	assertRatio(t, "Flash Heal (not instant)", tankHealed(t, talents, flash), tankHealed(t, "", flash), 1)
	if shieldAmount(t, talents) != shieldAmount(t, "") {
		t.Error("Twin Disciplines changed a Power Word: Shield, which absorbs and does not heal")
	}
}

// shieldAmount is what a rank 10 Power Word: Shield absorbs.
func shieldAmount(t *testing.T, talents string) float64 {
	t.Helper()
	sim, agent := healerSim(t, 60, talents)
	target := agent.GetMainTarget()
	shield := agent.PowerWordShield[10]
	shield.ApplyEffects(sim, target, shield)
	return shield.Shield(target).Remaining()
}

func TestImprovedPowerWordShieldFollowsTheClientsRanks(t *testing.T) {
	base := shieldAmount(t, "")
	for rank, want := range map[int]float64{1: 1.07, 2: 1.14, 3: 1.20} {
		assertRatio(t, "Improved Power Word: Shield", shieldAmount(t, onePoint(discTree, improvedPowerWordShld, rank)), base, want)
	}
}

func TestSpiritualHealingRaisesHealsButNotShields(t *testing.T) {
	talents := onePoint(holyTree, spiritualHealing, 3)
	greater := func(a *HealingPriest) *core.Spell { return a.GreaterHeal[5] }
	assertRatio(t, "Greater Heal", tankHealed(t, talents, greater), tankHealed(t, "", greater), 1.10)
	assertRatio(t, "Renew", renewTotal(t, talents), renewTotal(t, ""), 1.10)
	if shieldAmount(t, talents) != shieldAmount(t, "") {
		t.Error("Spiritual Healing changed a Power Word: Shield, which absorbs and does not heal")
	}
}

func TestSpiritualGuidanceGrantsFivePercentOfSpiritAsHealing(t *testing.T) {
	_, bare := healerSim(t, 60, "")
	_, guided := healerSim(t, 60, onePoint(holyTree, spiritualGuidance, 5))
	spirit := guided.GetStat(stats.Spirit)
	if got, want := guided.GetStat(stats.HealingPower)-bare.GetStat(stats.HealingPower), 0.25*spirit; math.Abs(got-want) > 0.01 {
		t.Errorf("healing power rose %.2f, want 25%% of %.0f spirit = %.2f", got, spirit, want)
	}
	// The damage half is the one-third rule on the healing stat: 8.33% of
	// spirit, the client's "8%" rounded.
	damage := guided.GetStat(stats.SpellDamage) - bare.GetStat(stats.SpellDamage)
	if want := 0.25 * spirit / 3; math.Abs(damage-want) > 0.01 {
		t.Errorf("spell damage rose %.2f, want a third of the healing, %.2f", damage, want)
	}
}

func TestMentalStrengthAddsThreePercentIntellectARank(t *testing.T) {
	_, bare := healerSim(t, 60, "")
	_, strong := healerSim(t, 60, onePoint(discTree, mentalStrength, 5))
	assertRatio(t, "Intellect", strong.GetStat(stats.Intellect), bare.GetStat(stats.Intellect), 1.15)
}

func cost(spell *core.Spell) float64 { return spell.Cost.GetCurrentCost() }

func TestImprovedHealingCutsTheCostOfTheNamedHeals(t *testing.T) {
	_, bare := healerSim(t, 60, "")
	_, improved := healerSim(t, 60, onePoint(holyTree, improvedHealing, 3))
	assertRatio(t, "Greater Heal cost", cost(improved.GreaterHeal[5]), cost(bare.GreaterHeal[5]), 0.85)
	assertRatio(t, "Heal cost", cost(improved.Heal[4]), cost(bare.Heal[4]), 0.85)
	if cost(improved.FlashHeal[7]) != cost(bare.FlashHeal[7]) {
		t.Error("Improved Healing cut Flash Heal, which it does not name")
	}
}

func TestMentalAgilityCutsInstantsAndSmiteButNotCastTimeHeals(t *testing.T) {
	_, bare := healerSim(t, 60, "")
	for rank, want := range map[int]float64{1: 0.97, 2: 0.93, 3: 0.90} {
		_, agile := healerSim(t, 60, onePoint(discTree, mentalAgility, rank))
		assertRatio(t, "Renew cost", cost(agile.Renew[10]), cost(bare.Renew[10]), want)
		assertRatio(t, "Power Word: Shield cost", cost(agile.PowerWordShield[10]), cost(bare.PowerWordShield[10]), want)
		assertRatio(t, "Smite cost", cost(agile.Smite[len(agile.Smite)-1]), cost(bare.Smite[len(bare.Smite)-1]), want)
		if cost(agile.FlashHeal[7]) != cost(bare.FlashHeal[7]) {
			t.Error("Mental Agility cut Flash Heal, which has a cast time")
		}
	}
}

func TestDivineFuryShortensHealGreaterHealAndSmite(t *testing.T) {
	_, bare := healerSim(t, 60, "")
	_, furious := healerSim(t, 60, onePoint(holyTree, divineFury, 5))
	if got, want := furious.GreaterHeal[5].DefaultCast.CastTime, bare.GreaterHeal[5].DefaultCast.CastTime-500*time.Millisecond; got != want {
		t.Errorf("Greater Heal casts in %v, want %v", got, want)
	}
	if got, want := furious.Heal[4].DefaultCast.CastTime, 2500*time.Millisecond; got != want {
		t.Errorf("Heal casts in %v, want %v", got, want)
	}
	if got, want := furious.FlashHeal[7].DefaultCast.CastTime, bare.FlashHeal[7].DefaultCast.CastTime; got != want {
		t.Errorf("Flash Heal casts in %v, want the unchanged %v", got, want)
	}
	top := len(bare.Smite) - 1
	if got, want := furious.Smite[top].DefaultCast.CastTime, bare.Smite[top].DefaultCast.CastTime-500*time.Millisecond; got != want {
		t.Errorf("Smite casts in %v, want %v", got, want)
	}
}

func TestSoulWardingRemovesTheShieldCooldownAndCutsItsCost(t *testing.T) {
	_, bare := healerSim(t, 60, "")
	_, warded := healerSim(t, 60, onePoint(discTree, soulWarding, 1))
	shield := warded.PowerWordShield[10]
	if got := shield.CD.Duration; got != 0 {
		t.Errorf("Power Word: Shield cooldown %v, want 0 (4 s less 4 s)", got)
	}
	assertRatio(t, "Power Word: Shield cost", cost(shield), cost(bare.PowerWordShield[10]), 0.85)
}

func TestDivineAegisShieldsAFifthOfACritByRank(t *testing.T) {
	sim, agent := healerSim(t, 60, onePoint(discTree, divineAegis, 3))
	target := agent.GetMainTarget()
	spell := agent.FlashHeal[7]
	spell.BonusCritRating += 1000 * core.CritRatingPerCritChance
	before := spell.SpellMetrics[target.UnitIndex].TotalCritHealing
	spell.ApplyEffects(sim, target, spell)
	crit := spell.SpellMetrics[target.UnitIndex].TotalCritHealing - before

	aegis := agent.GetSpell(core.ActionID{SpellID: 431624})
	if aegis == nil {
		t.Fatal("Divine Aegis registered no shield spell")
	}
	if got, want := aegis.Shield(target).Remaining(), 0.15*crit; math.Abs(got-want) > 0.01 {
		t.Errorf("Divine Aegis absorbs %.2f, want 15%% of the %.2f crit = %.2f", got, crit, want)
	}
	spell.ApplyEffects(sim, target, spell)
	if aegis.Shield(target).Remaining() <= 0.15*crit {
		t.Error("a second crit did not add to the shield")
	}
}

func TestInspirationRaisesArmorOfWhoeverIsCritHealed(t *testing.T) {
	sim, agent := healerSim(t, 60, onePoint(holyTree, inspiration, 3))
	target := agent.GetMainTarget()
	target.AddStatDynamic(sim, stats.Armor, 1000)
	spell := agent.FlashHeal[7]
	spell.BonusCritRating += noCrit
	spell.ApplyEffects(sim, target, spell)
	if got := target.GetStat(stats.Armor); got != 1000 {
		t.Fatalf("a heal that did not crit moved armor to %v", got)
	}
	spell.BonusCritRating -= noCrit
	spell.BonusCritRating += 1000 * core.CritRatingPerCritChance
	spell.ApplyEffects(sim, target, spell)
	if got := target.GetStat(stats.Armor); math.Abs(got-1250) > 0.01 {
		t.Errorf("armor after a crit heal %v, want 1250 (+25%%)", got)
	}
}

func TestInspirationIgnoresPeriodicHeals(t *testing.T) {
	sim, agent := healerSim(t, 60, onePoint(holyTree, inspiration, 3))
	target := agent.GetMainTarget()
	target.AddStatDynamic(sim, stats.Armor, 1000)
	renew := agent.Renew[10]
	renew.BonusCritRating += 1000 * core.CritRatingPerCritChance
	renew.ApplyEffects(sim, target, renew)
	for sim.CurrentTime < 16*time.Second && !sim.Step() {
	}
	if got := target.GetStat(stats.Armor); got != 1000 {
		t.Errorf("Renew ticks raised armor to %v", got)
	}
}

// critRate casts spell on target many times and returns the share that
// crit.
// beforeEach runs ahead of every cast.
func critRate(sim *core.Simulation, spell *core.Spell, target *core.Unit, casts int, beforeEach func()) float64 {
	before := spell.SpellMetrics[target.UnitIndex]
	for i := 0; i < casts; i++ {
		beforeEach()
		spell.ApplyEffects(sim, target, spell)
	}
	after := spell.SpellMetrics[target.UnitIndex]
	return float64(after.Crits-before.Crits) / float64(casts)
}

func TestRenewedHopeAddsCritOnWeakenedSoulTargetsAndShortensIt(t *testing.T) {
	sim, agent := healerSim(t, 60, onePoint(discTree, renewedHope, 5))
	shielded := agent.GetMainTarget()
	other := &sim.Raid.Parties[0].Players[1].GetCharacter().Unit
	flash := agent.FlashHeal[7]

	agent.PowerWordShield[10].ApplyEffects(sim, shielded, agent.PowerWordShield[10])
	weakened := shielded.GetAura("Weakened Soul")
	if weakened == nil || weakened.RemainingDuration(sim) != 15*time.Second {
		t.Fatalf("shielding did not apply a 15 s Weakened Soul: %v", weakened)
	}
	const casts = 4000
	keepWeakened := func() { weakened.Activate(sim); weakened.Refresh(sim) }
	withSoul := critRate(sim, flash, shielded, casts, keepWeakened)
	withoutSoul := critRate(sim, flash, other, casts, func() {})
	if diff := withSoul - withoutSoul; math.Abs(diff-0.10) > 0.03 {
		t.Errorf("crit chance on a Weakened Soul target is %.3f higher, want 0.10 (+2%% a rank at 5 ranks)", diff)
	}

	keepWeakened()
	flash.ApplyEffects(sim, shielded, flash)
	if got := weakened.RemainingDuration(sim); got != 10*time.Second {
		t.Errorf("Weakened Soul has %v left after a Flash Heal, want 10 s (5 s less)", got)
	}
}

func TestLitanyOfLightReturnsManaOnlyWhenTheHealChanges(t *testing.T) {
	sim, agent := healerSim(t, 60, onePoint(holyTree, litanyOfLight, 2))
	litany := agent.GetAura("Litany of Light")
	if litany == nil {
		t.Fatal("Litany of Light registered no aura")
	}
	flash, greater, shield := agent.FlashHeal[7], agent.GreaterHeal[5], agent.PowerWordShield[10]
	agent.SpendMana(sim, 3000, agent.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionPotion}))
	mana := func() float64 { return agent.CurrentMana() }
	step := func(spell *core.Spell) float64 {
		before := mana()
		litany.OnCastComplete(litany, sim, spell)
		return mana() - before
	}
	if got := step(flash); got != 0 {
		t.Errorf("the first heal returned %v mana, want none", got)
	}
	if got := step(flash); got != 0 {
		t.Errorf("repeating the same heal returned %v mana, want none", got)
	}
	if got, want := step(greater), 0.10*greater.Cost.BaseCost; math.Abs(got-want) > 0.01 {
		t.Errorf("a different heal returned %v mana, want 10%% of %v = %v", got, greater.Cost.BaseCost, want)
	}
	if got := step(shield); got != 0 {
		t.Errorf("a shield returned %v mana, want none: it is not a heal", got)
	}
}

func TestInnerFocusMakesTheNextHealFreeAndCritsMore(t *testing.T) {
	sim, agent := healerSim(t, 60, onePoint(discTree, innerFocus, 1))
	spell := agent.FlashHeal[7]
	full, crit := cost(spell), spell.BonusCritRating
	agent.InnerFocusAura.Activate(sim)
	if cost(spell) != 0 {
		t.Errorf("a heal under Inner Focus costs %v, want 0 (was %v)", cost(spell), full)
	}
	if got, want := spell.BonusCritRating-crit, 25*float64(core.CritRatingPerCritChance); math.Abs(got-want) > 1e-9 {
		t.Errorf("Inner Focus added %v crit rating, want %v", got, want)
	}
	agent.InnerFocusAura.Deactivate(sim)
	if cost(spell) != full {
		t.Errorf("the cost did not come back after Inner Focus: %v", cost(spell))
	}
}

func TestPowerInfusionTargetGetsTwentyPercentMoreHealingAndDamage(t *testing.T) {
	options := &proto.HealingPriest_Options{PowerInfusionTarget: &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0}}
	sim, agent := healerSimWith(t, 60, onePoint(discTree, powerInfusion, 1), options)
	if agent.PowerInfusion == nil {
		t.Fatal("Power Infusion is not registered with the talent and a target")
	}
	agent.PowerInfusion.ApplyEffects(sim, &agent.Unit, agent.PowerInfusion)
	if got := agent.PseudoStats.HealingDealtMultiplier; math.Abs(got-1.2) > 1e-9 {
		t.Errorf("healing done multiplier %v, want 1.2", got)
	}
	agent.GetAura("Power Infusion-" + core.ActionID{SpellID: 10060, Tag: agent.Index}.String()).Deactivate(sim)
	if agent.PseudoStats.HealingDealtMultiplier != 1 {
		t.Errorf("healing multiplier %v after Power Infusion ended, want 1", agent.PseudoStats.HealingDealtMultiplier)
	}
}

func TestPowerInfusionNeedsATargetAndTheTalent(t *testing.T) {
	_, noTarget := healerSim(t, 60, onePoint(discTree, powerInfusion, 1))
	if noTarget.PowerInfusion != nil {
		t.Error("Power Infusion registered with no target chosen")
	}
	self := &proto.HealingPriest_Options{PowerInfusionTarget: &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0}}
	_, noTalent := healerSimWith(t, 60, "", self)
	if noTalent.PowerInfusion != nil {
		t.Error("Power Infusion registered without its talent")
	}
}

func TestTheInnerFireOptionBuffsBeforeThePull(t *testing.T) {
	player := healer(60, "", &proto.HealingPriest_Options{UseInnerFire: true}, &proto.APLRotation{})
	sim, agent := agentSim(t, player)
	for sim.CurrentTime < 0 && !sim.Step() {
	}
	if aura := agent.GetAura("Inner Fire (Rank 6)"); aura == nil || !aura.IsActive() {
		t.Error("Inner Fire is not up after the prepull with the option on")
	}
}
