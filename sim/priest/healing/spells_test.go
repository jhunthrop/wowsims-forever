package healing

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/spellconst"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/priest"
)

const clientPriestPath = "../../core/testdata/conformance/client/priest.json"

// noCrit keeps a spell from critting, so a test can read its plain amount.
const noCrit = -1000 * core.CritRatingPerCritChance

// Single-talent builds, so a test reads one spell without the Holy and
// Discipline builds' other bonuses: one point in Penance (position 15 of
// Discipline) and one in Holy Nova (position 6 of Holy).
const (
	penanceOnly  = "000000000000001"
	holyNovaOnly = "-000001"
)

func loadClient(t *testing.T) spellconst.Class {
	t.Helper()
	client, err := spellconst.Load(clientPriestPath)
	if err != nil {
		t.Fatalf("loading the client's priest spellconst: %v", err)
	}
	return client
}

// healedBy casts spell on target without a crit and returns what it healed.
func healedBy(sim *core.Simulation, spell *core.Spell, target *core.Unit) float64 {
	spell.BonusCritRating += noCrit
	before := spell.SpellMetrics[target.UnitIndex].TotalHealing
	spell.ApplyEffects(sim, target, spell)
	spell.BonusCritRating -= noCrit
	return spell.SpellMetrics[target.UnitIndex].TotalHealing - before
}

// registered lists the ranks of a ladder the priest has, rank first.
func registered(spells []*core.Spell) map[int]*core.Spell {
	ranks := map[int]*core.Spell{}
	for rank, spell := range spells {
		if spell != nil {
			ranks[rank] = spell
		}
	}
	return ranks
}

// assertWithin fails unless got is inside [low, high], widened by a rounding
// allowance of one point for the client's float centre and variance.
func assertWithin(t *testing.T, label string, got, low, high float64) {
	t.Helper()
	const allowance = 1.0
	if got < low-allowance || got > high+allowance {
		t.Errorf("%s: healed %.2f, want %.2f-%.2f", label, got, low, high)
	}
}

// directHealLadders are the heals that land one roll on one target.
func directHealLadders(agent *HealingPriest) map[string][]*core.Spell {
	return map[string][]*core.Spell{
		"Lesser Heal":  agent.LesserHeal,
		"Heal":         agent.Heal,
		"Flash Heal":   agent.FlashHeal,
		"Greater Heal": agent.GreaterHeal,
	}
}

// TestEveryRankHealsTheClientRollPlusCoefficientTimesHealingPower casts
// each registered rank of the direct heals and checks the heal against the
// client's roll at level 60 plus the coefficient times the healing stat.
func TestEveryRankHealsTheClientRollPlusCoefficientTimesHealingPower(t *testing.T) {
	sim, agent := healerSim(t, 60, "")
	client := loadClient(t)
	target := agent.GetMainTarget()
	for name, ladder := range directHealLadders(agent) {
		ranks := registered(ladder)
		if len(ranks) == 0 {
			t.Fatalf("%s: no rank registered at level 60", name)
		}
		for rank, spell := range ranks {
			row, ok := client.ByID(spell.SpellID)
			if !ok {
				t.Fatalf("%s rank %d: spell %d is not in the client table", name, rank, spell.SpellID)
			}
			low, high, _ := row.DamageRange(0, 60)
			bonus := row.Effects[0].ResolvedSPCoefficient * spell.HealingPower(target)
			for draw := 0; draw < 5; draw++ {
				assertWithin(t, name, healedBy(sim, spell, target), low+bonus, high+bonus)
			}
		}
	}
}

// TestHealingPowerIsTheHealingStatAlone: a heal reads the healing stat and
// not spell power, which is damage only in the site's item data.
func TestHealingPowerIsTheHealingStatAlone(t *testing.T) {
	_, agent := healerSim(t, 60, "")
	spell := agent.FlashHeal[7]
	if got, want := spell.HealingPower(agent.GetMainTarget()), raidHealerStats[stats.HealingPower]; got != want {
		t.Errorf("healing power %v, want the %v the gear gives", got, want)
	}
}

// TestACrittingHealMultipliesByTheSpellCritMultiplier shows a forced crit
// lands for the multiplier on top of the plain heal.
func TestACrittingHealMultipliesByTheSpellCritMultiplier(t *testing.T) {
	sim, agent := healerSim(t, 60, "")
	target := agent.GetMainTarget()
	spell := agent.GreaterHeal[5]
	spell.BonusCritRating += 1000 * core.CritRatingPerCritChance
	before := spell.SpellMetrics[target.UnitIndex]
	spell.ApplyEffects(sim, target, spell)
	after := spell.SpellMetrics[target.UnitIndex]
	spell.BonusCritRating -= 1000 * core.CritRatingPerCritChance
	if after.Crits != before.Crits+1 {
		t.Fatalf("a heal with certain crit did not crit")
	}
	row, _ := loadClient(t).ByID(spell.SpellID)
	low, high, _ := row.DamageRange(0, 60)
	bonus := row.Effects[0].ResolvedSPCoefficient * spell.HealingPower(target)
	crit := after.TotalCritHealing - before.TotalCritHealing
	assertWithin(t, "Greater Heal crit", crit/1.5, low+bonus, high+bonus)
}

