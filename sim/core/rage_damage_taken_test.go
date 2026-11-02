package core

import "testing"

// Rage from damage taken divides by the rage conversion value at the
// receiver's own level: Kalgan's 1.x formula ("damage taken / rage
// conversion at your level x 2.5", 230.6 at 60) and cmangos-classic's
// Player::RewardRage, which takes GetLevel() of the player gaining the
// rage. A level-60 tank hit by a level-63 boss converts at 60's value.
func TestRageFromDamageTakenUsesTheReceiversLevel(t *testing.T) {
	tank := &Unit{Level: 60}
	boss := &Unit{Level: 63}
	if got, want := damageTakenRageConversion(tank, boss), GetRageConversion(60); got != want {
		t.Errorf("rage conversion for a level-60 tank hit by a level-63 boss = %.2f, want the tank's level-60 %.2f", got, want)
	}
}
