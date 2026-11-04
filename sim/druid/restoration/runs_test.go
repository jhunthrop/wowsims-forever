package restoration

import (
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/healsim"
)

const (
	defaultRotationPath = "../../../ui/restoration_druid/apls/forever_restoration.apl.json"
	// wildGrowthTickDecay is the 5% Wild Growth moves from tick to tick.
	wildGrowthTickDecay = 0.05
	// healCritMultiplier is an untalented heal's critical multiplier: heals
	// over time crit since client 1.60.1.70707 marks them PERIODIC_CAN_CRIT.
	healCritMultiplier = 1.5
)

// castOnce is a rotation that casts the spell once, at the start, at the
// raid member with the given index.
func castOnce(spellID, targetIndex int32) *proto.APLRotation {
	return core.APLRotationFromJsonString(`{"type":"TypeAPL","priorityList":[{"action":{
		"condition":{"cmp":{"op":"OpLt","lhs":{"currentTime":{}},"rhs":{"const":{"val":"1s"}}}},
		"castSpell":{"spellId":{"spellId":` + strconv.Itoa(int(spellID)) + `},"target":{"type":"Player","index":` + strconv.Itoa(int(targetIndex)) + `}}}}]}`)
}

// runOnce runs the player for the duration against the shared profile, once,
// with the debug log on.
func runOnce(t *testing.T, player *proto.Player, duration float64) *proto.RaidSimResult {
	t.Helper()
	req := healsim.Request(player, healsim.TestProfile(), duration, 1)
	req.SimOptions.Debug = true
	result := core.RunRaidSim(req)
	if result.Error != nil {
		t.Fatalf("sim failed: %s", result.Error.Message)
	}
	return result
}

// unitIndex is the unit index the metrics report for a raid player: the sim
// numbers its units from the encounter's enemy target, which is unit 0.
func unitIndex(raidPlayer int32) int32 { return raidPlayer + 1 }

// metricsByUnit is the healer's metrics for the spell, per unit index.
func metricsByUnit(result *proto.RaidSimResult, spellID int32) map[int32]*proto.TargetedActionMetrics {
	byUnit := map[int32]*proto.TargetedActionMetrics{}
	for _, action := range result.RaidMetrics.Parties[0].Players[healsim.HealerIndex].Actions {
		if action.Id.GetSpellId() != spellID {
			continue
		}
		for _, target := range action.Targets {
			byUnit[target.UnitIndex] = target
		}
	}
	return byUnit
}

// totalCasts is how many times the healer cast the spell, at whoever.
func totalCasts(result *proto.RaidSimResult, spellID int32) int32 {
	var casts int32
	for _, target := range metricsByUnit(result, spellID) {
		casts += target.Casts
	}
	return casts
}

var tickLog = regexp.MustCompile(`\[Target Dummy 1 \(#2\)\] \{SpellID: (\d+)\} tick (Hit|Crit) for ([0-9.]+) healing`)

// tickHeals is the amounts the spell's ticks landed on the first fake member,
// a critical tick taken back to what it would have healed without the crit.
func tickHeals(result *proto.RaidSimResult, spellID int32) []float64 {
	var heals []float64
	for _, match := range tickLog.FindAllStringSubmatch(result.Logs, -1) {
		if id, _ := strconv.Atoi(match[1]); int32(id) != spellID {
			continue
		}
		amount, _ := strconv.ParseFloat(match[3], 64)
		if match[2] == "Crit" {
			amount /= healCritMultiplier
		}
		heals = append(heals, amount)
	}
	return heals
}

// assertPartyHealing checks the spell ticked ticks times on every unit in
// the healer's party (the healer and fake members 1 to 4), for perTick on a
// plain tick and healCritMultiplier times that on a critical one, and never
// on the tank, who stands in the other party.
func assertPartyHealing(t *testing.T, result *proto.RaidSimResult, spellID int32, ticks int32, perTick float64) {
	t.Helper()
	byUnit := metricsByUnit(result, spellID)
	for player := int32(healsim.HealerIndex); player < healsim.TankIndex; player++ {
		unit := unitIndex(player)
		got, ok := byUnit[unit]
		if !ok {
			t.Errorf("spell %d never reached unit %d of the healer's party", spellID, unit)
			continue
		}
		want := perTick * (float64(got.Ticks) + healCritMultiplier*float64(got.CritTicks))
		if got.Ticks+got.CritTicks != ticks || math.Abs(got.Healing-want) > 0.01*want {
			t.Errorf("spell %d on unit %d: %d ticks (%d crits) for %.1f, want %d for %.1f", spellID, unit, got.Ticks+got.CritTicks, got.CritTicks, got.Healing, ticks, want)
		}
	}
	if tank, ok := byUnit[unitIndex(healsim.TankIndex)]; ok && tank.Ticks+tank.CritTicks != 0 {
		t.Errorf("spell %d healed the tank in the other party: %+v", spellID, tank)
	}
}

