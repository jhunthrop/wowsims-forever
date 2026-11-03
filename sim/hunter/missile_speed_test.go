package hunter

import "testing"

// Each shot travels at its client SpellMisc Speed (build 1.60.1.70753):
// Auto Shot 75, Aimed Shot 19434-20904, Arcane Shot 3044-14287 and Serpent
// Sting 1978-25295 at 40, Multi-Shot 2643 at 30, Sniper Shot 1310687-
// 1310786 at 60, Summon Hawk 1293241-1293527 at 35, and the traps (1499,
// 13795, 13813 families) at 0. The engine flew all of them at 24.
func TestHunterShotsTravelAtTheClientsSpeed(t *testing.T) {
	_, hunter, _ := newBareHunterAtLevel(t, 60)
	timer := hunter.NewTimer()
	for _, c := range []struct {
		name  string
		speed float64
		want  float64
	}{
		{"Auto Shot", hunter.AutoAttacks.RangedConfig().MissileSpeed, 40},
		{"Aimed Shot", hunter.getAimedShotConfig(6, timer).MissileSpeed, 40},
		{"Arcane Shot", hunter.getArcaneShotConfig(8, timer).MissileSpeed, 40},
		{"Serpent Sting", hunter.getSerpentStingConfig(9).MissileSpeed, 40},
		{"Multi-Shot", hunter.getMultiShotConfig(timer).MissileSpeed, 30},
		{"Sniper Shot", hunter.getSniperShotConfig(1).MissileSpeed, 60},
		{"Summon Hawk", hunter.getSummonHawkConfig(1, timer).MissileSpeed, 35},
		{"Freezing Trap", hunter.getFreezingTrapConfig(1, timer).MissileSpeed, 0},
		{"Immolation Trap", hunter.getImmolationTrapConfig(1, timer).MissileSpeed, 0},
		{"Explosive Trap", hunter.getExplosiveTrapConfig(1, timer).MissileSpeed, 0},
	} {
		if c.speed != c.want {
			t.Errorf("%s flies at %v, want the client's %v", c.name, c.speed, c.want)
		}
	}
}
