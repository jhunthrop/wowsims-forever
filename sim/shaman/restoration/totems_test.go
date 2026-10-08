package restoration

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/healsim"
	"github.com/wowsims/classic/sim/shaman"
)

// Top ranks of the water totems at level 60, from the client's spell
// constants (spells 10461, 10494 and 17360).
const (
	healingStreamTopRankID   = 10463
	healingStreamTopHealID   = 10461
	healingStreamTopRankHeal = 11
	healingStreamCoefficient = 0.022
	manaSpringTopRankID      = 10497
	manaSpringTopRankRestore = 10
	manaSpringMP5PerRestore  = 2.5
	manaTideTopRankID        = 17359
	manaTideTopRankPerTick   = 290
	manaTideTicks            = 4
	healingStreamFightLength = 20
	healingStreamTickGap     = 2
)

// castOnce is an APL that casts one spell, by id and rank, in the first
// second and nothing else.
func castOnce(spellID, rank int) *proto.APLRotation {
	return core.APLRotationFromJsonString(fmt.Sprintf(`{
		"type": "TypeAPL",
		"priorityList": [{"action": {"condition": {"cmp": {"op": "OpLt",
			"lhs": {"currentTime": {}}, "rhs": {"const": {"val": "1s"}}}},
			"castSpell": {"spellId": {"spellId": %d, "rank": %d}}}}]
	}`, spellID, rank))
}

// runHealer runs the player for seconds against a quiet raid and returns the
// healer's metrics.
func runHealer(t *testing.T, player *proto.Player, seconds float64) *proto.UnitMetrics {
	t.Helper()

	result := core.RunRaidSim(healsim.Request(player, quietRaid(), seconds, 1))
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}
	return result.RaidMetrics.Parties[0].Players[healsim.HealerIndex]
}

// manaGained is the mana the healer got from a spell id over the run.
func manaGained(metrics *proto.UnitMetrics, spellID int32) float64 {
	var total float64
	for _, resource := range metrics.Resources {
		if resource.Type == proto.ResourceType_ResourceTypeMana && resource.Id.GetSpellId() == spellID && resource.Gain > 0 {
			total += resource.Gain
		}
	}
	return total
}

// healthGained returns, for the heal with spellID on members of the healer's
// party, the events and the raw health they got.
func healthGained(result *proto.RaidSimResult, spellID int32) (events int32, gain float64) {
	for _, player := range result.RaidMetrics.Parties[0].Players[1:] {
		for _, resource := range player.Resources {
			if resource.Type == proto.ResourceType_ResourceTypeHealth && resource.Id.GetSpellId() == spellID {
				events += resource.Events
				gain += resource.Gain
			}
		}
	}
	return events, gain
}

func runHealingStream(t *testing.T, talents string) *proto.RaidSimResult {
	t.Helper()

	player := newHealerPlayer(60, talents, castOnce(healingStreamTopRankID, 5))
	result := core.RunRaidSim(healsim.Request(player, quietRaid(), healingStreamFightLength, 1))
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}
	return result
}

func TestHealingStreamHealsEveryPartyMemberEveryTwoSeconds(t *testing.T) {
	result := runHealingStream(t, "")

	events, gain := healthGained(result, healingStreamTopHealID)
	// Four fake party members; the tank stands in the other party.
	wantTicks := int32(4 * healingStreamFightLength / healingStreamTickGap)
	if events < wantTicks-4 || events > wantTicks {
		t.Errorf("the totem ticked %d times on the party, want about %d", events, wantTicks)
	}
	wantTick := healingStreamTopRankHeal + healingStreamCoefficient*testHealingPower
	if got := gain / float64(events); math.Abs(got-wantTick) > 1e-6 {
		t.Errorf("a tick heals %.4f, want %.4f (client 11 plus 0.022 of healing power)", got, wantTick)
	}
}

