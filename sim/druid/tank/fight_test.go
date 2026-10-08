package tank

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/druid"
	"github.com/wowsims/classic/sim/druid/feral"
)

const (
	bearRotationDir  = "../../../ui/feral_tank_druid/apls"
	bearRotationName = "forever_feral_bear"
	catRotationDir   = "../../../ui/feral_druid/apls"
	catRotationName  = "forever_feral"

	barkskinSpellID = 22812
)

func bearRotation() *proto.APLRotation {
	return core.GetAplRotation(bearRotationDir, bearRotationName).Rotation
}

// withoutSpell is the rotation with every action that casts the spell cut.
func withoutSpell(rotation *proto.APLRotation, spellID int32) *proto.APLRotation {
	kept := make([]*proto.APLListItem, 0, len(rotation.PriorityList))
	for _, item := range rotation.PriorityList {
		cast := item.GetAction().GetCastSpell()
		if cast != nil && cast.SpellId.GetSpellId() == spellID {
			continue
		}
		kept = append(kept, item)
	}
	return &proto.APLRotation{Type: rotation.Type, PriorityList: kept}
}

func logFight(t *testing.T, label string, m *proto.UnitMetrics) {
	t.Helper()
	t.Logf("%s: dps %.1f threat/s %.1f dtps %.1f tmi %.1f chance of death %.4f",
		label, m.Dps.Avg, m.Threat.Avg, m.Dtps.Avg, m.Tmi.Avg, m.ChanceOfDeath)
}

// A level 60 bear in decent gear survives the 180 second fight with the
// healer the harness gives it.
func TestBearSurvivesTheTankFight(t *testing.T) {
	player := bearPlayer(60, druid.ForeverBearTalents, loadGear(t), bearRotation())
	m := runHarness(t, player, harnessIterations)
	logFight(t, "bear", m)
	if m.ChanceOfDeath != 0 {
		t.Errorf("chance of death = %v, want 0", m.ChanceOfDeath)
	}
	if m.Dtps.Avg <= 0 {
		t.Error("the bear took no damage: the boss is not hitting it")
	}
}

// Barkskin used, the bear takes less damage than with it left out.
func TestBarkskinLowersDamageTaken(t *testing.T) {
	gear := loadGear(t)
	with := runHarness(t, bearPlayer(60, druid.ForeverBearTalents, gear, bearRotation()), harnessIterations)
	without := runHarness(t, bearPlayer(60, druid.ForeverBearTalents, gear, withoutSpell(bearRotation(), barkskinSpellID)), harnessIterations)
	logFight(t, "with Barkskin", with)
	logFight(t, "without Barkskin", without)
	if with.Dtps.Avg >= without.Dtps.Avg {
		t.Errorf("damage taken per second with Barkskin %.1f is not under %.1f without it", with.Dtps.Avg, without.Dtps.Avg)
	}
}

// The bear makes more threat than the same gear running the cat's damage
// rotation: the form's 1.3x against the cat's 0.71x, and a threat kit.
func TestBearOutThreatsACatInTheSameGear(t *testing.T) {
	gear := loadGear(t)
	bear := runHarness(t, bearPlayer(60, druid.ForeverBearTalents, gear, bearRotation()), harnessIterations)

	cat := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassDruid,
		Race:               proto.Race_RaceTauren,
		Level:              60,
		Equipment:          gear,
		Consumes:           &proto.Consumes{},
		Buffs:              core.FullBuffs.Player,
		TalentsString:      "500005301-55000230320202051-15",
		Rotation:           core.GetAplRotation(catRotationDir, catRotationName).Rotation,
		InFrontOfTarget:    true,
		DistanceFromTarget: 5,
	}, &proto.Player_FeralDruid{FeralDruid: &proto.FeralDruid{Options: &proto.FeralDruid_Options{
		InnervateTarget: &proto.UnitReference{}, LatencyMs: 100, AssumeBleedActive: true,
	}}})
	catMetrics := runHarness(t, cat, harnessIterations)
	logFight(t, "bear", bear)
	logFight(t, "cat", catMetrics)
	if bear.Threat.Avg <= catMetrics.Threat.Avg {
		t.Errorf("bear threat per second %.1f is not over the cat's %.1f", bear.Threat.Avg, catMetrics.Threat.Avg)
	}
}

func init() {
	feral.RegisterFeralDruid()
}