func TestTranquilityHealsTheWholePartyEveryTwoSeconds(t *testing.T) {
	power := healerBonusStats[stats.HealingPower]
	player := newPlayer(60, "", healerBonusStats, castOnce(9863, healsim.TankIndex))
	result := runOnce(t, player, 20)

	const rank4TickAtLevel60, coefficient = 285, 0.067
	assertPartyHealing(t, result, 9863, 5, rank4TickAtLevel60+coefficient*power)
	if casts := totalCasts(result, 9863); casts != 1 {
		t.Errorf("Tranquility cast %d times, want once", casts)
	}
}

func TestWildGrowthHealsTheTargetsPartyOverSevenTicksThatFadeAway(t *testing.T) {
	power := healerBonusStats[stats.HealingPower]
	talents := talentsString(t, map[string]int{"wild_growth": 1})
	player := newPlayer(60, talents, healerBonusStats, castOnce(1238215, 1))
	result := runOnce(t, player, 20)

	const rank3TickAtLevel60, coefficient = 97, 0.033
	average := rank3TickAtLevel60 + coefficient*power
	assertPartyHealing(t, result, 1238215, 7, average)

	ticks := tickHeals(result, 1238215)
	if len(ticks) != 7 {
		t.Fatalf("Wild Growth ticked %d times on a member, want 7", len(ticks))
	}
	for i, heal := range ticks {
		want := average * (1 + wildGrowthTickDecay*float64(3-i))
		if math.Abs(heal-want) > 0.5 {
			t.Errorf("tick %d healed %.1f, want %.1f", i+1, heal, want)
		}
	}
}

func TestWildGrowthOnTheTankHealsOnlyHisParty(t *testing.T) {
	talents := talentsString(t, map[string]int{"wild_growth": 1})
	player := newPlayer(60, talents, healerBonusStats, castOnce(1238215, healsim.TankIndex))
	byUnit := metricsByUnit(runOnce(t, player, 20), 1238215)
	for unit, metrics := range byUnit {
		if healed := metrics.Ticks > 0; healed != (unit == unitIndex(healsim.TankIndex)) {
			t.Errorf("Wild Growth on the tank: unit %d healed = %v", unit, healed)
		}
	}
}

// loadDefaultRotation is the written rotation the site ships.
func loadDefaultRotation(t *testing.T) *proto.APLRotation {
	t.Helper()
	raw, err := os.ReadFile(defaultRotationPath)
	if err != nil {
		t.Fatalf("the default rotation: %v", err)
	}
	return core.APLRotationFromJsonString(string(raw))
}

// castIDs is the rotation's healing kit: what it has to be seen casting.
// Rejuvenation is rank 10 (9841) on the tank and rank 7 (8910) on the party (round 15 wave 2)
// since site round 13 (rank 4, 2090, from round 9 left 40% of the raid
// fight's mana unspent).
// Wild Growth is not in it: the site's guide build does not take the
// talent, so the shipped rotation has no Wild Growth line since client
// 1.60.1.70707.
var castIDs = struct{ healingTouch, regrowth, rejuvenation, partyRejuvenation, tranquility, swiftmend, innervate int32 }{
	healingTouch: 9889, regrowth: 9858, rejuvenation: 9841, partyRejuvenation: 8910, tranquility: 9863,
	swiftmend: 18562, innervate: 29166,
}

func TestDefaultRotationHealsTheTestRaid(t *testing.T) {
	player := newPlayer(60, StandardTalents, healerBonusStats, loadDefaultRotation(t))
	summary := runHealing(t, player, 180, 20)

	if summary.EffectiveHPS <= 0 {
		t.Fatalf("effective HPS = %v", summary.EffectiveHPS)
	}
	if summary.OverhealPct < 0 || summary.OverhealPct >= 0.8 {
		t.Errorf("overheal = %.2f, want 0 to under 0.8", summary.OverhealPct)
	}
	if summary.TimeToOOMSeconds < 150 {
		t.Errorf("out of mana after %.0fs, want the mana to last most of 180s", summary.TimeToOOMSeconds)
	}
	for name, id := range map[string]int32{
		"Healing Touch": castIDs.healingTouch, "Regrowth": castIDs.regrowth, "Rejuvenation": castIDs.rejuvenation,
		"party Rejuvenation": castIDs.partyRejuvenation,
		"Tranquility":        castIDs.tranquility, "Swiftmend": castIDs.swiftmend, "Innervate": castIDs.innervate,
		// Nature's Swiftness is the tank-under-35% save: with rank 10 on the
		// tank this light test raid never gets him that low. The site's raid
		// profile still casts it (spec-breakdown -heal, about 1.6 a fight).
	} {
		if summary.Casts[id] == 0 {
			t.Errorf("the rotation never cast %s (%d)", name, id)
		}
	}
}

