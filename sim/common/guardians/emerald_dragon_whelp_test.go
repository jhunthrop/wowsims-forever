package guardians

import (
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
)

// The whelp's sheet states melee crit 4.5% and spell crit 13% (wowsims'
// figures); the round-9 split keeps each on its own table, so its melee
// does not crit at its Acid Spit's 13%.
func TestEmeraldDragonWhelpKeepsItsCritTablesApart(t *testing.T) {
	s := emeraldWhelpBaseStats()
	if s[stats.Crit] != 0 || s[stats.MeleeCrit] != 4.5 || s[stats.SpellCrit] != 13 {
		t.Errorf("whelp crit = every table %.1f, melee %.1f, spell %.1f; want 0, 4.5, 13", s[stats.Crit], s[stats.MeleeCrit], s[stats.SpellCrit])
	}
}
