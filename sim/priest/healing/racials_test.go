package healing

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// The two Forever priest racial heals, read from the 1.60.1.70707 client:
//
//	Divine Grace (human; SkillLineAbility skill line 56, RaceMasks 1):
//	  ranks 1277370/71/72/74/76/77/78 at levels 10/18/26/34/42/50/58.
//	  Rank 7 1277378: effect 0 heal 1383, +8.2 a level to 64, variance
//	  0.164934, coefficient 0.429; effect 1 states the 50% health gate.
//	  No mana cost, 1.5 s GCD, 10 minute category cooldown. "Instantly
//	  heals a friendly target below $s2% Health for $s1 and removes
//	  Weakened Soul from that target. Cannot be cast on self."
//	Contingency Plan (gnome; skill line 613, RaceMasks 64): wards
//	  1277462/1277634/1277638/1277639/1277640 at 20/30/40/50/60, 30 s,
//	  no mana cost, 1.5 s GCD, 10 minute category cooldown. "The next time
//	  this ally takes damage dropping their Health below $s1% (35), they
//	  will gain a shield absorbing $1277466s2 (926 at rank 5) damage and
//	  begin healing for $1277460o2 Health over $1277460d": 1277460 heals
//	  134 every 3 s for 15 s at coefficient 0.2 a tick.

func racialPriest(t *testing.T, race proto.Race, level int32) (*core.Simulation, *HealingPriest) {
	t.Helper()
	player := healer(level, "", &proto.HealingPriest_Options{}, &proto.APLRotation{})
	player.Race = race
	return agentSim(t, player)
}

func TestDivineGraceIsTheHumanPriestsAndFollowsTheLevel(t *testing.T) {
	for _, c := range []struct {
		race  proto.Race
		level int32
		id    int32
	}{
		{proto.Race_RaceHuman, 60, 1277378},
		{proto.Race_RaceHuman, 50, 1277377},
		{proto.Race_RaceHuman, 20, 1277371},
		{proto.Race_RaceHuman, 9, 0},
		{proto.Race_RaceGnome, 60, 0},
	} {
		_, agent := racialPriest(t, c.race, c.level)
		got := int32(0)
		if agent.DivineGrace != nil {
			got = agent.DivineGrace.SpellID
		}
		if got != c.id {
			t.Errorf("%v priest at %d: Divine Grace %d, want %d", c.race, c.level, got, c.id)
		}
	}
}

func TestDivineGraceHealsAHurtAllyForTheClientRoll(t *testing.T) {
	sim, agent := racialPriest(t, proto.Race_RaceHuman, 60)
	tank := agent.GetMainTarget()
	spell := agent.DivineGrace
	if spell.CanCast(sim, tank) {
		t.Fatal("Divine Grace can be cast on a target above 50% health")
	}
	tank.RemoveHealth(sim, tank.MaxHealth()*0.7)
	if !spell.CanCast(sim, tank) {
		t.Fatal("Divine Grace cannot be cast on a target below 50% health")
	}
	// Cannot be cast on self: the autocast's self-target goes to the tank.
	priestRaw := spell.SpellMetrics[agent.UnitIndex].TotalHealing
	centre := 1383 + 8.2*2
	low, high := centre*(1-0.164934/2), centre*(1+0.164934/2)
	bonus := 0.429 * spell.HealingPower(tank)
	healed := healedBy(sim, spell, tank)
	if healed > (high+bonus)*1.5+0.01 || healed < low+bonus-0.01 {
		t.Errorf("Divine Grace healed %.1f, want %.1f-%.1f before a crit", healed, low+bonus, high+bonus)
	}
	if got := spell.SpellMetrics[agent.UnitIndex].TotalHealing; got != priestRaw {
		t.Errorf("Divine Grace healed the priest for %.1f", got-priestRaw)
	}
	if spell.CD.Duration != 10*time.Minute {
		t.Errorf("Divine Grace cooldown %v, want 10m", spell.CD.Duration)
	}
}

func TestContingencyPlanIsTheGnomePriestsAndFollowsTheLevel(t *testing.T) {
	for _, c := range []struct {
		race  proto.Race
		level int32
		id    int32
	}{
		{proto.Race_RaceGnome, 60, 1277640},
		{proto.Race_RaceGnome, 40, 1277638},
		{proto.Race_RaceGnome, 19, 0},
		{proto.Race_RaceHuman, 60, 0},
	} {
		_, agent := racialPriest(t, c.race, c.level)
		got := int32(0)
		if agent.ContingencyPlan != nil {
			got = agent.ContingencyPlan.SpellID
		}
		if got != c.id {
			t.Errorf("%v priest at %d: Contingency Plan %d, want %d", c.race, c.level, got, c.id)
		}
	}
}

// The ward waits for the hit that drops its holder below 35%, then gives
// a 926 absorb and five 134 + 0.2 x healing power ticks.
func TestContingencyPlanShieldsAndHealsWhenItsHolderFallsBelowThirtyFivePercent(t *testing.T) {
	sim, agent := racialPriest(t, proto.Race_RaceGnome, 60)
	tank := agent.GetMainTarget()
	ward := agent.ContingencyPlan
	ward.ApplyEffects(sim, tank, ward)

	core.DealRaidDamage(sim, tank, tank.MaxHealth()*0.5)
	if agent.ContingencyShield.Shield(tank).IsActive() {
		t.Fatal("the ward fired above 35% health")
	}
	core.DealRaidDamage(sim, tank, tank.MaxHealth()*0.2)
	shield := agent.ContingencyShield.Shield(tank)
	if !shield.IsActive() || math.Abs(shield.Remaining()-926) > 0.01 {
		t.Fatalf("after the ward fired the shield holds %.1f (active %v), want 926", shield.Remaining(), shield.IsActive())
	}
	hot := agent.ContingencyHot.Hot(tank)
	if !hot.IsActive() || hot.NumberOfTicks != 5 || hot.TickLength != 3*time.Second {
		t.Fatalf("the ward's heal over time is %v ticks of %v (active %v), want 5 of 3s", hot.NumberOfTicks, hot.TickLength, hot.IsActive())
	}
	if want := 134 + 0.2*agent.ContingencyHot.HealingPower(tank); math.Abs(hot.SnapshotBaseDamage-want) > 0.01 {
		t.Errorf("a tick heals %.1f, want %.1f", hot.SnapshotBaseDamage, want)
	}
	// One ward, one firing.
	core.DealRaidDamage(sim, tank, 1)
	if agent.ContingencyWards.Get(tank).IsActive() {
		t.Error("the ward is still up after it fired")
	}
}

// Desperate Prayer is the dwarf priest's: its learn rows (13908, 19243,
// skill line 56) carry RaceMasks 4.
func TestDesperatePrayerIsTheDwarfPriests(t *testing.T) {
	for race, want := range map[proto.Race]bool{proto.Race_RaceDwarf: true, proto.Race_RaceHuman: false, proto.Race_RaceUndead: false} {
		_, agent := racialPriest(t, race, 60)
		if got := len(registered(agent.DesperatePrayer)) > 0; got != want {
			t.Errorf("a %v priest has Desperate Prayer = %v, want %v", race, got, want)
		}
	}
}
