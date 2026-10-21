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

func TestRiptideHealsWithinTheClientRollAndStartsItsHeal(t *testing.T) {
	sim, healer := newHealer(t, 60, riptideOnly)
	tank := raidMember(sim, healsim.TankIndex)
	spell := healer.Riptide[3]

	healing, crit := healOf(sim, spell, tank)
	assertRoll(t, "Riptide direct heal", spell, healing, crit, 1)

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
	if got := healer.Riptide[3].CD.Duration; got != 6*time.Second {
		t.Errorf("Riptide cooldown = %v, want 6s", got)
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
			// One direct heal and one hit per tick.
			if target.Casts != 1 || target.Hits != 1+riptideTicks {
				t.Errorf("Riptide cast %d time(s) and landed %d heals, want 1 and %d", target.Casts, target.Hits, 1+riptideTicks)
			}
			return
		}
	}
	t.Fatal("the run never cast Riptide")
}
