package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	googleProto "google.golang.org/protobuf/proto"
)

// Judgement of the Crusader raises the Holy damage the target takes by its
// bonus times each spell's spell-power coefficient. Consecration's ticks
// carry a coefficient (0.042 a tick on the first targets), so the debuff
// must reach them, at raid 60 (the preset's debuff) as for a solo paladin.
func consecrationDamage(t *testing.T, crusader proto.TristateEffect) float64 {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassPaladin,
		Race:               proto.Race_RaceHuman,
		Level:              60,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              core.FullBuffs.Player,
		TalentsString:      Phase45RetTalents,
		Rotation:           consecrationRotation,
		DistanceFromTarget: 5,
	}, PlayerOptionsSealofRighteousness)
	debuffs := googleProto.Clone(core.FullBuffs.Debuffs).(*proto.Debuffs)
	debuffs.JudgementOfTheCrusader = crusader
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, debuffs)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 30, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}
	metrics := findActionMetrics(result, consecrationSpellID)
	if metrics == nil {
		t.Fatal("Consecration never cast")
	}
	var damage float64
	for _, target := range metrics.Targets {
		damage += target.Damage
	}
	return damage
}

func TestCrusaderReachesConsecrationTicks(t *testing.T) {
	without := consecrationDamage(t, proto.TristateEffect_TristateEffectMissing)
	with := consecrationDamage(t, proto.TristateEffect_TristateEffectRegular)
	if with <= without {
		t.Errorf("Consecration damage with Judgement of the Crusader %.1f, without %.1f: the debuff never reached the ticks", with, without)
	}
}
