package paladin

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/spellconst"
)

// Templar's Bulwark (1311015) is a Protection talent on the 10-point row. Client
// 1.60.1.70707 shipped it with SpellLevels 30, and hotfix push 113139 set that row to 0;
// 1.60.1.70753 ships 0. With no level of its own, a paladin uses it as soon as the talent
// is taken (level 19 at the earliest), so the engine must not gate it at 30.
func TestTemplarsBulwarkMatchesTheClient(t *testing.T) {
	client, err := spellconst.Load(clientPaladinSpellconst)
	if err != nil {
		t.Fatal(err)
	}
	spell, ok := client.ByID(templarsBulwarkActionID)
	if !ok {
		t.Fatalf("spell %d is not in the client's spellconst", templarsBulwarkActionID)
	}
	if got, want := templarsBulwarkLevel, spell.SpellLevel; int(got) != want {
		t.Errorf("required level %d, client %d", got, want)
	}
	if got, want := float64(templarsBulwarkManaCost), spell.Cost; got != want {
		t.Errorf("mana cost %v, client %v", got, want)
	}
	if got, want := templarsBulwarkDuration, time.Duration(spell.DurationMS)*time.Millisecond; got != want {
		t.Errorf("duration %v, client %v", got, want)
	}
}
