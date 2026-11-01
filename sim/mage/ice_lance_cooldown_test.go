package mage

import (
	"testing"
	"time"
)

// Client 1.60.1.70685 moved Ice Lance rank 1 (1312002) into Fire Blast's
// cooldown category (SpellCategories Category 19, the one every Fire Blast
// rank carries) with Fire Blast's 8 s category recovery. The trained ranks
// (400640, 1240044-1240047) have no row of their own in the shipped
// tables, but Wowhead Forever's hotfix-applied tooltips show "8 sec
// cooldown" on every rank, the same data that gives Ice Lance its 300%
// Frozen bonus over the shipped 200. Every rank therefore waits 8 s, on
// the category timer it shares with Fire Blast.
func TestEveryIceLanceRankHasFireBlastsEightSecondCategoryCooldown(t *testing.T) {
	built, sim, boss := newFrostMageSimWithTalents(t, bossTarget(), ForeverFrostTalents)
	for rank := 1; rank <= IceLanceRanks; rank++ {
		lance := built.IceLance[rank]
		if lance == nil {
			t.Fatalf("rank %d is not registered at level 60", rank)
		}
		if got := lance.CD.Duration; got != 8*time.Second {
			t.Errorf("rank %d (%d): cooldown %v, want 8s", rank, IceLanceSpellId[rank], got)
		}
	}
	lance := built.IceLance[IceLanceRanks]
	fireBlast := built.FireBlast[FireBlastRanks]
	if lance.CD.Timer != fireBlast.CD.Timer {
		t.Fatal("Ice Lance and Fire Blast keep separate cooldown timers; client category 19 shares one")
	}
	fireBlast.CD.Use(sim)
	if lance.CD.IsReady(sim) {
		t.Error("Ice Lance is ready right after a Fire Blast; the shared category cooldown should hold it")
	}
	_ = boss
}
