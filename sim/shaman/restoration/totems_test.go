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
	manaTideTopRankID        = 1325256 // rank 4 since 1.60.1.70425
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
	return runHealingStreamWithCrit(t, talents, 0)
}

// runHealingStreamWithCrit runs the totem with extra crit rating on the
// shaman; a large negative amount takes the heal's crit chance to zero, so
// a tick's size can be read exactly.
func runHealingStreamWithCrit(t *testing.T, talents string, critRating float64) *proto.RaidSimResult {
	t.Helper()

	player := newHealerPlayer(60, talents, castOnce(healingStreamTopRankID, 5))
	bonus := stats.FromFloatArray(player.BonusStats.Stats)
	bonus[stats.Crit] += critRating
	player.BonusStats = &proto.UnitStats{Stats: bonus.ToFloatArray()}
	result := core.RunRaidSim(healsim.Request(player, quietRaid(), healingStreamFightLength, 1))
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}
	return result
}

// noCrit is enough negative crit rating to take any crit chance to zero.
const noCrit = -100 * core.CritRatingPerCritChance

func TestHealingStreamHealsEveryPartyMemberEveryTwoSeconds(t *testing.T) {
	result := runHealingStreamWithCrit(t, "", noCrit)

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

// Healing Stream's ticks can crit: every rank's heal (5672 to 10461)
// carries SPELL_ATTR8_PERIODIC_CAN_CRIT (SpellMisc Attributes_8 0x200) in
// the 1.60.1.70707 tables, the rule every other heal over time follows.
// No second source states it either way, so the client's table decides.
func TestHealingStreamTicksCanCrit(t *testing.T) {
	result := runHealingStreamWithCrit(t, "", 100*core.CritRatingPerCritChance)

	events, gain := healthGained(result, healingStreamTopHealID)
	plain := healingStreamTopRankHeal + healingStreamCoefficient*testHealingPower
	if got := gain / float64(events); got < plain*1.4 {
		t.Errorf("at 100%% crit a tick heals %.4f on average, want a crit's %.4f or more", got, plain*1.5)
	}
}

// The totem's ticks crit only on the shaman's every-table crit (item crit
// and aura-290 buffs such as Thundering Strikes), not its Intellect spell
// crit: in the beta logs (foreverlogs.gg, levels 20 to 35, read 2026-11-02)
// five Restoration and Elemental shamans with no Enhancement points show 0
// crits in 21,134 Healing Stream ticks while their own heals crit 4 to 6%,
// and the Enhancement shamans, with Thundering Strikes, crit about 5%.
func TestHealingStreamIgnoresTheShamansSpellTableCrit(t *testing.T) {
	player := newHealerPlayer(60, "", castOnce(healingStreamTopRankID, 5))
	bonus := stats.FromFloatArray(player.BonusStats.Stats)
	bonus[stats.Crit] = 0
	bonus[stats.SpellCrit] += 100 * core.CritRatingPerCritChance
	player.BonusStats = &proto.UnitStats{Stats: bonus.ToFloatArray()}
	result := core.RunRaidSim(healsim.Request(player, quietRaid(), healingStreamFightLength, 1))
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}
	events, gain := healthGained(result, healingStreamTopHealID)
	plain := healingStreamTopRankHeal + healingStreamCoefficient*testHealingPower
	if got := gain / float64(events); got > plain*1.01 {
		t.Errorf("with 100%% spell-table crit and no every-table crit a tick heals %.4f on average, want %.4f (no crits)", got, plain)
	}
}

// A Healing Stream crit is a periodic heal's crit, and Water Shield's proc
// mask (SpellAuraOptions 223904, ProcTypeMask 0x362A8) has no periodic bit,
// so the totem's crits restore no mana.
func TestHealingStreamCritsDoNotTriggerWaterShield(t *testing.T) {
	rotation := core.APLRotationFromJsonString(fmt.Sprintf(`{
		"type": "TypeAPL",
		"priorityList": [
			{"action": {"condition": {"not": {"val": {"auraIsActive": {"auraId": {"spellId": 408510}}}}},
				"castSpell": {"spellId": {"spellId": 408510}}}},
			{"action": {"condition": {"cmp": {"op": "OpLt",
				"lhs": {"currentTime": {}}, "rhs": {"const": {"val": "3s"}}}},
				"castSpell": {"spellId": {"spellId": %d, "rank": 5}}}}
		]
	}`, healingStreamTopRankID))
	player := newHealerPlayer(60, FullTalents, rotation)
	bonus := stats.FromFloatArray(player.BonusStats.Stats)
	bonus[stats.Crit] += 100 * core.CritRatingPerCritChance
	player.BonusStats = &proto.UnitStats{Stats: bonus.ToFloatArray()}
	metrics := runHealer(t, player, healingStreamFightLength)
	if got := manaGained(metrics, 408511); got != 0 {
		t.Errorf("Water Shield restored %v mana from Healing Stream crits, want none", got)
	}
}

// Restorative Totems raises Healing Stream 10% a rank. Purification
// (16178) does not: its SpellClassMask_0 448 holds Healing Wave, Lesser
// Healing Wave and Chain Heal, and mask_2 16 Riptide (1.60.1.70707);
// Healing Stream's heal is 0x2000.
func TestRestorativeTotemsRaiseHealingStreamAndPurificationDoesNot(t *testing.T) {
	talents := restoTalents(map[int]int{posRestorativeTotems: 5, posPurification: 5})
	result := runHealingStreamWithCrit(t, talents, noCrit)

	events, gain := healthGained(result, healingStreamTopHealID)
	wantTick := (healingStreamTopRankHeal + healingStreamCoefficient*testHealingPower) * 1.5
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
	if got, want := healer.ManaTideTotem[4].Cost.GetCurrentCost(), 60*0.75; math.Abs(got-want) > 1e-6 {
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

// Healing Stream's ticks (10461, cast by the shaman's totem 10463) are the
// shaman's own healing, so they count in its healing per second: a fight
// that only drops the totem heals for what its ticks heal.
func TestHealingStreamCountsInTheShamansHealing(t *testing.T) {
	result := runHealingStreamWithCrit(t, "", noCrit)
	_, gain := healthGained(result, healingStreamTopHealID)
	shaman := result.RaidMetrics.Parties[0].Players[0]
	if got, floor := shaman.Hps.Avg*healingStreamFightLength, gain; got < floor*0.999 {
		t.Errorf("the shaman's healing is %.0f over the fight, want at least the %.0f its totem healed the party", got, floor)
	}
}