// TestRanksRegisterOnlyWhenTheLevelReachesThem is the level-awareness
// check: a level-30 priest has the ranks it has learned and nothing above.
func TestRanksRegisterOnlyWhenTheLevelReachesThem(t *testing.T) {
	_, agent := healerSim(t, 30, HolyTalents)
	want := map[string][]int{
		"Lesser Heal": {1, 2, 3},
		"Heal":        {1, 2, 3},
		"Flash Heal":  {1, 2},
	}
	for name, ladder := range directHealLadders(agent) {
		got := registered(ladder)
		if wanted, ok := want[name]; ok {
			for _, rank := range wanted {
				if got[rank] == nil {
					t.Errorf("%s rank %d missing at level 30", name, rank)
				}
			}
			if len(got) != len(wanted) {
				t.Errorf("%s has %d ranks at level 30, want %d", name, len(got), len(wanted))
			}
		} else if len(got) != 0 {
			t.Errorf("%s has %d ranks at level 30, want none (first rank is level 40)", name, len(got))
		}
	}
	if len(registered(agent.Renew)) != 4 || len(registered(agent.PowerWordShield)) != 5 {
		t.Errorf("level 30: Renew %d ranks, Power Word: Shield %d ranks, want 4 and 5", len(registered(agent.Renew)), len(registered(agent.PowerWordShield)))
	}
	if len(registered(agent.PrayerOfHealing)) != 1 {
		t.Errorf("level 30: Prayer of Healing has %d ranks, want 1", len(registered(agent.PrayerOfHealing)))
	}
}

// TestAtLevelSixtyTheTopRanksAreTheClientsLevelSixtyRanks pins which rank
// ids a level-60 priest ends up casting.
func TestAtLevelSixtyTheTopRanksAreTheClientsLevelSixtyRanks(t *testing.T) {
	_, agent := healerSim(t, 60, HolyTalents)
	tops := map[string]struct {
		ladder []*core.Spell
		id     int32
		ranks  int
	}{
		"Flash Heal":         {agent.FlashHeal, 10917, 7},
		"Greater Heal":       {agent.GreaterHeal, 25314, 5},
		"Renew":              {agent.Renew, 25315, 10},
		"Prayer of Healing":  {agent.PrayerOfHealing, 25316, 5},
		"Power Word: Shield": {agent.PowerWordShield, 10901, 10},
		"Binding Heal":       {agent.BindingHeal, 1240774, 6},
		"Prayer of Mending":  {agent.PrayerOfMending, 1240827, 3},
		"Inner Fire":         {agent.InnerFire, 10952, 6},
	}
	for name, top := range tops {
		got := registered(top.ladder)
		if len(got) != top.ranks {
			t.Errorf("%s: %d ranks at level 60, want %d", name, len(got), top.ranks)
			continue
		}
		if id := got[top.ranks].SpellID; id != top.id {
			t.Errorf("%s top rank is spell %d, want %d", name, id, top.id)
		}
	}
}

// TestSpellsGatedByATalentExistOnlyWithIt covers Binding Heal, Penance,
// Holy Nova and Prayer of Mending.
func TestSpellsGatedByATalentExistOnlyWithIt(t *testing.T) {
	_, bare := healerSim(t, 60, "")
	for name, ladder := range map[string][]*core.Spell{
		"Binding Heal": bare.BindingHeal, "Penance": bare.Penance,
		"Holy Nova": bare.HolyNova, "Prayer of Mending": bare.PrayerOfMending,
	} {
		if len(registered(ladder)) != 0 {
			t.Errorf("%s registers without its talent", name)
		}
	}
	_, holy := healerSim(t, 60, HolyTalents)
	if len(registered(holy.BindingHeal)) == 0 || len(registered(holy.PrayerOfMending)) == 0 {
		t.Error("the Holy build is missing Binding Heal or Prayer of Mending")
	}
	_, disc := healerSim(t, 60, DiscTalents)
	if len(registered(disc.Penance)) != 4 {
		t.Errorf("the Discipline build has %d Penance ranks, want 4", len(registered(disc.Penance)))
	}
}

