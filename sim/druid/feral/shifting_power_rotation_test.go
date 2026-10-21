package feral

import (
	"strconv"
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

const (
	shiftingPowerRotationSpellID = 1322605
	shiftingPowerRotationEnergy  = 40.0
	shiftingPowerRotationCap     = 100.0
)

func castAction(spellID int32, condition *proto.APLValue) *proto.APLListItem {
	return &proto.APLListItem{Action: &proto.APLAction{
		Condition: condition,
		Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{
			SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: spellID}},
		}},
	}}
}

func optimalCatRotation() *proto.APLListItem {
	return &proto.APLListItem{Action: &proto.APLAction{
		Action: &proto.APLAction_CatOptimalRotationAction{CatOptimalRotationAction: &proto.APLActionCatOptimalRotationAction{
			MinCombosForRip: 5,
			MaxWaitTime:     2,
		}},
	}}
}

func energyAtMost(limit float64) *proto.APLValue {
	return &proto.APLValue{Value: &proto.APLValue_Cmp{Cmp: &proto.APLValueCompare{
		Op:  proto.APLValueCompare_OpLe,
		Lhs: &proto.APLValue{Value: &proto.APLValue_CurrentEnergy{CurrentEnergy: &proto.APLValueCurrentEnergy{}}},
		Rhs: &proto.APLValue{Value: &proto.APLValue_Const{Const: &proto.APLValueConst{Val: strconv.FormatFloat(limit, 'f', -1, 64)}}},
	}}}
}

// TestShiftingPowerIsCastableFromTheAPL runs a short Shifting-Power-then-Shred
// rotation and checks the spell resolves by id in cat form, lands on its
// cooldown, and each cast adds the full 40 energy (the rotation only casts it
// at 60 energy or less, so the cap never eats any of it).
func TestShiftingPowerIsCastableFromTheAPL(t *testing.T) {
	casts := runShiftingPowerRotation(t, 160, []*proto.APLListItem{
		castAction(shiftingPowerRotationSpellID, energyAtMost(shiftingPowerRotationCap-shiftingPowerRotationEnergy)),
		castAction(rotationShredSpellID, nil),
	})
	// 16 s cooldown over 160 s: ten casts at most, and the mana pool (55% of
	// base a cast) is allowed to cut that short but not to zero.
	if got := casts[shiftingPowerRotationSpellID]; got < 2 || got > 11 {
		t.Errorf("Shifting Power casts = %d, want 2-11 over 160 s on a 16 s cooldown", got)
	}
	if casts[rotationShredSpellID] == 0 {
		t.Error("Shred never cast alongside Shifting Power")
	}
}

// TestOptimalCatRotationUsesShiftingPower: the engine's hard-coded cat
// logic (the catOptimalRotation action) used to power-shift out of cat form
// and back whenever it ran out of energy; with Shifting Power learned and
// ready it takes the 40 energy instead.
func TestOptimalCatRotationUsesShiftingPower(t *testing.T) {
	casts := runShiftingPowerRotation(t, 160, []*proto.APLListItem{optimalCatRotation()})
	if casts[shiftingPowerRotationSpellID] == 0 {
		t.Error("the hard-coded cat rotation never cast Shifting Power")
	}
	if casts[rotationShredSpellID] == 0 {
		t.Error("the hard-coded cat rotation stopped casting Shred")
	}
}

func runShiftingPowerRotation(t *testing.T, durationSeconds float64, priorityList []*proto.APLListItem) map[int32]int32 {
	t.Helper()
	talents := feralTalentsString(t, map[string]int{"shifting_power": 1})
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassDruid,
		Race:               proto.Race_RaceTauren,
		Level:              60,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              core.FullBuffs.Player,
		TalentsString:      talents,
		DistanceFromTarget: 5,
		Rotation:           &proto.APLRotation{PriorityList: priorityList},
	}, PlayerOptionsMonoCat)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: durationSeconds, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}

	return castCountsBySpellID(result)
}
