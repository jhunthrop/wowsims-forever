package hunter

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// petCritWithScope is the summoned pet's melee crit (every-table plus melee
// crit, in percent) for a level-60 Orc hunter whose gun carries enchant.
// Needs -tags=with_db.
func petCritWithScope(t *testing.T, enchant int32) float64 {
	t.Helper()
	if _, ok := core.EnchantsByEffectID[enchant]; !ok && enchant != 0 {
		core.EnchantsByEffectID[enchant] = core.Enchant{EffectID: enchant}
	}
	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotRanged] = &proto.ItemSpec{Id: scopeTestGun, Enchant: enchant}
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{Items: items},
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
	ps := h.pet.GetStats()
	return (ps[stats.Crit] + ps[stats.MeleeCrit]) / core.CritRatingPerCritChance
}

// Blizzard, WoW Forever Beta Development Notes (16 October 2026), Pets:
// "Fixed an issue where pets were only ever inheriting the player's Melee
// Critical Strike chance instead of whichever is higher, melee or ranged."
// SAF-T Ultra Precision Scope (8720, 1310308 "Ranged Crit Bonus +02") lifts
// the hunter's ranged crit 2 above his melee crit, so the pet inherits the
// ranged figure: 2 more crit than with no scope.
func TestHunterPetInheritsTheHigherRangedCrit(t *testing.T) {
	base := petCritWithScope(t, 0)
	scoped := petCritWithScope(t, 8720)
	if got := scoped - base; math.Abs(got-2) > 1e-9 {
		t.Errorf("the scope adds %.2f to the pet's crit, want 2 (the hunter's ranged crit is the higher)", got)
	}
}
