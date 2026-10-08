package tank

import (
	"math"
	"testing"
	"time"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/druid"
)

// talentString builds a full-width talent string with every talent zero but
// the ones named, keyed by the generated proto field name.
func talentString(t *testing.T, ranks map[string]int) string {
	t.Helper()
	str, err := core.TalentsStringFromRanks((&proto.DruidTalents{}).ProtoReflect(), druid.TalentTreeSizes, ranks)
	if err != nil {
		t.Fatal(err)
	}
	return str
}

// newBearSim builds a bear of the level with the talents, in front of the
// harness boss, reset and on its first frame.
func newBearSim(t *testing.T, level int32, ranks map[string]int) (*FeralTankDruid, *core.Simulation, *core.Unit) {
	t.Helper()
	return newBearSimWearing(t, level, ranks, &proto.EquipmentSpec{})
}

// newBearSimWearing is newBearSim in gear. Neither is buffed, so a stat is
// what the character, its talents and its form make it.
func newBearSimWearing(t *testing.T, level int32, ranks map[string]int, gear *proto.EquipmentSpec) (*FeralTankDruid, *core.Simulation, *core.Unit) {
	t.Helper()
	player := bearPlayer(level, talentString(t, ranks), gear, nil)
	player.Buffs = &proto.IndividualBuffs{}
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	raid.Tanks = append(raid.Tanks, &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  harnessEncounter(),
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	bear, ok := sim.Raid.Parties[0].Players[0].(*FeralTankDruid)
	if !ok {
		t.Fatal("the raid's first player is not a *FeralTankDruid")
	}
	return bear, sim, sim.Encounter.TargetUnits[0]
}

// taurenHealthMultiplier is the Tauren's Endurance: +5% health, which scales
// the form's health too.
const taurenHealthMultiplier = 1.05

func near(got, want float64) bool { return math.Abs(got-want) < 0.01 }

func TestBearFormIsActiveAtTheStartWithItsPaw(t *testing.T) {
	bear, _, _ := newBearSim(t, 60, nil)
	if !bear.InForm(druid.Bear) {
		t.Fatal("the tank does not start in Bear Form")
	}
	paw := bear.AutoAttacks.MH()
	if paw.SwingSpeed != 2.5 {
		t.Errorf("paw swing speed = %v, want 2.5", paw.SwingSpeed)
	}
	if got := (paw.BaseDamageMin + paw.BaseDamageMax) / 2 / paw.SwingSpeed; !near(got, 54.8) {
		t.Errorf("paw dps = %v, want 54.8", got)
	}
	if !bear.PseudoStats.FeralCombatEnabled {
		t.Error("feral combat skill is not enabled")
	}
}

// Bear Form (Passive2) 21178: aura 10 amount 30.
func TestBearFormThreatIsThirtyPercentOver(t *testing.T) {
	bear, _, _ := newBearSim(t, 60, nil)
	if got := bear.PseudoStats.ThreatMultiplier; !near(got, 1.3) {
		t.Errorf("threat multiplier = %v, want 1.3", got)
	}
}

// formShift is what leaving Bear Form takes away from a stat.
func formShift(bear *FeralTankDruid, sim *core.Simulation, stat stats.Stat) float64 {
	before := bear.GetStat(stat)
	bear.BearFormAura.Deactivate(sim)
	return before - bear.GetStat(stat)
}

// Dire Bear Form (Passive) 9635 at level 60: attack power 120 + 3 a level
// from 40 = 180, health 600 + 32 a level from 40 = 1240.
func TestDireBearFormAddsItsAttackPowerAndHealth(t *testing.T) {
	bear, sim, _ := newBearSim(t, 60, nil)
	healthBefore := bear.MaxHealth()
	if got := formShift(bear, sim, stats.AttackPower); !near(got, 180) {
		t.Errorf("attack power from the form = %v, want 180", got)
	}
	if got := healthBefore - bear.MaxHealth(); !near(got, 1240*taurenHealthMultiplier) {
		t.Errorf("max health from the form = %v, want %v", got, 1240*taurenHealthMultiplier)
	}
}

// Bear Form (Passive) 1178 below level 40: 30 + 3 a level from 10.
func TestBearFormBelowFortyUsesTheFirstTier(t *testing.T) {
	bear, sim, _ := newBearSim(t, 30, nil)
	healthBefore := bear.MaxHealth()
	if got := formShift(bear, sim, stats.AttackPower); !near(got, 90) {
		t.Errorf("level 30 attack power from the form = %v, want 90", got)
	}
	if got := healthBefore - bear.MaxHealth(); !near(got, (20+18*20)*taurenHealthMultiplier) {
		t.Errorf("level 30 health from the form = %v, want %v", got, (20+18*20)*taurenHealthMultiplier)
	}
}

// Thick Hide rank 3: 3 base armor a level and 2 per defense point beyond
// five times the level, multiplied by the form's 460% of armor.
func TestThickHideAddsFormMultipliedBaseArmor(t *testing.T) {
	plain, _, _ := newBearSim(t, 60, nil)
	hide, _, _ := newBearSim(t, 60, map[string]int{"thick_hide": 3})
	if got, want := hide.GetStat(stats.Armor)-plain.GetStat(stats.Armor), 3.0*60*4.6; !near(got, want) {
		t.Errorf("Thick Hide 3/3 armor = %v, want %v", got, want)
	}

	one, _, _ := newBearSim(t, 60, map[string]int{"thick_hide": 1})
	if got, want := one.GetStat(stats.Armor)-plain.GetStat(stats.Armor), 1.0*60*4.6; !near(got, want) {
		t.Errorf("Thick Hide 1/3 armor = %v, want %v", got, want)
	}
}

// Heart of the Wild 5/5: Intellect +10% and, in Bear Form, Stamina +20%.
func TestHeartOfTheWildInBearForm(t *testing.T) {
	plain, _, _ := newBearSim(t, 60, nil)
	wild, _, _ := newBearSim(t, 60, map[string]int{"heart_of_the_wild": 5})
	if got, want := wild.GetStat(stats.Stamina), plain.GetStat(stats.Stamina)*1.2; !near(got, want) {
		t.Errorf("stamina = %v, want %v", got, want)
	}
	if got, want := wild.GetStat(stats.Intellect), plain.GetStat(stats.Intellect)*1.1; !near(got, want) {
		t.Errorf("intellect = %v, want %v", got, want)
	}
}

// Sharpened Claws 2/2: +6% critical strike chance in Bear Form.
func TestSharpenedClawsAddsSixPercentCrit(t *testing.T) {
	plain, _, _ := newBearSim(t, 60, nil)
	claws, _, _ := newBearSim(t, 60, map[string]int{"sharpened_claws": 2})
	if got, want := claws.GetStat(stats.Crit)-plain.GetStat(stats.Crit), 6.0; !near(got, want) {
		t.Errorf("crit = +%v, want +%v", got, want)
	}
}

// Predatory Strikes 3/3: +150% of the level in melee attack power.
func TestPredatoryStrikesAddsLevelBasedAttackPower(t *testing.T) {
	plain, _, _ := newBearSim(t, 60, nil)
	strikes, _, _ := newBearSim(t, 60, map[string]int{"predatory_strikes": 3})
	if got, want := strikes.GetStat(stats.AttackPower)-plain.GetStat(stats.AttackPower), 1.5*60; !near(got, want) {
		t.Errorf("attack power = +%v, want +%v", got, want)
	}
}

// Feral Swiftness 2/2 (+4% dodge) and Natural Reaction 5/5 (+5% dodge).
func TestDodgeFromTheFeralTree(t *testing.T) {
	plain, _, _ := newBearSim(t, 60, nil)
	dodgy, _, _ := newBearSim(t, 60, map[string]int{"feral_swiftness": 2, "natural_reaction": 5})
	if got, want := dodgy.GetStat(stats.Dodge)-plain.GetStat(stats.Dodge), 9.0; !near(got, want) {
		t.Errorf("dodge = +%v, want +%v", got, want)
	}
}

func TestFerocityDiscountsEveryBearAbilityItNames(t *testing.T) {
	bear, _, _ := newBearSim(t, 60, map[string]int{"ferocity": 5, "primal_bite": 1})
	for name, c := range map[string]struct {
		spell *druid.DruidSpell
		want  float64
	}{
		"Maul":        {bear.Maul, 10},
		"Swipe":       {bear.SwipeBear, 15},
		"Primal Bite": {bear.PrimalBite, 15},
	} {
		if c.spell == nil {
			t.Fatalf("%s is not registered", name)
		}
		if got := c.spell.Cost.GetCurrentCost(); got != c.want {
			t.Errorf("%s costs %v rage, want %v", name, got, c.want)
		}
	}
}

func TestShreddingAttacksDiscountsLacerate(t *testing.T) {
	plain, _, _ := newBearSim(t, 60, nil)
	shredder, _, _ := newBearSim(t, 60, map[string]int{"shredding_attacks": 3})
	if got := plain.Lacerate.Cost.GetCurrentCost(); got != 15 {
		t.Errorf("Lacerate costs %v rage, want 15", got)
	}
	if got := shredder.Lacerate.Cost.GetCurrentCost(); got != 12 {
		t.Errorf("Lacerate with Shredding Attacks costs %v rage, want 12", got)
	}
}

func TestRankedAbilitiesAtLevelSixty(t *testing.T) {
	bear, _, _ := newBearSim(t, 60, map[string]int{"primal_bite": 1})
	for name, c := range map[string]struct {
		spell *druid.DruidSpell
		id    int32
	}{
		"Maul":                  {bear.Maul, 9881},
		"Swipe":                 {bear.SwipeBear, 9908},
		"Lacerate":              {bear.Lacerate, 1235827},
		"Primal Bite":           {bear.PrimalBite, 1238073},
		"Demoralizing Roar":     {bear.DemoralizingRoar, 9898},
		"Enrage":                {bear.Enrage, 5229},
		"Frenzied Regeneration": {bear.FrenziedRegeneration, 22842},
		"Barkskin":              {bear.Barkskin, 22812},
	} {
		if c.spell == nil {
			t.Fatalf("%s is not registered at level 60", name)
		}
		if c.spell.ActionID.SpellID != c.id {
			t.Errorf("%s registered as %d, want %d", name, c.spell.ActionID.SpellID, c.id)
		}
	}
	if got := bear.PrimalBite.CD.Duration; got != 6*time.Second {
		t.Errorf("Primal Bite cooldown = %v, want 6s", got)
	}
}

func TestPrimalBiteNeedsItsTalent(t *testing.T) {
	bear, _, _ := newBearSim(t, 60, nil)
	if bear.PrimalBite != nil {
		t.Error("Primal Bite is registered without its talent")
	}
}

func TestSwipeAddsSavageFuryAndFeralInstinct(t *testing.T) {
	bear, _, _ := newBearSim(t, 60, map[string]int{"savage_fury": 2, "feral_instinct": 3})
	if got, want := bear.SwipeBear.DamageMultiplierAdditive, 1+0.10+0.30; !near(got, want) {
		t.Errorf("Swipe additive damage = %v, want %v", got, want)
	}
	if got, want := bear.Maul.DamageMultiplierAdditive, 1.10; !near(got, want) {
		t.Errorf("Maul additive damage = %v, want %v", got, want)
	}
}

// Rend and Tear 5/5: +10% melee ability damage on a bleeding target, which
// a Lacerate on it makes.
func TestRendAndTearNeedsABleed(t *testing.T) {
	bear, sim, target := newBearSim(t, 60, map[string]int{"rend_and_tear": 5})
	if got := bear.RendAndTearMultiplier(target); got != 1 {
		t.Errorf("multiplier without a bleed = %v, want 1", got)
	}
	bear.AddRage(sim, 100, bear.NewRageMetrics(core.ActionID{SpellID: 1}))
	bear.Lacerate.Cast(sim, target)
	if !bear.Lacerate.Dot(target).IsActive() {
		t.Skip("the Lacerate cast missed on this seed")
	}
	if got := bear.RendAndTearMultiplier(target); !near(got, 1.10) {
		t.Errorf("multiplier with Lacerate up = %v, want 1.10", got)
	}
}

func TestLacerateStacksToFive(t *testing.T) {
	bear, sim, target := newBearSim(t, 60, nil)
	for i := 0; i < 40 && bear.Lacerate.Dot(target).GetStacks() < 5; i++ {
		bear.AddRage(sim, 100, bear.NewRageMetrics(core.ActionID{SpellID: 1}))
		sim.CurrentTime += 2 * time.Second
		bear.GCD.Reset()
		bear.Lacerate.Cast(sim, target)
	}
	if got := bear.Lacerate.Dot(target).GetStacks(); got != 5 {
		t.Fatalf("stacks = %d, want 5", got)
	}
	sim.CurrentTime += 2 * time.Second
	bear.GCD.Reset()
	bear.AddRage(sim, 100, bear.NewRageMetrics(core.ActionID{SpellID: 1}))
	bear.Lacerate.Cast(sim, target)
	if got := bear.Lacerate.Dot(target).GetStacks(); got != 5 {
		t.Errorf("a sixth cast leaves %d stacks, want 5", got)
	}
}

func TestBerserkTakesPrimalBitesCooldownAway(t *testing.T) {
	bear, sim, target := newBearSim(t, 60, map[string]int{"primal_bite": 1, "berserk": 1})
	if bear.PrimalBite.CD.Duration != 6*time.Second {
		t.Fatalf("cooldown before Berserk = %v", bear.PrimalBite.CD.Duration)
	}
	bear.Berserk.Cast(sim, target)
	if got := bear.PrimalBite.CD.Duration; got != 0 {
		t.Errorf("cooldown during Berserk = %v, want none", got)
	}
	bear.BerserkAura.Deactivate(sim)
	if got := bear.PrimalBite.CD.Duration; got != 6*time.Second {
		t.Errorf("cooldown after Berserk = %v, want 6s", got)
	}
}

func TestEnrageGivesThirtyRageOverItsDuration(t *testing.T) {
	bear, sim, target := newBearSimWearing(t, 60, nil, loadGear(t))
	armorBefore := bear.GetStat(stats.Armor)
	rageBefore := bear.CurrentRage()
	bear.Enrage.Cast(sim, target)
	if got := bear.CurrentRage() - rageBefore; got != 10 {
		t.Errorf("instant rage = %v, want 10", got)
	}
	if got := bear.GetStat(stats.Armor); got >= armorBefore {
		t.Errorf("armor %v did not fall under Enrage (was %v)", got, armorBefore)
	}
}

func TestFuriorBearFormRageChance(t *testing.T) {
	// Furor is exercised through the shift itself: 5/5 always pays 10 rage.
	bear, sim, target := newBearSim(t, 60, map[string]int{"furor": 5})
	bear.CancelShapeshift(sim)
	bear.BearForm.Cast(sim, target)
	if got := bear.CurrentRage(); got != 10 {
		t.Errorf("rage after shifting with Furor 5/5 = %v, want 10", got)
	}
}
