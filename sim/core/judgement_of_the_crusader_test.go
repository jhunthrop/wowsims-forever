package core

import "testing"

// Forever's Judgement of the Crusader rank 6 (20303) is aura 14 at 161
// Holy damage taken (client SpellEffect; Wowhead Forever "up to 161"),
// where Era's was 140. The raid debuff is that rank.
func TestJudgementOfTheCrusaderRaidDebuffIsTheClientsRankSix(t *testing.T) {
	if JudgementOfTheCrusaderRaidBonus != 161 {
		t.Errorf("raid Judgement of the Crusader = %v Holy damage taken, want 161", JudgementOfTheCrusaderRaidBonus)
	}
}
