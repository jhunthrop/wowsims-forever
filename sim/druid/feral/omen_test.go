package feral

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Omen of Clarity (spell 16864) is a baseline trainer spell in Forever
// (SkillLineAbility: skill line 574, AcquireMethod 0, level 20), not a
// talent: every druid from level 20 has it.
func TestOmenOfClarityIsBaselineFromLevel20(t *testing.T) {
	cat, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, nil))
	if cat.ClearcastingAura == nil {
		t.Fatal("a level-60 druid with no talents has no Clearcasting aura")
	}
	low, _, _ := newFeralDruidSimWithTalents(t, 19, feralTalentsString(t, nil))
	if low.ClearcastingAura != nil {
		t.Error("a level-19 druid has Clearcasting before Omen of Clarity is learned")
	}
}

// Clearcasting (spell 16870) takes 100% off the next ability's cost and is
// spent by it.
func TestClearcastingMakesTheNextShredFree(t *testing.T) {
	cat, sim, target := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, nil))
	cat.SpendEnergy(sim, cat.CurrentEnergy(), cat.Shred.EnergyMetrics())
	sim.CurrentTime += core.GCDDefault
	cat.ClearcastingAura.Activate(sim)
	sim.CurrentTime += core.GCDDefault
	cat.GCD.Reset()

	if !cat.Shred.CanCast(sim, target) {
		t.Fatal("Shred cannot be cast at 0 energy under Clearcasting")
	}
	cat.Shred.Cast(sim, target)
	if cat.ClearcastingAura.IsActive() {
		t.Error("Clearcasting is still up after the Shred it paid for")
	}
}

// Tiger's Fury's only client row (SkillLineAbility 5217) has AcquireMethod
// 3, a Season of Discovery rune: a Forever druid cannot learn it.
func TestTigersFuryIsNotLearnable(t *testing.T) {
	cat, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, nil))
	if spell := cat.GetSpell(core.ActionID{SpellID: 5217}); spell != nil {
		t.Error("Tiger's Fury (5217) is registered, but no Forever druid can learn it")
	}
}

// Rend and Tear: "Increases damage done by your melee abilities on Bleeding
// targets by 3%" a rank on 70755 (the server's talent text; the client curve
// still reads 2%). The cat's own Rip makes the target bleed, so at 5/5 every
// Shred while Rip is up deals 15% more and the few before the first Rip deal
// the same. Damage multipliers draw no random numbers, so the same seed runs
// the same fight: the ratio sits just under 1.15, never above it.
func TestRendAndTearRaisesShredOnABleedingTarget(t *testing.T) {
	without := shredDamage(t, nil)
	with := shredDamage(t, map[string]int{"rend_and_tear": 5})
	if without == 0 {
		t.Fatal("no Shred damage in the reference run")
	}
	if got := with / without; got < 1.11 || got > 1.1501 {
		t.Errorf("Shred damage with Rend and Tear 5/5 over without = %.4f, want 1.11-1.15", got)
	}
}

func TestRendAndTearCountsTheCatsOwnBleeds(t *testing.T) {
	cat, sim, target := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, map[string]int{"rend_and_tear": 5}))
	cat.AssumeBleedActive = false
	if got := cat.RendAndTearMultiplier(target); got != 1 {
		t.Fatalf("multiplier with no bleed = %v, want 1", got)
	}
	cat.Rip.Dot(target).Apply(sim)
	if got := cat.RendAndTearMultiplier(target); got < 1.149 || got > 1.151 {
		t.Errorf("multiplier with Rip up = %v, want 1.15", got)
	}
}

// shredDamage runs a Rip-then-Shred rotation for 120 s and returns Shred's
// total damage.
func shredDamage(t *testing.T, ranks map[string]int) float64 {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassDruid,
		Race:               proto.Race_RaceTauren,
		Level:              60,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              core.FullBuffs.Player,
		TalentsString:      feralTalentsString(t, ranks),
		DistanceFromTarget: 5,
		Rotation: &proto.APLRotation{PriorityList: []*proto.APLListItem{
			castAction(rotationRipSpellID, &proto.APLValue{Value: &proto.APLValue_Not{Not: &proto.APLValueNot{
				Val: &proto.APLValue{Value: &proto.APLValue_DotIsActive{DotIsActive: &proto.APLValueDotIsActive{
					SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: rotationRipSpellID}},
				}}},
			}}}),
			castAction(rotationShredSpellID, nil),
		}},
	}, PlayerOptionsMonoCat)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 120, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}
	var damage float64
	for _, action := range result.RaidMetrics.Parties[0].Players[0].Actions {
		if action.Id.GetSpellId() != rotationShredSpellID {
			continue
		}
		for _, target := range action.Targets {
			damage += target.Damage
		}
	}
	return damage
}

// Round 9: Clearcasting is one charge ("your next damage, healing or
// offensive feral ability", ProcCharges 1). A white swing that procs it and
// a Shred cast at the same moment (the rotation acts right after the swing)
// must spend it. The old guard compared the aura's remaining time to its
// full duration, so that Shred was free and the charge paid for another.
func TestAShredAtTheProcsMomentSpendsClearcasting(t *testing.T) {
	cat, sim, target := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, nil))
	sim.CurrentTime += core.GCDDefault
	cat.ProcClearcasting(sim, cat.AutoAttacks.MHAuto())
	aura := cat.ClearcastingAura
	aura.OnCastComplete(aura, sim, cat.Shred.Spell)
	if aura.IsActive() {
		t.Error("a Shred cast at the proc's moment did not spend Clearcasting")
	}
	_ = target
}

// The cast whose hit procced Clearcasting does not spend it.
func TestTheShredThatProccedClearcastingKeepsIt(t *testing.T) {
	cat, sim, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, nil))
	sim.CurrentTime += core.GCDDefault
	cat.ProcClearcasting(sim, cat.Shred.Spell)
	aura := cat.ClearcastingAura
	aura.OnCastComplete(aura, sim, cat.Shred.Spell)
	if !aura.IsActive() {
		t.Error("the Shred that procced Clearcasting spent it")
	}
}
