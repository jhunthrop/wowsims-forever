package warlock

import "testing"

// Bane of Doom is learned as id 603 (the only learn row); its damage is that
// client row's, 1742 at coefficient 4.0, not the non-learnable 449432 row's
// 3200 at 1.0.
const (
	baneOfDoomClientDamage      = 1742.0
	baneOfDoomClientCoefficient = 4.0
)

func TestBaneOfDoomDeclaresTheLearnableClientRow(t *testing.T) {
	_, built, _ := newBareWarlockForDamageTest(t)
	spell := built.CurseOfDoom
	if spell == nil || spell.ActionID.SpellID != 603 {
		t.Fatal("level-60 warlock has no Bane of Doom under id 603")
	}
	want := [2]float64{baneOfDoomClientDamage, baneOfDoomClientDamage}
	if spell.ClientBaseDamage != want {
		t.Errorf("ClientBaseDamage = %v, want %v", spell.ClientBaseDamage, want)
	}
	if spell.BonusCoefficient != baneOfDoomClientCoefficient {
		t.Errorf("spell coefficient = %v, want %v", spell.BonusCoefficient, baneOfDoomClientCoefficient)
	}
}

func TestBaneOfDoomSnapshotsTheClientDamageAndCoefficient(t *testing.T) {
	for attempt := 0; attempt < landAttempts; attempt++ {
		sim, built, target := newBareWarlockForDamageTest(t)
		spell := built.CurseOfDoom
		spell.ApplyEffects(sim, target, spell)
		dot := spell.Dot(target)
		if !dot.IsActive() {
			continue
		}
		if dot.SnapshotBaseDamage != baneOfDoomClientDamage {
			t.Errorf("snapshot base damage = %v, want %v", dot.SnapshotBaseDamage, baneOfDoomClientDamage)
		}
		if dot.BonusCoefficient != baneOfDoomClientCoefficient {
			t.Errorf("dot coefficient = %v, want %v", dot.BonusCoefficient, baneOfDoomClientCoefficient)
		}
		return
	}
	t.Fatal("Bane of Doom never landed")
}
