package mage

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// The published level-60 Frost build (data/builds/1.60.1.70707/bis/
// mage-frost.json), which carries Arcane Concentration 5/5.
const clearcastingTestFrostTalents = "230005200000000000-51020000000000000-253510130101301221"

// Clearcasting (client 12536) is "your next damage spell": ProcCharges 1,
// a -1000% cost mod (misc 14) on the damage-spell class mask. The engine
// used to keep it for its whole 15 s, because the consume asked for the
// spell's current cost, which Clearcasting itself had already made zero.
func TestClearcastingIsSpentByTheNextDamageSpell(t *testing.T) {
	built, sim, target := newFrostMageSimWithTalents(t, core.DefaultTargetProtoLvl60, clearcastingTestFrostTalents)
	if built.ClearcastingAura == nil {
		t.Fatal("Clearcasting is not registered with Arcane Concentration taken")
	}
	built.ClearcastingAura.Activate(sim)
	sim.CurrentTime += time.Second

	lance := topRank(t, built.IceLance)
	before := built.CurrentMana()
	if !lance.Cast(sim, target) {
		t.Fatal("Ice Lance did not cast")
	}
	if spent := before - built.CurrentMana(); spent != 0 {
		t.Errorf("Ice Lance under Clearcasting cost %.1f mana, want 0", spent)
	}
	if built.ClearcastingAura.IsActive() {
		t.Fatal("Clearcasting is still active after the damage spell it paid for")
	}

	sim.CurrentTime += 2 * time.Second // past the global cooldown
	blast := topRank(t, built.ArcaneExplosion)
	before = built.CurrentMana()
	if !blast.Cast(sim, target) {
		t.Fatal("Arcane Explosion did not cast")
	}
	if spent := before - built.CurrentMana(); spent <= 0 {
		t.Errorf("the cast after the Clearcasting one cost %.1f mana, want its full cost", spent)
	}
}

// Blizzard's 2 November 2026 beta notes: "Clearcasting is no longer
// consumed on Arcane Missiles if Missile Barrage is active at the same
// time." The engine read the cast's cost instead, which is 0 without
// Arcane Power but 30% under it (the client adds percentage cost mods), so
// an Arcane Power Barrage Missiles spent the charge.
func TestClearcastingSurvivesABarrageMissilesUnderArcanePower(t *testing.T) {
	built, sim, target := newFrostMageSimWithTalents(t, core.DefaultTargetProtoLvl60, arcanePowerCostTestTalents)
	if built.ClearcastingAura == nil || built.MissileBarrageAura == nil || built.ArcanePowerAura == nil {
		t.Fatal("the Arcane build did not register Clearcasting, Missile Barrage and Arcane Power")
	}
	built.ArcanePowerAura.Activate(sim)
	built.MissileBarrageAura.Activate(sim)
	built.ClearcastingAura.Activate(sim)
	sim.CurrentTime += time.Second

	missiles := topRank(t, built.ArcaneMissiles)
	before := built.CurrentMana()
	if !missiles.Cast(sim, target) {
		t.Fatal("Arcane Missiles did not cast")
	}
	if spent := before - built.CurrentMana(); spent != 0 {
		t.Errorf("a Barrage Missiles under Arcane Power and Clearcasting cost %.1f mana, want 0", spent)
	}
	if built.MissileBarrageAura.IsActive() {
		t.Error("Missile Barrage was not spent by the Arcane Missiles it made free")
	}
	if !built.ClearcastingAura.IsActive() {
		t.Error("Clearcasting was spent on a Missile Barrage Arcane Missiles; Blizzard keeps it")
	}

	// Without Missile Barrage the same channel spends the charge.
	built, sim, target = newFrostMageSimWithTalents(t, core.DefaultTargetProtoLvl60, arcanePowerCostTestTalents)
	built.ArcanePowerAura.Activate(sim)
	built.ClearcastingAura.Activate(sim)
	sim.CurrentTime += time.Second
	if !topRank(t, built.ArcaneMissiles).Cast(sim, target) {
		t.Fatal("the plain Arcane Missiles did not cast")
	}
	if built.ClearcastingAura.IsActive() {
		t.Error("Clearcasting survived a plain Arcane Missiles")
	}
}

// Ice Barrier is not in Clearcasting's class mask: it is not a damage
// spell, so it pays its cost and leaves the charge for the next nuke.
func TestClearcastingDoesNotPayForIceBarrier(t *testing.T) {
	built, sim, target := newFrostMageSimWithTalents(t, core.DefaultTargetProtoLvl60, clearcastingTestFrostTalents)
	built.ClearcastingAura.Activate(sim)
	sim.CurrentTime += time.Second

	barrier := topRank(t, built.IceBarrier)
	before := built.CurrentMana()
	if !barrier.Cast(sim, target) {
		t.Fatal("Ice Barrier did not cast")
	}
	if spent := before - built.CurrentMana(); spent <= 0 {
		t.Errorf("Ice Barrier under Clearcasting cost %.1f mana, want its full cost", spent)
	}
	if !built.ClearcastingAura.IsActive() {
		t.Error("Ice Barrier spent Clearcasting; only a damage spell spends it")
	}
}

