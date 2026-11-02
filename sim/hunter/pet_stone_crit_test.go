package hunter

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// survivalMainHand is Survival's band-60 main hand (1728) alone. Needs
// -tags=with_db.
func survivalMainHand() *proto.EquipmentSpec {
	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotMainHand] = &proto.ItemSpec{Id: 1728}
	return &proto.EquipmentSpec{Items: items}
}

func petCritWithImbue(t *testing.T, imbue proto.WeaponImbue) (hunterMelee, pet float64) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              60,
			Equipment:          survivalMainHand(),
			Consumes:           &proto.Consumes{MainHandImbue: imbue},
			DistanceFromTarget: 25,
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	for i := 0; i < 10 && sim.CurrentTime <= 0; i++ {
		sim.Step()
	}
	h := sim.Raid.Parties[0].Players[0].(*Hunter)
	if !h.pet.IsEnabled() {
		t.Fatal("the pet is not summoned")
	}
	target := sim.Encounter.TargetUnits[0]
	swing := h.AutoAttacks.MHAuto()
	ps := h.pet.GetStats()
	return 100 * swing.PhysicalCritChance(h.AttackTables[target.UnitIndex][swing.CastType]), ps[stats.Crit] + ps[stats.MeleeCrit]
}

// Elemental Sharpening Stone (22755, aura 52 with SpellEquippedItems mask
// 42483: melee weapons) is a temporary enchant on the hunter's own melee
// weapon. The pet does not wield it: Hunter Pet Scaling's crit channel is
// the hunter's crit chance with its ranged attacks, which the stone does
// not reach (the engine takes its 2 back off ranged). The hunter's melee
// keeps it.
func TestHunterPetDoesNotInheritTheSharpeningStone(t *testing.T) {
	baseMelee, basePet := petCritWithImbue(t, proto.WeaponImbue_WeaponImbueUnknown)
	stoneMelee, stonePet := petCritWithImbue(t, proto.WeaponImbue_ElementalSharpeningStone)
	if got := stoneMelee - baseMelee; math.Abs(got-2) > 1e-9 {
		t.Errorf("the stone adds %.2f to the hunter's melee crit, want 2", got)
	}
	if got := stonePet - basePet; math.Abs(got) > 1e-9 {
		t.Errorf("the stone adds %.2f to the pet's crit, want 0", got)
	}
}