// TestDefaultRotationManaMatchesCastCosts: the mana the healer spends on its
// spells is each spell's current cost times its casts, no more and no less.
func TestDefaultRotationManaMatchesCastCosts(t *testing.T) {
	player := newPlayer(60, StandardTalents, healerBonusStats, loadDefaultRotation(t))
	result := runOnce(t, player, 120)
	resto, _ := newDruid(t, player)

	for _, resource := range result.RaidMetrics.Parties[0].Players[healsim.HealerIndex].Resources {
		spell := resto.GetSpell(core.ActionID{SpellID: resource.Id.GetSpellId()})
		if resource.Type != proto.ResourceType_ResourceTypeMana || resource.Gain >= 0 || spell == nil {
			continue
		}
		want := spell.Cost.GetCurrentCost() * float64(resource.Events)
		if math.Abs(-resource.Gain-want) > 0.01*want {
			t.Errorf("%s spent %.0f mana over %d casts, want %.0f", spell.ActionID, -resource.Gain, resource.Events, want)
		}
	}
}

var firstCast = regexp.MustCompile(`Casting \{SpellID: (\d+)\}`)

// TestDefaultRotationWithoutTheTalentsOpensWithTheTanksHeals: a druid without
// Nature's Swiftness, Swiftmend and Wild Growth has none of their spells or
// auras, and a condition naming one of those must not turn a cast on
// unconditionally (it once opened a level 30 druid with a Healing Touch on a
// full-health tank, and then another every cast).
func TestDefaultRotationWithoutTheTalentsOpensWithTheTanksHeals(t *testing.T) {
	player := newPlayer(60, "", healerBonusStats, loadDefaultRotation(t))
	logs := runOnce(t, player, 30).Logs

	first := firstCast.FindStringSubmatch(logs)
	if first == nil || first[1] != strconv.Itoa(int(castIDs.regrowth)) {
		t.Errorf("the first cast was %v, want Regrowth (%d) on the tank", first, castIDs.regrowth)
	}
	if got := len(regexp.MustCompile(`Casting \{SpellID: `+strconv.Itoa(int(castIDs.healingTouch))+`\}`).FindAllString(logs, -1)); got > 3 {
		t.Errorf("Healing Touch cast %d times in 30s of a tank who is mostly above 55%% health", got)
	}
}

// readLeylineWarning is the one warning a druid who is not an Alliance
// Skyborne gets: the rotation's Read Leyline line names a racial only that
// race registers (sim/core/racial_read_leyline.go), so the line is skipped.
const readLeylineWarning = "does not know spell {SpellID: 1259705}"

func TestDefaultRotationRunsWithoutWarnings(t *testing.T) {
	for _, race := range []proto.Race{proto.Race_RaceHighOrderSkyborne, proto.Race_RaceTauren, proto.Race_RaceNightElf} {
		t.Run(race.String(), func(t *testing.T) {
			player := newPlayer(60, StandardTalents, healerBonusStats, loadDefaultRotation(t))
			player.Race = race
			sim := healsim.Request(player, healsim.TestProfile(), 60, 1)
			result := core.ComputeStats(&proto.ComputeStatsRequest{Raid: sim.Raid, Encounter: sim.Encounter})
			if result.ErrorResult != "" {
				t.Fatal(result.ErrorResult)
			}
			for i, action := range result.RaidStats.Parties[0].Players[healsim.HealerIndex].RotationStats.PriorityList {
				for _, warning := range action.Warnings {
					if race != proto.Race_RaceHighOrderSkyborne && strings.Contains(warning, readLeylineWarning) {
						continue
					}
					t.Errorf("rotation action %d: %s", i, warning)
				}
			}
		})
	}
}

// The written rotation opens with the autocast line, so a druid that
// carries a Major Mana Potion and a Demonic Rune drinks both: they are
// self-cast, and the healer's current target is a friend.
func TestDefaultRotationUsesTheManaConsumables(t *testing.T) {
	player := newPlayer(60, StandardTalents, healerBonusStats, loadDefaultRotation(t))
	player.Consumes = healsim.ManaConsumables()
	result := core.RunRaidSim(healsim.Request(player, healsim.TestProfile(), 180, 20))
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}
	metrics := result.RaidMetrics.Parties[0].Players[healsim.HealerIndex]
	for _, name := range healsim.UnusedManaConsumables(metrics) {
		t.Errorf("the druid never used its %s", name)
	}
}
