package paladin

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/spellconst"
)

// Seal of Fury's absorb is the client's own spell "Light's Fury" (1310927), a
// school absorb of every school lasting 10 s, which the launch logs name on
// every Seal of Fury proc. The engine named the shield with an unused id
// before.
func TestLightsFuryMatchesTheClient(t *testing.T) {
	client, err := spellconst.Load(clientPaladinSpellconst)
	if err != nil {
		t.Fatal(err)
	}
	spell, ok := client.ByID(sealOfFuryShieldID)
	if !ok {
		t.Fatalf("spell %d is not in the client's spellconst", sealOfFuryShieldID)
	}
	if want := "Light's Fury"; spell.Name != want {
		t.Errorf("spell %d is %q, want %q", sealOfFuryShieldID, spell.Name, want)
	}
	if got, want := sealOfFuryAbsorbDuration, time.Duration(spell.DurationMS)*time.Millisecond; got != want {
		t.Errorf("duration %v, client %v", got, want)
	}
}
