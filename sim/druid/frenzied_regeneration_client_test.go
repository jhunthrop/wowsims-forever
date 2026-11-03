package druid

import "testing"

// Frenzied Regeneration (22842), client 1.60.1.70753 Spell.Description:
// "Converts up to 10 Rage per second into health for $d. Each point of Rage
// is converted into $s1% health.", with effect 0 (aura 23) amount 1, so one
// point of rage is 1% of maximum health. Wowhead Forever's server tooltip
// reads the same ("Each point of Rage is converted into 1% health"). The
// engine had the 0.3% of a later expansion.
func TestFrenziedRegenerationIsOnePercentAPointOfRage(t *testing.T) {
	if frenziedRegenerationHealthPerRage != 0.01 {
		t.Errorf("Frenzied Regeneration heals %v of maximum health a point of rage, want the client's 1%%", frenziedRegenerationHealthPerRage)
	}
}
