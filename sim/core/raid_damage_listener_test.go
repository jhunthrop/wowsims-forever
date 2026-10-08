package core

import "testing"

func TestRaidDamageListenersHearEveryHitInOrder(t *testing.T) {
	raid := &Raid{}
	var heard []float64
	raid.OnRaidDamage(func(_ *Simulation, _ *Unit, damage float64, isTank bool) {
		if isTank {
			damage = -damage
		}
		heard = append(heard, damage)
	})
	raid.OnRaidDamage(func(_ *Simulation, _ *Unit, damage float64, _ bool) { heard = append(heard, damage*10) })

	raid.notifyRaidDamage(nil, nil, 5, false)
	raid.notifyRaidDamage(nil, nil, 7, true)

	want := []float64{5, 50, -7, 70}
	if len(heard) != len(want) {
		t.Fatalf("heard %v, want %v", heard, want)
	}
	for i := range want {
		if heard[i] != want[i] {
			t.Errorf("heard %v, want %v", heard, want)
			break
		}
	}
}