// TestRenewTicksFiveTimesEveryThreeSeconds runs a Renew out and checks the
// tick count, the period and the total against the client's per-tick
// amount plus the coefficient per tick.
func TestRenewTicksFiveTimesEveryThreeSeconds(t *testing.T) {
	sim, agent := healerSim(t, 60, "")
	target := agent.GetMainTarget()
	renew := agent.Renew[10]
	row, _ := loadClient(t).ByID(renew.SpellID)
	perTick := row.Effects[0].Amount + row.Effects[0].ResolvedSPCoefficient*renew.HealingPower(target)

	metrics := func() core.SpellMetrics { return renew.SpellMetrics[target.UnitIndex] }
	start := sim.CurrentTime
	renew.ApplyEffects(sim, target, renew)
	ticksAt := []time.Duration{}
	lastHits := metrics().Hits
	for sim.CurrentTime < start+16*time.Second {
		if sim.Step() {
			break
		}
		if hits := metrics().Hits; hits != lastHits {
			ticksAt = append(ticksAt, sim.CurrentTime-start)
			lastHits = hits
		}
	}
	if len(ticksAt) != 5 {
		t.Fatalf("Renew ticked %d times, want 5 (at %v)", len(ticksAt), ticksAt)
	}
	for i, at := range ticksAt {
		if want := time.Duration(i+1) * 3 * time.Second; at != want {
			t.Errorf("tick %d at %v, want %v", i+1, at, want)
		}
	}
	assertWithin(t, "Renew total", metrics().TotalHealing, 5*perTick, 5*perTick)
}

// TestPowerWordShieldAbsorbsAndBlocksAnotherForWeakenedSoul covers the
// absorb amount, Weakened Soul on the target, the cooldown and that the
// shield soaks raid damage and counts as the shield's healing.
func TestPowerWordShieldAbsorbsAndBlocksAnotherForWeakenedSoul(t *testing.T) {
	sim, agent := healerSim(t, 60, "")
	target := agent.GetMainTarget()
	shield := agent.PowerWordShield[10]
	row, _ := loadClient(t).ByID(shield.SpellID)
	want := row.Effects[0].Amount + row.Effects[0].ResolvedSPCoefficient*shield.HealingPower(target)

	if !shield.CanCast(sim, target) {
		t.Fatal("a fresh target cannot be shielded")
	}
	shield.ApplyEffects(sim, target, shield)
	if got := shield.Shield(target).Remaining(); math.Abs(got-want) > 0.01 {
		t.Errorf("shield absorbs %.2f, want %.2f", got, want)
	}
	if shield.CanCast(sim, target) {
		t.Error("a target with Weakened Soul can be shielded again")
	}
	soaked := 300.0
	if left := target.AbsorbDamage(sim, soaked); left != 0 {
		t.Errorf("a %v hit passed %v through a %.0f shield", soaked, left, want)
	}
	if got := shield.SpellMetrics[target.UnitIndex].TotalEffectiveHealing; got != soaked {
		t.Errorf("shield's effective healing %v, want the %v it soaked", got, soaked)
	}
}

// TestPrayerOfHealingHealsTheTargetsWholeParty counts a heal per member.
func TestPrayerOfHealingHealsTheTargetsWholeParty(t *testing.T) {
	sim, agent := healerSim(t, 60, "")
	member := &sim.Raid.Parties[0].Players[1].GetCharacter().Unit
	spell := agent.PrayerOfHealing[5]
	spell.BonusCritRating += noCrit
	spell.ApplyEffects(sim, member, spell)
	healed := 0
	for _, unit := range sim.Raid.AllPlayerUnits {
		if spell.SpellMetrics[unit.UnitIndex].Hits > 0 {
			healed++
		}
	}
	if healed != 5 {
		t.Errorf("Prayer of Healing reached %d units, want the healer and its four party members", healed)
	}
	if hits := spell.SpellMetrics[agent.GetMainTarget().UnitIndex].Hits; hits != 0 {
		t.Errorf("Prayer of Healing reached the tank, who stands in another party")
	}
}

// TestBindingHealHealsTargetAndCaster checks both halves land.
func TestBindingHealHealsTargetAndCaster(t *testing.T) {
	sim, agent := healerSim(t, 60, HolyTalents)
	target := agent.GetMainTarget()
	spell := agent.BindingHeal[6]
	spell.BonusCritRating += noCrit
	spell.ApplyEffects(sim, target, spell)
	if spell.SpellMetrics[target.UnitIndex].Hits != 1 || spell.SpellMetrics[agent.Unit.UnitIndex].Hits != 1 {
		t.Errorf("Binding Heal hit the target %d times and the caster %d times, want once each",
			spell.SpellMetrics[target.UnitIndex].Hits, spell.SpellMetrics[agent.Unit.UnitIndex].Hits)
	}
}

