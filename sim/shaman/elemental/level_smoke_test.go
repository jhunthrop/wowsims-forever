package elemental

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/shaman"
)

// TestLevelSmoke is the level-aware sim design's engine smoke test; see
// sim/hunter/level_smoke_test.go for the full explanation.
func TestLevelSmoke(t *testing.T) {
	core.RunLevelSmoke(t, core.LevelSmokePreset{
		Label:       "ElementalShaman",
		Class:       proto.Class_ClassShaman,
		Race:        proto.Race_RaceTroll,
		Talents:     DefaultTalents,
		SpecOptions: PlayerOptionsAdaptive,
	})
}

// spellFamily is one ranked spell an elemental APL names by spellId: its
// per-rank learn levels and per-rank spellIds, both already exported
// from the shaman package's generated constants.
type spellFamily struct {
	levels   []int
	spellIDs []int32
}

func families() []spellFamily {
	return []spellFamily{
		{levels: shaman.LightningBoltLevel[:], spellIDs: shaman.LightningBoltSpellId[:]},
		{levels: shaman.SearingTotemLevel[:], spellIDs: shaman.SearingTotemSpellId[:]},
		{levels: shaman.MagmaTotemLevel[:], spellIDs: shaman.MagmaTotemSpellId[:]},
		{levels: shaman.FlameShockLevel[:], spellIDs: shaman.FlameShockSpellId[:]},
		{levels: shaman.ChainLightningLevel[:], spellIDs: shaman.ChainLightningSpellId[:]},
		{levels: shaman.EarthShockLevel[:], spellIDs: shaman.EarthShockSpellId[:]},
		{levels: shaman.FrostShockLevel[:], spellIDs: shaman.FrostShockSpellId[:]},
	}
}

// resolve mirrors sim/request's rewriteRotationRanks (a different repo:
// jhunthrop/foreversixty/sim/request, not this engine fork) closely
// enough to exercise the same engine code paths: given any rank's
// spellId for a known family, return the highest rank actually learned
// at level. The bool is false ("drop this action") when no real rank is
// learned yet. An id this test doesn't recognize as ranked (buffs,
// totems this list doesn't carry, non-ranked abilities) passes through
// unchanged rather than being dropped, since an unrecognized id is far
// more likely to be something this smoke doesn't need to rewrite than a
// ranked spell it forgot.
func resolve(level int32, id int32) (int32, bool) {
	for _, f := range families() {
		for _, sid := range f.spellIDs {
			if sid != id {
				continue
			}
			rank := 0
			for r, lvl := range f.levels {
				if int32(lvl) <= level {
					rank = r
				}
			}
			if rank == 0 {
				return 0, false
			}
			return f.spellIDs[rank], true
		}
	}
	return id, true
}

func rewriteRankedSpellIDs(node any, level int32, drop *bool) {
	switch v := node.(type) {
	case map[string]any:
		if raw, ok := v["spellId"]; ok {
			if num, isNumber := raw.(float64); isNumber {
				id := int32(num)
				newID, learned := resolve(level, id)
				if !learned {
					*drop = true
					return
				}
				v["spellId"] = float64(newID)
				return
			}
		}
		for _, val := range v {
			rewriteRankedSpellIDs(val, level, drop)
		}
	case []any:
		for _, item := range v {
			rewriteRankedSpellIDs(item, level, drop)
		}
	}
}

func filterRankedActions(v any, level int32) any {
	arr, ok := v.([]any)
	if !ok {
		return v
	}
	kept := make([]any, 0, len(arr))
	for _, item := range arr {
		drop := false
		rewriteRankedSpellIDs(item, level, &drop)
		if !drop {
			kept = append(kept, item)
		}
	}
	return kept
}

// rewriteForLevel is the JSON-structural equivalent, run here by hand
// against this package's own generated rank tables, of what
// sim/request's rewriteRotationRanks does in the site repo before
// handing a rotation to this engine (level-aware-sim-design.md point 4):
// rewrite prepullActions/priorityList so every ranked spellId names the
// rank level has actually learned, dropping an action whose spell has no
// learned rank. Without it this test would just hand the engine the raw
// max-rank ids the embedded APL names, which the AI silently treats as
// unavailable at anything below 60 - never exercising the ApplyEffects
// of the rank the character actually has.
func rewriteForLevel(raw []byte, level int32) []byte {
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		panic(err)
	}
	for _, key := range []string{"prepullActions", "priorityList"} {
		if v, ok := root[key]; ok {
			root[key] = filterRankedActions(v, level)
		}
	}
	out, err := json.Marshal(root)
	if err != nil {
		panic(err)
	}
	return out
}

// TestLevelSmokeRotation goes further than TestLevelSmoke: it hands the
// engine elemental's actual preset rotations, rank-rewritten by hand for
// each level the way sim/request rewrites them for real, so every
// registered spell is not just built but cast during the sim. This is
// the shape a shaman-elemental defect was found in at level 38 (a
// bracket-keyed weapon-imbue table returning an unlearned rank for any
// level between its brackets - fixed in shaman/{rockbiter,windfury}
// weapon.go); it is kept here, rotation-driven and across every level
// rather than just 38, as the regression test for that class of defect.
func TestLevelSmokeRotation(t *testing.T) {
	for _, aplFile := range []string{"default", "forever_elemental"} {
		raw, err := os.ReadFile(fmt.Sprintf("../../../ui/elemental_shaman/apls/%s.apl.json", aplFile))
		if err != nil {
			t.Fatal(err)
		}

		for _, level := range core.LevelSmokeLevels {
			level := level
			t.Run(fmt.Sprintf("%s/L%d", aplFile, level), func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("panic at level %d: %v", level, r)
					}
				}()

				rotation := core.APLRotationFromJsonString(string(rewriteForLevel(raw, level)))

				player := core.WithSpec(
					&proto.Player{
						Class:              proto.Class_ClassShaman,
						Race:               proto.Race_RaceTroll,
						Level:              level,
						Equipment:          &proto.EquipmentSpec{},
						Buffs:              core.FullBuffs.Player,
						TalentsString:      DefaultTalents,
						DistanceFromTarget: 20,
						Rotation:           rotation,
					},
					PlayerOptionsAdaptive,
				)
				raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

				result := core.RunRaidSim(&proto.RaidSimRequest{
					Raid: raid,
					Encounter: &proto.Encounter{
						Duration: 90,
						Targets:  []*proto.Target{core.NewDefaultTarget()},
					},
					SimOptions: &proto.SimOptions{
						Iterations: 1,
						RandomSeed: 1,
						IsTest:     true,
					},
				})
				if result.Error != nil {
					t.Fatalf("level %d sim error: %s", level, result.Error.Message)
				}
			})
		}
	}
}
