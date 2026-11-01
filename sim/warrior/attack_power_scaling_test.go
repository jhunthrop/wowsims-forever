package warrior

import "testing"

// Client 1.60.1.70707 rewrote every Revenge rank's description to
// "${$m1+($<APRatio>*$apm)} to ${$M1+(...)}" with APRatio 0.25; the
// Forever tooltips read, rank 6 (25288), 137.7-168.3 + 0.25 x AP and rank
// 5 (11601) 109.07-132.93 + 0.25 x AP, and wowfdb logged it on 30 Oct.
// The AP coefficient lives only in the description variable, not in
// SpellEffect, so the engine adds it by hand.
func TestRevengeAddsAQuarterOfAttackPower(t *testing.T) {
	if got := revengeBaseDamage(121, 0, 0); got != 121 {
		t.Errorf("Revenge at 0 AP = %.1f, want the client roll 121", got)
	}
	if got := revengeBaseDamage(121, 75, 1200); got != 121+75+300 {
		t.Errorf("Revenge at 1200 AP with Dreadnaught 2 = %.1f, want %d", got, 121+75+300)
	}
}

// Blizzard's 24 Sep notes corrected Sunder Armor's threat on every rank
// "including a small increase to threat generated from Attack Power";
// sixtysim's beta measurement puts it at 5% of attack power on top of the
// client's flat amount (206 at rank 5).
func TestSunderArmorThreatGrowsWithAttackPower(t *testing.T) {
	if got := sunderArmorFlatThreat(5, 0); got != 206 {
		t.Errorf("Sunder rank 5 at 0 AP = %.1f, want 206", got)
	}
	if got := sunderArmorFlatThreat(5, 1200); got != 206+60 {
		t.Errorf("Sunder rank 5 at 1200 AP = %.1f, want 266", got)
	}
}