func TestRestorativeTotemsAndPurificationRaiseHealingStream(t *testing.T) {
	talents := restoTalents(map[int]int{posRestorativeTotems: 5, posPurification: 5})
	result := runHealingStream(t, talents)

	events, gain := healthGained(result, healingStreamTopHealID)
	// Restorative Totems adds 10 percent a rank to the totem, Purification
	// 2 percent a rank to every heal, and the two add: +60 percent.
	wantTick := (healingStreamTopRankHeal + healingStreamCoefficient*testHealingPower) * 1.6
	if got := gain / float64(events); math.Abs(got-wantTick) > 1e-6 {
		t.Errorf("a tick heals %.4f, want %.4f", got, wantTick)
	}
}

func TestManaSpringRestoresManaAndRestorativeTotemsRaiseIt(t *testing.T) {
	for name, tc := range map[string]struct {
		talents string
		want    float64
	}{
		"plain":                {"", manaSpringTopRankRestore * manaSpringMP5PerRestore},
		"Restorative Totems 5": {restoTalents(map[int]int{posRestorativeTotems: 5}), manaSpringTopRankRestore * manaSpringMP5PerRestore * 1.25},
	} {
		sim, healer := newHealer(t, 60, tc.talents)

		before := healer.GetStat(stats.MP5)
		healer.ManaSpringTotem[4].Cast(sim, &healer.Unit)
		if got := healer.GetStat(stats.MP5) - before; math.Abs(got-tc.want) > 1e-6 {
			t.Errorf("%s: Mana Spring added %.3f mp5, want %.3f", name, got, tc.want)
		}
	}
}

func TestTotemicFocusCutsTotemCosts(t *testing.T) {
	_, healer := newHealer(t, 60, restoTalents(map[int]int{posTotemicFocus: 5, posManaTideTotem: 1}))

	if got, want := healer.HealingStreamTotem[5].Cost.GetCurrentCost(), 80*0.75; math.Abs(got-want) > 1e-6 {
		t.Errorf("Healing Stream Totem costs %v, want %v", got, want)
	}
	if got, want := healer.ManaTideTotem[3].Cost.GetCurrentCost(), 60*0.75; math.Abs(got-want) > 1e-6 {
		t.Errorf("Mana Tide Totem costs %v, want %v", got, want)
	}
}

func TestOnlyOneWaterTotemStands(t *testing.T) {
	sim, healer := newHealer(t, 60, restoTalents(map[int]int{posManaTideTotem: 1}))
	springAura := healer.GetAura("Mana Spring Totem (Rank 4)")
	stream := healer.HealingStreamTotem[5]
	tide := healer.ManaTideTotem[3]
	partyMember := raidMember(sim, 1)

	healer.ManaSpringTotem[4].Cast(sim, &healer.Unit)
	if !springAura.IsActive() {
		t.Fatal("Mana Spring should be up after it is cast")
	}

	sim.CurrentTime += 2 * time.Second // past the totem global cooldown
	stream.Cast(sim, &healer.Unit)
	if springAura.IsActive() {
		t.Error("dropping Healing Stream should have ended Mana Spring: both are water totems")
	}
	if !stream.Hot(partyMember).IsActive() {
		t.Error("Healing Stream should be healing the party")
	}

	sim.CurrentTime += 2 * time.Second
	tide.Cast(sim, &healer.Unit)
	if stream.Hot(partyMember).IsActive() {
		t.Error("Mana Tide Totem should have replaced Healing Stream")
	}
	if got := healer.TotemExpirations[shaman.WaterTotem] - sim.CurrentTime; got != 13*time.Second {
		t.Errorf("the water slot should be held for the tide's 13s, got %v", got)
	}
}

func TestManaTideTotemOnlyExistsWithTheTalent(t *testing.T) {
	_, without := newHealer(t, 60, "")
	if len(without.ManaTideTotem) != 0 {
		t.Error("Mana Tide Totem registered without its talent")
	}
}

func TestManaTideTotemRestoresTheClientAmountFourTimes(t *testing.T) {
	player := newHealerPlayer(60, restoTalents(map[int]int{posManaTideTotem: 1}), castOnce(manaTideTopRankID, 3))
	metrics := runHealer(t, player, 20)

	want := float64(manaTideTicks * manaTideTopRankPerTick)
	if got := manaGained(metrics, manaTideTopRankID); got != want {
		t.Errorf("Mana Tide Totem restored %v mana, want %v (4 ticks of 290)", got, want)
	}
}
