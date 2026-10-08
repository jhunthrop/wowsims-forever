package paladin

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
	"github.com/wowsims/classic/sim/core/proto"
)

// Benediction is 2% a rank in the client, not the 3% the engine once
// applied: the client's single spell row carries the full-rank total.
func TestBenedictionMatchesClient(t *testing.T) {
	client := clientdamagetest.Load(t, clientPaladinSpellconst)
	row, ok := client.ByID(benedictionClientSpellID)
	if !ok {
		t.Fatalf("Benediction %d is not in the client's spellconst", benedictionClientSpellID)
	}
	want := -row.Effects[0].Amount
	if got := float64(benedictionCostReductionPerRank * benedictionMaxRanks); got != want {
		t.Errorf("full-rank cost reduction %v%%, client %v%%", got, want)
	}
}

func TestBenedictionCostByRank(t *testing.T) {
	for rank, want := range []int32{100, 98, 96, 94, 92, 90} {
		paladin := &Paladin{Talents: &proto.PaladinTalents{Benediction: int32(rank)}}
		if got := paladin.benediction(); got != want {
			t.Errorf("rank %d: instant spells cost %d%%, want %d%%", rank, got, want)
		}
	}
}
