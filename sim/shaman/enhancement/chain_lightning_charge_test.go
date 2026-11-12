package enhancement

import (
	"math"
	"testing"
)

// Chain Lightning's bounce loop scales the spell's DamageMultiplier down
// a bounce at a time and put the entering value back after the hits were
// dealt. Dealing the first hit spends the Stormstrike charge, whose
// dynamic mod divides the same multiplier by 1.2 as it leaves, so the
// restore reinstated the spent charge's 1.2 on the spell for good: every
// Stormstrike then Chain Lightning pair compounded it (the rotation search
// found "Chain Lightning" worth 1e40 DPS for an Enhancement shaman). The
// multiplier must be restored before the hits are dealt, so a mod that
// leaves on the hit stays gone.
func TestChainLightningKeepsNoSpentStormstrikeCharge(t *testing.T) {
	sim, built := newIsolatedTalentShaman(t, shamanTalentString(nil, map[int]int{enhNodeStormstrike: 1}, nil))
	target := sim.Encounter.TargetUnits[0]
	chain := built.ChainLightning[len(built.ChainLightning)-1]
	if chain == nil {
		t.Fatal("no Chain Lightning at 60")
	}
	// Before the partial resist roll, which is the target's, not the charge's.
	damage := func() float64 {
		result := chain.CalcDamage(sim, target, 1000, chain.OutcomeAlwaysHit)
		return result.Damage / result.ResistanceMultiplier
	}

	plain := damage()
	before := chain.DamageMultiplier
	// Stormstrike's own roll can miss or be dodged; land it until the charge
	// is up.
	stormstrikeUntilCharged := func(round int) {
		for try := 0; try < 100; try++ {
			built.Stormstrike.ApplyEffects(sim, target, built.Stormstrike)
			if math.Abs(damage()/plain-1.2) < 1e-6 {
				return
			}
		}
		t.Fatalf("round %d: no Stormstrike landed in 100 tries", round)
	}
	for round := 1; round <= 3; round++ {
		stormstrikeUntilCharged(round)
		chain.ApplyEffects(sim, target, chain)
		if got := chain.DamageMultiplier; math.Abs(got-before) > 1e-9 {
			t.Fatalf("round %d: after Chain Lightning spent the charge its DamageMultiplier is %v, want %v (the entering value without the charge)", round, got, before)
		}
		if got := damage() / plain; math.Abs(got-1) > 1e-6 {
			t.Fatalf("round %d: after one Chain Lightning the next deals %v of its damage, want 1 (one charge)", round, got)
		}
	}
}
