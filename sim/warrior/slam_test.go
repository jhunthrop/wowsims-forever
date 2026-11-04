package warrior

import (
	"testing"
	"time"
)

// Improved Slam, the client's trait curves on 12862 (1.60.1.70755): the
// cast time (e0, misc 10) and global cooldown (e1, misc 21) lose 250/500
// ms and the cooldown (e2, misc 11) 2.5/5 s at ranks 1/2 ("Reduces the
// cooldown of your Slam ability by 2.5/5.0 sec"; a flat 3 s at either
// rank before). With any point "Slam no longer interrupts or delays your
// melee swing".
func TestImprovedSlamReductionsPerPoint(t *testing.T) {
	cases := []struct {
		points   int32
		cast     time.Duration
		cooldown time.Duration
	}{
		{0, 0, 0},
		{1, 250 * time.Millisecond, 2500 * time.Millisecond},
		{2, 500 * time.Millisecond, 5 * time.Second},
	}
	for _, c := range cases {
		cast, gcd, cooldown := improvedSlamReductions(c.points)
		if cast != c.cast || gcd != c.cast || cooldown != c.cooldown {
			t.Errorf("points %d: reductions = (cast %s, gcd %s, cooldown %s), want (%s, %s, %s)",
				c.points, cast, gcd, cooldown, c.cast, c.cast, c.cooldown)
		}
	}
}

// The max rank's reductions are the values the client row 12862 itself
// carries (the trait curve's last point is the row's base points).
func TestImprovedSlamMaxRankIsTheClientRow(t *testing.T) {
	row, ok := clientWarrior(t).ByID(TalentSpellIDs["improved_slam"][0])
	if !ok {
		t.Fatal("Improved Slam 12862 is not in the client table")
	}
	cast, gcd, cooldown := improvedSlamReductions(int32(len(TalentSpellIDs["improved_slam"])))
	byMisc := map[int32]time.Duration{10: cast, 21: gcd, 11: cooldown}
	for _, e := range row.Effects {
		want, ok := byMisc[e.MiscValue]
		if e.Aura != clientAuraAddFlatModifier || !ok {
			continue
		}
		if got := -time.Duration(e.Amount) * time.Millisecond; got != want {
			t.Errorf("12862 e%d (misc %d) cuts %s, the engine %s", e.Index, e.MiscValue, got, want)
		}
		delete(byMisc, e.MiscValue)
	}
	if len(byMisc) != 0 {
		t.Errorf("12862 has no aura 107 effect for misc %v", byMisc)
	}
}

func TestImprovedSlamKeepsTheSwingWithAnyPoint(t *testing.T) {
	if improvedSlamKeepsTheSwing(0) {
		t.Error("untalented Slam must still stop the swing timer for its cast")
	}
	for _, points := range []int32{1, 2} {
		if !improvedSlamKeepsTheSwing(points) {
			t.Errorf("%d point(s): Slam must no longer delay the melee swing", points)
		}
	}
}

// Every learned Slam rank's category cooldown is the client's: 15000 ms
// since 1.60.1.70755 (18000 before), and every rank's cost and cast time.
func TestSlamRanksMatchTheClient(t *testing.T) {
	client := clientWarrior(t)
	for rank := 1; rank <= SlamRanks; rank++ {
		row, ok := client.ByID(slamRankSpellID[rank])
		if !ok {
			t.Fatalf("Slam rank %d (%d) is not in the client table", rank, slamRankSpellID[rank])
		}
		if got, want := slamRankCooldownMS[rank], row.CategoryCooldownMS; got != want {
			t.Errorf("Slam rank %d cooldown %d ms, client %d", rank, got, want)
		}
		if got, want := slamRankCastTimeMS[rank], row.CastTimeMS; got != want {
			t.Errorf("Slam rank %d cast %d ms, client %d", rank, got, want)
		}
		if got, want := slamRankManaCost[rank], row.Cost; got != want {
			t.Errorf("Slam rank %d cost %v tenths, client %v", rank, got, want)
		}
	}
	if slamCooldownMS != 15000 {
		t.Errorf("Slam's cooldown is %d ms, want 15000 (1.60.1.70755)", slamCooldownMS)
	}
}
