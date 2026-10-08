package holy

import (
	"encoding/json"
	"fmt"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	_ "github.com/wowsims/classic/sim/common" // imported to get caster sets included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/healsim"
	"github.com/wowsims/classic/sim/paladin/paladintest"
)

func init() {
	RegisterHolyPaladin()
}

// fight is one healing sim of a Holy Paladin against the shared fake raid.
type fight struct {
	level    int32
	talents  map[string]int
	bonus    stats.Stats
	rotation string
	duration float64
	model    *proto.RaidDamageModel
	// realMana runs the healer on the mana its stats give it instead of the
	// plentiful pool the numeric tests use.
	realMana bool
}

const (
	tankIndex      = healsim.TankIndex
	defaultLevel   = 60
	defaultSeconds = 20
	// plentyOfMana keeps a numeric test from running dry; the test is about
	// one cast's numbers, not the mana game.
	plentyOfMana = 1e6
)

func paladinTalents(t *testing.T, ranks map[string]int) string {
	t.Helper()
	return paladintest.TalentString(t, ranks)
}

func (f fight) player(t *testing.T) *proto.Player {
	t.Helper()
	level := f.level
	if level == 0 {
		level = defaultLevel
	}
	bonus := f.bonus
	if !f.realMana {
		bonus[stats.Mana] += plentyOfMana
	}
	return core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassPaladin,
		Race:          proto.Race_RaceHuman,
		Level:         level,
		Equipment:     &proto.EquipmentSpec{},
		Buffs:         &proto.IndividualBuffs{},
		TalentsString: paladinTalents(t, f.talents),
		Rotation:      parseRotation(t, f.rotation),
		BonusStats:    &proto.UnitStats{Stats: bonus.ToFloatArray()},
	}, &proto.Player_HolyPaladin{HolyPaladin: &proto.HolyPaladin{Options: &proto.PaladinOptions{
		PrimarySeal: proto.PaladinSeal_Righteousness,
	}}})
}

// run runs the fight for the given iterations and returns the result.
func (f fight) run(t *testing.T, iterations int32) *proto.RaidSimResult {
	t.Helper()
	duration := f.duration
	if duration == 0 {
		duration = defaultSeconds
	}
	model := f.model
	if model == nil {
		model = idleRaid()
	}
	result := core.RunRaidSim(healsim.Request(f.player(t), model, duration, iterations))
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}
	return result
}

// idleRaid is a fake raid that takes no damage: its members exist to be
// healed and report raw healing, and nothing hurts them.
func idleRaid() *proto.RaidDamageModel {
	return &proto.RaidDamageModel{Profile: "idle", TankHealth: 1e9, MemberHealth: 1e9}
}

func parseRotation(t *testing.T, rotation string) *proto.APLRotation {
	t.Helper()
	if rotation == "" {
		return nil
	}
	apl := &proto.APLRotation{}
	if err := protojson.Unmarshal([]byte(rotation), apl); err != nil {
		t.Fatalf("rotation does not parse: %v", err)
	}
	return apl
}

// castLoop is a rotation that casts the given spell on the tank whenever
// it can, and nothing else.
func castLoop(spellID int32) string {
	return castLoopOn(spellID, tankIndex)
}

func castLoopOn(spellID int32, target int) string {
	return rotationOf(castItem(spellID, target))
}

func castItem(spellID int32, target int) map[string]any {
	return map[string]any{"action": map[string]any{"castSpell": map[string]any{
		"spellId": map[string]any{"spellId": spellID},
		"target":  map[string]any{"type": "Player", "index": target},
	}}}
}

func rotationOf(items ...map[string]any) string {
	encoded, err := json.Marshal(map[string]any{"type": "TypeAPL", "priorityList": items})
	if err != nil {
		panic(fmt.Sprintf("rotation does not encode: %v", err))
	}
	return string(encoded)
}

// spellTotals sums one action's metrics over every target.
type spellTotals struct {
	casts, hits, crits   int32
	healing, critHealing float64
	effective            float64
}

// normalHeal is the average heal that did not crit; hits counts those
// only, crits are counted apart.
func (totals spellTotals) normalHeal() float64 {
	return (totals.healing - totals.critHealing) / float64(totals.hits)
}

// landed is how many heals landed. A passive spell (a heal a cast triggers)
// reports no casts of its own, so tests count what landed.
func (totals spellTotals) landed() int32 {
	return totals.hits + totals.crits
}

func (totals spellTotals) critHeal() float64 {
	return totals.critHealing / float64(totals.crits)
}

func totalsOf(result *proto.RaidSimResult, spellID int32) spellTotals {
	var totals spellTotals
	for _, action := range result.RaidMetrics.Parties[0].Players[healsim.HealerIndex].Actions {
		if action.Id.GetSpellId() != spellID {
			continue
		}
		for _, target := range action.Targets {
			totals.casts += target.Casts
			totals.hits += target.Hits
			totals.crits += target.Crits
			totals.healing += target.Healing
			totals.critHealing += target.CritHealing
			totals.effective += target.EffectiveHealing
		}
	}
	return totals
}

// resourceGain is the mana an action id gave the healer, summed over the
// run's iterations.
func resourceGain(result *proto.RaidSimResult, spellID int32) float64 {
	var gain float64
	for _, resource := range result.RaidMetrics.Parties[0].Players[healsim.HealerIndex].Resources {
		if resource.Id.GetSpellId() == spellID && resource.Type == proto.ResourceType_ResourceTypeMana {
			gain += resource.Gain
		}
	}
	return gain
}

// castTimeMS is the total time the healer spent casting a spell, over the run.
func castTimeMS(result *proto.RaidSimResult, spellID int32) float64 {
	var total float64
	for _, action := range result.RaidMetrics.Parties[0].Players[healsim.HealerIndex].Actions {
		if action.Id.GetSpellId() == spellID {
			for _, target := range action.Targets {
				total += target.CastTimeMs
			}
		}
	}
	return total
}

// whileBefore is a priority-list item that runs only before the given time.
func whileBefore(seconds string, item map[string]any) map[string]any {
	action := item["action"].(map[string]any)
	action["condition"] = map[string]any{"cmp": map[string]any{
		"op":  "OpLt",
		"lhs": map[string]any{"currentTime": map[string]any{}},
		"rhs": map[string]any{"const": map[string]any{"val": seconds}},
	}}
	return item
}
