package core

import (
	"math"
	"testing"
)

// A Classic creature's melee scales with its attack power as a share of
// the whole swing, not as a flat addition: fully removing a level 63
// boss's ~270 attack power takes about 30% off its damage. Two sources
// agree: the 2020 Classic testing post "Boss Attacks and You" (median
// swing = weapon damage x (1 + 0.15873% x attack power), level 63 boss
// ~270 attack power; Broodlord 2240 +/- 448 before attack power) and
// vmangos Creature::UpdateDamagePhysical (weapon damage x (0.7 + 0.3 x
// attack power / database attack power)). With 270 attack power the two
// give the same 30% share and the same slope.
func TestEnemyMeleeFollowsClassicAttackPower(t *testing.T) {
	const minDamage = 2000.0
	weapon := Weapon{BaseDamageMin: minDamage, SwingSpeed: 2}
	roll := func(ap float64) float64 {
		sim := &Simulation{rand: NewSplitMix(7)}
		return weapon.EnemyWeaponDamage(sim, ap, 0.33)
	}
	bare := roll(0)
	full := roll(270)
	if got, want := full/bare, 1+270.0/630; math.Abs(got-want) > 1e-3 {
		t.Errorf("270 attack power scales the swing by %.4f, want %.4f (1 + AP x 0.15873%%)", got, want)
	}
	share := 1 - bare/full
	if math.Abs(share-0.30) > 0.005 {
		t.Errorf("attack power is %.1f%% of a 270 attack power boss's swing, want about 30%%", 100*share)
	}
	// Forever's Demoralizing Shout rank 5 at 60 (SpellEffect amount -196,
	// -1.4 a level from 54): 204.4 off 270 is about 23% less damage.
	shouted := roll(270 - 204.4)
	if cut := 1 - shouted/full; math.Abs(cut-0.227) > 0.005 {
		t.Errorf("Demoralizing Shout cuts the swing by %.1f%%, want about 22.7%%", 100*cut)
	}
}
