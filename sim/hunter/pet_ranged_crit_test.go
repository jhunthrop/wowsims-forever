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
	return petAndHunterCrit(t, enchant, &proto.RaidBuffs{}).pet
}

// critPair is a summoned pet's melee crit and its hunter's, in percent.
type critPair struct{ pet, hunter float64 }

// petAndHunterCrit is the summoned pet's melee crit and its level-60 Orc
// hunter's (every-table plus melee crit, in percent) under raidBuffs, with
// enchant on the hunter's gun. Needs -tags=with_db.
func petAndHunterCrit(t *testing.T, enchant int32, raidBuffs *proto.RaidBuffs) critPair {
	t.Helper()
	_, h := startHunterWithPet(t, enchant, raidBuffs)
	return critsOf(h)
}

// critsOf reads the pet's and the hunter's every-table plus melee crit, in
// percent.
func critsOf(h *Hunter) critPair {
	ps, hs := h.pet.GetStats(), h.GetStats()
	return critPair{
		pet:    (ps[stats.Crit] + ps[stats.MeleeCrit]) / core.CritRatingPerCritChance,
		hunter: (hs[stats.Crit] + hs[stats.MeleeCrit]) / core.CritRatingPerCritChance,
	}
}

// startHunterWithPet runs a level-60 Orc hunter's sim to the pull with his
// pet summoned, under raidBuffs and with enchant on his gun. Needs
// -tags=with_db.
func startHunterWithPet(t *testing.T, enchant int32, raidBuffs *proto.RaidBuffs) (*core.Simulation, *Hunter) {
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
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, raidBuffs, &proto.Debuffs{})
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
	return sim, h
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

// The raid's druid crit aura reaches the pet once, through the hunter.
// Leader of the Pack (24932) and Moonkin Aura (24907) are flagged "can only
// target players" in the 70790 client (SpellMisc Attributes_3 0x100), so
// neither lands on the pet itself; the pet inherits the hunter's crit
// (Hunter Pet Scaling 415429, Blizzard 16 October 2026), which carries the
// aura's 3%. A raid with both druids adds 3 to the hunter and 3 to the pet,
// never 6 to the pet.
func TestHunterPetTakesTheRaidCritAuraOnceThroughTheHunter(t *testing.T) {
	bare := petAndHunterCrit(t, 0, &proto.RaidBuffs{})
	both := petAndHunterCrit(t, 0, &proto.RaidBuffs{LeaderOfThePack: true, MoonkinAura: true})
	if got := both.hunter - bare.hunter; math.Abs(got-3) > 1e-9 {
		t.Errorf("the druid auras add %.2f to the hunter's crit, want 3", got)
	}
	if got := both.pet - bare.pet; math.Abs(got-3) > 1e-9 {
		t.Errorf("the druid auras add %.2f to the pet's crit, want 3 (inherited once, never applied to the pet as well)", got)
	}
}

// The scope's ranged-only crit reaches the pet once, at the summon. A
// change to the hunter's stats in the fight (a crit buff here) moves the
// pet's crit by that change alone, never by the scope again.
func TestHunterPetTakesTheScopeOnceThroughTheFight(t *testing.T) {
	sim, h := startHunterWithPet(t, 8720, &proto.RaidBuffs{})
	before := critsOf(h)
	for i := 0; i < 3; i++ {
		h.AddStatsDynamic(sim, stats.Stats{stats.MeleeCrit: core.CritRatingPerCritChance})
	}
	after := critsOf(h)
	if got := after.hunter - before.hunter; math.Abs(got-3) > 1e-9 {
		t.Fatalf("the buffs add %.2f to the hunter's crit, want 3", got)
	}
	if got := after.pet - before.pet; math.Abs(got-3) > 1e-9 {
		t.Errorf("the hunter's 3 crit moves the pet's by %.2f, want 3 (the scope is inherited once, at the summon)", got)
	}
}
