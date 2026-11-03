package priest

import "testing"

// The client's Vampiric Embrace (15286) states the party's share of the
// priest's Shadow damage as its first effect's amount, in percent.
func TestVampiricEmbraceHealsThePartyTheClientsShare(t *testing.T) {
	spell, ok := loadClient(t).ByID(15286)
	if !ok || len(spell.Effects) == 0 {
		t.Fatal("the client states no Vampiric Embrace 15286 effect")
	}
	if want := spell.Effects[0].Amount / 100; vampiricEmbraceHealShare != want {
		t.Errorf("Vampiric Embrace heals %v of Shadow damage, the client states %v", vampiricEmbraceHealShare, want)
	}
}
