package hunter

import "testing"

// The client's EffectBonusCoefficient (SpellEffect, build 1.60.1.70753) is
// 0 on every trained rank of Arcane Shot (3044, 14281-14287), Serpent
// Sting (1978, 13549-13555, 25295) and Volley's damage rows (1279715,
// 1279719, 1279721), where the casters' spells carry theirs (Frostbolt
// 25304 0.814, Shadow Bolt 25307 0.857). Only Season of Discovery's
// reissues (Serpent Sting 425728-425737, 0.08 to 0.2) state one. Spell
// power does not add to them; the engine gave them Era-convention
// coefficients (Arcane Shot 0.204 to 0.429, Serpent Sting 0.08 to 0.2 a
// tick, Volley 0.056).
func TestHunterShotsTakeNoSpellPower(t *testing.T) {
	_, hunter, _ := newBareHunterAtLevel(t, 60)
	timer := hunter.NewTimer()
	for rank := 1; rank <= 8; rank++ {
		if got := hunter.getArcaneShotConfig(rank, timer).BonusCoefficient; got != 0 {
			t.Errorf("Arcane Shot rank %d has spell power coefficient %v, want the client's 0", rank, got)
		}
	}
	for rank := 1; rank <= 9; rank++ {
		if got := hunter.getSerpentStingConfig(rank).Dot.BonusCoefficient; got != 0 {
			t.Errorf("Serpent Sting rank %d ticks with spell power coefficient %v, want the client's 0", rank, got)
		}
	}
	if hunter.Volley == nil {
		t.Fatal("a level 60 hunter has no Volley")
	}
	if got := hunter.Volley.AOEDot().BonusCoefficient; got != 0 {
		t.Errorf("Volley ticks with spell power coefficient %v, want the client's 0", got)
	}
}