// Arcane Concentration (client 11213) has ProcCategoryRecovery 1000: after
// a proc, no hit can proc it again for 1 s.
func TestArcaneConcentrationHasAOneSecondProcCooldown(t *testing.T) {
	built, sim, target := newFrostMageSimWithTalents(t, core.DefaultTargetProtoLvl60, clearcastingTestFrostTalents)
	bolt := topRank(t, built.Frostbolt)
	result := &core.SpellResult{Outcome: core.OutcomeHit, Target: target}

	procs := 0
	for i := 0; i < 400; i++ {
		built.ClearcastingAura.Deactivate(sim)
		built.OnSpellHitDealt(sim, bolt, result)
		if built.ClearcastingAura.IsActive() {
			procs++
		}
	}
	if procs != 1 {
		t.Errorf("400 hits inside one second gave %d Clearcasting procs, want 1 (1 s proc cooldown)", procs)
	}
}

// topRank is the highest registered rank of a spell.
func topRank(t *testing.T, ranks []*core.Spell) *core.Spell {
	t.Helper()
	for i := len(ranks) - 1; i >= 0; i-- {
		if ranks[i] != nil {
			return ranks[i]
		}
	}
	t.Fatal("no rank of the spell is registered")
	return nil
}

// Missile Barrage's -100% (400589 effect 2, misc 14) adds to the other
// cost mods on Arcane Missiles rather than replacing them. It used to set
// Cost.Multiplier to 0 and back to 100, so a Barrage that came and went
// under Clearcasting left Arcane Missiles at double cost once
// Clearcasting's own -100 came off.
func TestMissileBarrageAddsToTheOtherCostMods(t *testing.T) {
	built, sim, _ := newFrostMageSimWithTalents(t, core.DefaultTargetProtoLvl60, clearcastingTestFrostTalents)
	am := topRank(t, built.ArcaneMissiles)

	built.ClearcastingAura.Activate(sim)
	built.MissileBarrageAura.Activate(sim)
	built.MissileBarrageAura.Deactivate(sim)
	built.ClearcastingAura.Deactivate(sim)
	if got := am.Cost.Multiplier; got != 100 {
		t.Errorf("Arcane Missiles Cost.Multiplier after Clearcasting and Missile Barrage both ended = %d, want 100", got)
	}
}

// Round 9 (casters lane): a charge procced by one spell's hit is spent by the
// next damage cast even when that cast is an instant completing at the
// proc's moment (a Fire Blast or Ice Lance right after Scorch lands). The
// old guard skipped every cast at the moment the aura started, so that
// instant was free and the charge paid for a second spell.
func TestAnInstantAtTheProcsMomentSpendsClearcasting(t *testing.T) {
	built, sim, target := newFrostMageSimWithTalents(t, core.DefaultTargetProtoLvl60, clearcastingTestFrostTalents)
	built.ProcClearcasting(sim, topRank(t, built.Frostbolt))
	lance := topRank(t, built.IceLance)
	if !lance.Cast(sim, target) {
		t.Fatal("Ice Lance did not cast")
	}
	if built.ClearcastingAura.IsActive() {
		t.Error("an Ice Lance at the proc's moment did not spend Clearcasting")
	}
}

// Round 11 (casters M1): the mage's Clearcasting is -1000% cost (12536
// effect 0, SpellEffect 689319: aura 108 misc 14, base points -1000, in the
// shipped table and the hotfix push), where the druid's and shaman's are
// -100. Percent cost mods add, so at -100 an Arcane Blast at four stacks
// (+400%) still cost 400% of its price under Clearcasting; at -1000 it is
// free, as the next damage spell should be.
func TestClearcastingMakesAStackedArcaneBlastFree(t *testing.T) {
	sim, built := newArcaneBlastTestMage(t)
	if built.ClearcastingAura == nil {
		t.Fatal("Clearcasting is not registered with Arcane Concentration taken")
	}
	arcaneBlast := built.ArcaneBlast[ArcaneBlastRanks]
	target := sim.Encounter.TargetUnits[0]
	for cast := int32(1); cast <= arcaneBlastMaxStacks; cast++ {
		arcaneBlast.ApplyEffects(sim, target, arcaneBlast)
	}
	built.ClearcastingAura.Activate(sim)
	if got := arcaneBlast.Cost.GetCurrentCost(); got != 0 {
		t.Errorf("Arcane Blast at %d stacks under Clearcasting costs %.1f mana, want 0", arcaneBlastMaxStacks, got)
	}
}