// TestDesperatePrayerHealsOnlyThePriest names a friend and checks the
// priest is healed instead.
func TestDesperatePrayerHealsOnlyThePriest(t *testing.T) {
	sim, agent := healerSim(t, 60, "")
	friend := agent.GetMainTarget()
	spell := agent.DesperatePrayer[7]
	spell.ApplyEffects(sim, friend, spell)
	if spell.SpellMetrics[agent.Unit.UnitIndex].Hits+spell.SpellMetrics[agent.Unit.UnitIndex].Crits != 1 || spell.SpellMetrics[friend.UnitIndex].Hits != 0 {
		t.Error("Desperate Prayer did not heal exactly the priest")
	}
}

// TestPenanceHealsOnThreePulsesOverTwoSeconds runs the channel out.
func TestPenanceHealsOnThreePulsesOverTwoSeconds(t *testing.T) {
	sim, agent := healerSim(t, 60, penanceOnly)
	target := agent.GetMainTarget()
	penance := agent.Penance[4]
	row, _ := loadClient(t).ByID(1316991) // the volley's heal helper
	pulse := row.Effects[0].Amount + row.Effects[0].ResolvedSPCoefficient*penance.HealingPower(target)

	penance.BonusCritRating += noCrit
	start := sim.CurrentTime
	penance.ApplyEffects(sim, target, penance)
	pulses := []time.Duration{0}
	last := penance.SpellMetrics[target.UnitIndex].Hits
	for sim.CurrentTime < start+3*time.Second {
		if sim.Step() {
			break
		}
		if hits := penance.SpellMetrics[target.UnitIndex].Hits; hits != last {
			pulses = append(pulses, sim.CurrentTime-start)
			last = hits
		}
	}
	if len(pulses) != 3 || pulses[1] != time.Second || pulses[2] != 2*time.Second {
		t.Fatalf("Penance pulsed at %v, want the cast, 1 s and 2 s", pulses)
	}
	assertWithin(t, "Penance total", penance.SpellMetrics[target.UnitIndex].TotalHealing, 3*pulse, 3*pulse)
}

// TestHolyNovaHealsThePartyAndHurtsEnemies checks both halves.
func TestHolyNovaHealsThePartyAndHurtsEnemies(t *testing.T) {
	sim, agent := healerSim(t, 60, holyNovaOnly)
	spell := agent.HolyNova[6]
	spell.ApplyEffects(sim, &agent.Unit, spell)
	enemy := sim.Encounter.TargetUnits[0]
	if spell.SpellMetrics[enemy.UnitIndex].TotalDamage <= 0 {
		t.Error("Holy Nova did no damage to the enemy")
	}
	healed := 0
	for _, unit := range sim.Raid.AllPlayerUnits {
		if spell.SpellMetrics[unit.UnitIndex].TotalHealing > 0 {
			healed++
		}
	}
	if healed != 5 {
		t.Errorf("Holy Nova healed %d units, want the healer's party of five", healed)
	}
}

// TestPrayerOfMendingSpendsAChargeOnTheNextHealAndJumps checks the heal
// the holder gets, the charge count and the jump to the most hurt member.
func TestPrayerOfMendingSpendsAChargeOnTheNextHealAndJumps(t *testing.T) {
	sim, agent := healerSim(t, 60, HolyTalents)
	tank := agent.GetMainTarget()
	hurt := &sim.Raid.Parties[0].Players[2].GetCharacter().Unit
	hurt.RemoveHealth(sim, 3000)

	mending := agent.PrayerOfMending[3]
	mending.ApplyEffects(sim, tank, mending)
	if mending.SpellMetrics[tank.UnitIndex].Hits != 0 {
		t.Fatal("Prayer of Mending healed on the cast")
	}
	healedBy(sim, agent.FlashHeal[7], tank)
	if got := mending.SpellMetrics[tank.UnitIndex].Hits + mending.SpellMetrics[tank.UnitIndex].Crits; got != 1 {
		t.Fatalf("the tank's next heal spent %d charges, want 1", got)
	}
	healedBy(sim, agent.FlashHeal[7], hurt)
	if got := mending.SpellMetrics[hurt.UnitIndex].Hits + mending.SpellMetrics[hurt.UnitIndex].Crits; got != 1 {
		t.Errorf("the charges did not jump to the most hurt member: %d heals there", got)
	}
}

// TestInnerFireAddsArmorByRank checks the armor and its talent.
func TestInnerFireAddsArmorByRank(t *testing.T) {
	sim, agent := healerSim(t, 60, "")
	before := agent.GetStat(stats.Armor)
	spell := agent.InnerFire[6]
	spell.ApplyEffects(sim, &agent.Unit, spell)
	if got := agent.GetStat(stats.Armor) - before; got != 1395 {
		t.Errorf("Inner Fire rank 6 adds %v armor, want the client's 1395", got)
	}
}

var _ = priest.SpellCode_PriestNone
