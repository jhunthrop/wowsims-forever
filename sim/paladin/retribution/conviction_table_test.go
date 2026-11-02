package retribution

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// Conviction (20117) is aura 290 with a SpellEquippedItems row (180908:
// item class 2, subclass mask 173555, the melee weapons), the same
// weapon restriction as Cruelty. The beta logs settle what it reaches:
// across 105 foreverlogs.gg paladins at levels 19 to 35, Retribution
// paladins (5 or more points, Conviction is tier 1) crit Consecration
// 6.48% of 84,212 ticks and the rest 8.05% of 53,536, while their melee
// crits 10.1% against 6.2%. Conviction shows on the weapon and not on
// Consecration, so it is melee and ranged crit; heals keep it as their own
// bonus (too few logged Retribution heals to say otherwise).
func TestConvictionIsWeaponCritAndHealCritOnly(t *testing.T) {
	crit := func(rank int) (melee, spell, healCrit float64) {
		player := utilitySealPaladin(60, `{"type": "TypeAPL"}`)
		player.TalentsString = talentString(t, map[string]int{"conviction": rank})
		raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid:       raid,
			Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.NewDefaultTarget()}},
			SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
		}, simsignals.CreateSignals())
		sim.Reset()
		character := sim.Raid.Parties[0].Players[0].GetCharacter()
		s := character.GetStats()
		var heal *core.Spell
		for _, spell := range character.Spellbook {
			if spell.ProcMask.Matches(core.ProcMaskSpellHealing) {
				heal = spell
				break
			}
		}
		if heal == nil {
			t.Fatal("no heal registered at 60")
		}
		return s[stats.Crit] + s[stats.MeleeCrit], s[stats.Crit] + s[stats.SpellCrit], 100 * heal.HealingCritChance()
	}
	m0, s0, h0 := crit(0)
	m3, s3, h3 := crit(3)
	if got := m3 - m0; math.Abs(got-6*core.CritRatingPerCritChance) > 1e-9 {
		t.Errorf("Conviction 3/3 adds %.2f melee crit, want 6", got)
	}
	if got := s3 - s0; math.Abs(got) > 1e-9 {
		t.Errorf("Conviction 3/3 adds %.2f spell crit, want 0 (not Consecration)", got)
	}
	if got := h3 - h0; math.Abs(got-6) > 1e-9 {
		t.Errorf("Conviction 3/3 adds %.2f%% to a heal's crit, want 6", got)
	}
}
