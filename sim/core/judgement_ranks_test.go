package core

import (
	"testing"
	"time"
)

// Judgement of Wisdom and Judgement of Light from the 1.60.1.70009 client
// (data/builds/1.60.1.70009/raw):
//
//	Judgement of Wisdom 20186 / 20354 / 20355 (levels 38 / 48 / 58, 40 s):
//	  aura 42 (proc trigger), ProcChance 100, ProcTypeMask_0 139944 (every
//	  melee, ranged and spell attack taken); the mana is spells 20268 /
//	  20352 / 20353, effect 30 (energize), base points 33 / 46 / 59.
//	Judgement of Light 20185 / 20344 / 20345 / 20346 (levels 30 / 40 / 50 /
//	  60, 40 s): aura 42, ProcTypeMask_0 40 (melee taken); the heal is spells
//	  20267 / 20341 / 20342 / 20343, effect 10 (heal), base points 25 / 34 /
//	  49 / 61, no coefficient.
func TestJudgementOfWisdomRanksMatchClient(t *testing.T) {
	want := BuffRanks{
		{SpellID: 20186, Level: 38, Amount: 33},
		{SpellID: 20354, Level: 48, Amount: 46},
		{SpellID: 20355, Level: 58, Amount: 59},
	}
	assertRanks(t, "Judgement of Wisdom", JudgementOfWisdomRanks, want)
}

func TestJudgementOfLightRanksMatchClient(t *testing.T) {
	want := BuffRanks{
		{SpellID: 20185, Level: 30, Amount: 25},
		{SpellID: 20344, Level: 40, Amount: 34},
		{SpellID: 20345, Level: 50, Amount: 49},
		{SpellID: 20346, Level: 60, Amount: 61},
	}
	assertRanks(t, "Judgement of Light", JudgementOfLightRanks, want)
}

func assertRanks(t *testing.T, name string, got, want BuffRanks) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s has %d ranks, want %d", name, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s rank %d = %+v, want %+v", name, i+1, got[i], want[i])
		}
	}
}

// Every judgement lasts 40 seconds in the client (duration index 31 on
// spells 20185 to 20355), where the engine had kept vanilla's 10.
func TestJudgementsLastFortySeconds(t *testing.T) {
	if JudgementDuration != 40*time.Second {
		t.Errorf("judgement duration = %v, want 40s", JudgementDuration)
	}
}
