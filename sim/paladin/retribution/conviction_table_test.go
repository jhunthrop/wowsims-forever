package retribution

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// Conviction (20117) follows the client. Its one effect (SpellEffect
// 698173) shipped in 1.60.1.70707 as aura 52, melee and ranged crit; the
// server's hotfix push 113139 (first seen by wago.tools 2026-10-31) makes
// it aura 290, crit with every attack, spell and heal, with no class mask
// (EffectSpellClassMask 0,0,0,0) and no school (EffectMiscValue 0). Its
// SpellEquippedItems row 180908 (class 2, subclass mask 173555, the melee
// weapons) only says a melee weapon must be worn. Every beta log round 10
// read (foreverlogs.gg, 2026-09-18 to 2026-10-23) predates the push, so the
// Consecration ticks it measured were aura 52's.
func TestConvictionIsCritOnEveryTable(t *testing.T) {
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
	if got := s3 - s0; math.Abs(got-6*core.CritRatingPerCritChance) > 1e-9 {
		t.Errorf("Conviction 3/3 adds %.2f spell crit, want 6 (aura 290)", got)
	}
	if got := h3 - h0; math.Abs(got-6) > 1e-9 {
		t.Errorf("Conviction 3/3 adds %.2f%% to a heal's crit, want 6", got)
	}
}
