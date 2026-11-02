package common

import "testing"

// Four Forever "fatal wound" procs deal more below 25% health. The client's
// Spell.csv text reads "Deals $s2% increased damage to targets below $s3%
// health" for 18107 (Gut Ripper), 10373 (Pendulum of Doom), 16549 (Blackhand
// Doomsaw) and 1305394 (Vis'kag the Bloodletter); Wowhead Forever renders
// $s2 as 100, 50, 50 and 75 and $s3 as 25 (tooltips read 2026-11-02). Round 8 modelled Gut Ripper's; the
// other three dealt their plain roll at every health.
func TestFatalWoundsHitHarderBelowAQuarter(t *testing.T) {
	want := map[int32]float64{GutRipper: 1.0, PendulumOfDoom: 0.5, BlackhandDoomsaw: 0.5, ViskagTheBloodletter: 0.75}
	for item, bonus := range want {
		if got, ok := fatalWoundExecuteBonus[item]; !ok || got != bonus {
			t.Errorf("item %d: execute bonus %v (registered %v), want %v", item, got, ok, bonus)
		}
		if got := fatalWoundDamage(100, bonus, true); got != 100*(1+bonus) {
			t.Errorf("item %d below 25%%: %v, want %v", item, got, 100*(1+bonus))
		}
		if got := fatalWoundDamage(100, bonus, false); got != 100 {
			t.Errorf("item %d above 25%%: %v, want 100", item, got)
		}
	}
}
