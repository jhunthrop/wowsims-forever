package restoration

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/healsim"
)

const (
	riptideRankThreeTick = 161 // spell 1239243's heal-over-time tick at level 60
	riptideTickCoeff     = 0.1
	riptideTicks         = 5
	riptideTickGap       = 3 * time.Second
)

func TestRiptideRanksOnlyExistWithTheTalent(t *testing.T) {
	_, without := newHealer(t, 60, "")
	if len(without.Riptide) != 0 {
		t.Errorf("a shaman without the talent registered %d Riptide slots", len(without.Riptide))
	}

	_, with := newHealer(t, 60, riptideOnly)
	if with.Riptide[3] == nil {
		t.Fatal("a level-60 shaman with the talent should have Riptide rank 3")
	}
}

// TestRiptideOnlyStartsItsHealOverTime: since client 1.60.1.70755 every
// rank's effect 0 is a dummy (Effect 3, 0 base points), so the cast heals
// nothing at once and only leaves its heal over time.
func TestRiptideOnlyStartsItsHealOverTime(t *testing.T) {
	sim, healer := newHealer(t, 60, riptideOnly)
	tank := raidMember(sim, healsim.TankIndex)
	spell := healer.Riptide[3]

	if healing, _ := healOf(sim, spell, tank); healing != 0 {
		t.Errorf("Riptide healed %.2f on the cast, want 0 (no direct heal since 1.60.1.70755)", healing)
	}

	hot := spell.Hot(tank)
	if !hot.IsActive() {
		t.Fatal("Riptide should leave its heal over time on the target")
	}
	if hot.NumberOfTicks != riptideTicks || hot.TickLength != riptideTickGap {
		t.Errorf("heal over time = %d ticks of %v, want %d of %v", hot.NumberOfTicks, hot.TickLength, riptideTicks, riptideTickGap)
	}
	wantTick := riptideRankThreeTick + riptideTickCoeff*testHealingPower
	if hot.SnapshotBaseDamage != wantTick {
		t.Errorf("a tick heals %.2f, want %.2f (client 161 plus 0.1 of %d healing power)", hot.SnapshotBaseDamage, wantTick, testHealingPower)
	}
}

func TestRiptideRanksShareOneHealOverTimePerTarget(t *testing.T) {
	sim, healer := newHealer(t, 60, riptideOnly)
	tank := raidMember(sim, healsim.TankIndex)

	healOf(sim, healer.Riptide[1], tank)
	healOf(sim, healer.Riptide[3], tank)

	if healer.Riptide[1].Hot(tank).IsActive() {
		t.Error("rank 1's heal over time should have been replaced by rank 3's")
	}
	if !healer.Riptide[3].Hot(tank).IsActive() {
		t.Error("rank 3's heal over time should be up")
	}
}

func TestRiptideRanksShareOneCooldown(t *testing.T) {
	sim, healer := newHealer(t, 60, riptideOnly)
	tank := raidMember(sim, healsim.TankIndex)

	healer.Riptide[3].Cast(sim, tank)

	if healer.Riptide[1].CD.IsReady(sim) {
		t.Error("casting rank 3 should put rank 1 on cooldown too")
	}
	// Client 1.60.1.70755: CategoryRecoveryTime 12000 (6000 before).
	if got := healer.Riptide[3].CD.Duration; got != 12*time.Second {
		t.Errorf("Riptide cooldown = %v, want 12s", got)
	}
}

func TestChainHealOnARiptidedTargetHeals25PercentMore(t *testing.T) {
	sim, healer := newHealer(t, 60, riptideOnly)
	tank := raidMember(sim, healsim.TankIndex)
	chain := healer.ChainHeal[3]

	plain, plainCrit := healOf(sim, chain, tank)
	assertRoll(t, "Chain Heal without Riptide", chain, plain, plainCrit, 1)

	healOf(sim, healer.Riptide[3], tank)
	boosted, boostedCrit := healOf(sim, chain, tank)
	assertRoll(t, "Chain Heal with Riptide", chain, boosted, boostedCrit, 1.25)
}

func TestRiptideBoostsOnlyTheChainHealCastDirectlyOnItsTarget(t *testing.T) {
	sim, healer := newHealer(t, 60, riptideOnly)
	tank := raidMember(sim, healsim.TankIndex)
	member := raidMember(sim, 2)
	chain := healer.ChainHeal[3]

	// Riptide on a member the chain will jump to, then a Chain Heal on the
	// tank: the jump is the plain 50 percent of the roll.
	member.RemoveHealth(sim, 3000)
	healOf(sim, healer.Riptide[3], member)
	before := chain.SpellMetrics[member.UnitIndex].TotalHealing
	chain.ApplyEffects(sim, tank, chain)

	jump := chain.SpellMetrics[member.UnitIndex].TotalHealing - before
	crit := chain.SpellMetrics[member.UnitIndex].Crits > 0
	assertRoll(t, "Chain Heal jump onto a Riptided member", chain, jump, crit, 0.5)
}

// onlyRiptideOnce casts Riptide on the tank in the first second and nothing
// else, so the run shows one cast and everything it leaves behind.
func onlyRiptideOnce() *proto.APLRotation {
	return core.APLRotationFromJsonString(`{
		"type": "TypeAPL",
		"priorityList": [{"action": {"condition": {"cmp": {"op": "OpLt",
			"lhs": {"currentTime": {}}, "rhs": {"const": {"val": "1s"}}}},
			"castSpell": {"spellId": {"spellId": 1239243, "rank": 3},
			"target": {"type": "Player", "index": 5}}}}]
	}`)
}

func TestRiptideTicksFiveTimesOverFifteenSeconds(t *testing.T) {
	player := newHealerPlayer(60, riptideOnly, onlyRiptideOnce())
	result := core.RunRaidSim(healsim.Request(player, quietRaid(), 16, 1))
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}

	for _, action := range result.RaidMetrics.Parties[0].Players[healsim.HealerIndex].Actions {
		if action.Id.GetSpellId() != 1239243 {
			continue
		}
		for _, target := range action.Targets {
			if target.Casts == 0 {
				continue
			}
			// No direct heal (gone since 1.60.1.70755) and five ticks
			// (plain or critical: the heal over time crits since
			// 1.60.1.70707).
			direct, ticks := target.Hits+target.Crits, target.Ticks+target.CritTicks
			if target.Casts != 1 || direct != 0 || ticks != riptideTicks {
				t.Errorf("Riptide cast %d time(s), landed %d direct heals and %d ticks, want 1, 0 and %d", target.Casts, direct, ticks, riptideTicks)
			}
			return
		}
	}
	t.Fatal("the run never cast Riptide")
}

// Riptide's heal over time crits: every rank carries the client's
// PERIODIC_CAN_CRIT flag (1.60.1.70707), so its ticks roll the healing crit
// chance through the dot's own outcome.
func TestRiptideTicksCanCrit(t *testing.T) {
	sim, healer := newHealer(t, 60, riptideOnly)
	tank := raidMember(sim, healsim.TankIndex)
	spell := healer.Riptide[3]
	hot := spell.Hot(tank)
	if !hot.CanCrit {
		t.Fatal("Riptide's heal over time is not crit-capable")
	}
	hot.Apply(sim)
	for i := 0; i < 400; i++ {
		hot.TickOnce(sim)
	}
	if spell.SpellMetrics[tank.UnitIndex].CritTicks == 0 {
		t.Errorf("400 Riptide ticks at %.1f%% healing crit never crit", 100*spell.HealingCritChance())
	}
}
