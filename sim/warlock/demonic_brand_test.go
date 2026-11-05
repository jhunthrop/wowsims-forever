package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// demonicBrandRank3TalentsString takes only Demonic Brand 3/3 (field 13 of
// the 19-node Demonology tree).
const demonicBrandRank3TalentsString = "-00000000000003"

func newDemonicBrandTestWarlock(t *testing.T) (*core.Simulation, *Warlock, *core.Unit) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassWarlock,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			TalentsString: demonicBrandRank3TalentsString,
		},
		&proto.Player_Warlock{
			Warlock: &proto.Warlock{
				Options: &proto.WarlockOptions{
					Armor:       proto.WarlockOptions_NoArmor,
					Summon:      proto.WarlockOptions_Succubus,
					WeaponImbue: proto.WarlockOptions_NoWeaponImbue,
				},
			},
		},
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	for i := 0; i < 10 && sim.CurrentTime <= 0; i++ {
		sim.Step()
	}
	agent, ok := sim.Raid.Parties[0].Players[0].(WarlockAgent)
	if !ok {
		t.Fatal("the raid's first player is not a warlock agent")
	}
	return sim, agent.GetWarlock(), sim.Encounter.TargetUnits[0]
}

// The talent text (1293695, every rank) and Wowhead's Forever tooltips for
// 1293697 (Shadow) and 1293698 (Fire): a consumed brand deals
// ((level - 26) x 1.5 + 14) to ((level - 26) x 1.5 + 17), plus 0.078 x the
// warlock's spell power of that school. At 60: 65 to 68.
func TestDemonicBrandDamageFollowsTheTalentText(t *testing.T) {
	for _, tc := range []struct{ level, lo, hi int }{{60, 65, 68}, {40, 35, 38}, {26, 14, 17}} {
		lo, hi := demonicBrandDamageRange(tc.level)
		if lo != float64(tc.lo) || hi != float64(tc.hi) {
			t.Errorf("level %d: %v to %v, want %d to %d", tc.level, lo, hi, tc.lo, tc.hi)
		}
	}
}

// A landed Searing Pain brands the target with 6 charges at rank 3 (curve
// 103069: 2/4/6) and its threat drops 50% (curve 83195: -17/-33/-50); each
// landed Succubus attack on the target spends a charge on a Shadow hit
// (1293697).
func TestDemonicBrandSearingPainBrandsAndThePetSpendsIt(t *testing.T) {
	sim, built, target := newDemonicBrandTestWarlock(t)
	if len(built.SearingPain) == 0 {
		t.Fatal("level-60 warlock has no Searing Pain")
	}
	searingPain := built.SearingPain[len(built.SearingPain)-1]
	if got, want := searingPain.ThreatMultiplier, 1.0; got != want {
		t.Errorf("Searing Pain threat multiplier = %v, want %v (2 x 0.5)", got, want)
	}

	brand := target.GetAuraByID(core.ActionID{SpellID: 1293696})
	if brand == nil {
		t.Fatal("no Demonic Brand aura on the target")
	}
	for i := 0; i < 20 && !brand.IsActive(); i++ {
		searingPain.ApplyEffects(sim, target, searingPain)
	}
	if !brand.IsActive() {
		t.Fatal("Searing Pain landed but no brand went up")
	}
	if got := brand.GetStacks(); got != 6 {
		t.Errorf("brand charges = %d, want 6", got)
	}

	hit := built.Succubus.GetSpell(core.ActionID{SpellID: 1293697})
	if hit == nil {
		t.Fatal("the Succubus has no Demonic Brand Shadow hit (1293697)")
	}
	for i := 0; i < 2000 && brand.IsActive() && brand.GetStacks() == 6; i++ {
		if sim.Step() {
			break
		}
	}
	if brand.IsActive() && brand.GetStacks() == 6 {
		t.Fatal("the Succubus attacked for the whole window and spent no charge")
	}
	if hit.SpellMetrics[target.UnitIndex].Casts == 0 {
		t.Error("a charge was spent but the Shadow hit was never cast")
	}
}
