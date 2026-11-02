package restoration

import "testing"

// Innervate is learned at level 40 (SpellLevels 29166: SpellLevel 40;
// SkillLineAbility 573 trainer row), so a druid below 40 has none to cast.
func TestInnervateIsLearnedAtForty(t *testing.T) {
	for _, c := range []struct {
		level int32
		has   bool
	}{{20, false}, {39, false}, {40, true}, {60, true}} {
		resto, _ := newDruid(t, newPlayer(c.level, "", healerBonusStats, nil))
		if got := resto.Innervate != nil; got != c.has {
			t.Errorf("a level %d druid has Innervate = %v, want %v", c.level, got, c.has)
		}
	}
}
