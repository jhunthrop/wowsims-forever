package restoration

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Healing Stream's ticks crit on the shaman's every-table crit only (beta
// logs: 0 crits in 21,134 ticks for shamans with no aura-290 talent). A
// "Nature spells" crit bonus (The Ten Storms 5-piece, aura 57) is spell
// crit, so it stays off the totem's ticks as the Intellect crit does.
func TestHealingStreamIgnoresANatureSchoolSpellCrit(t *testing.T) {
	sim, healer := newHealer(t, 60, "")
	shaman := healer.GetShaman()
	heal := shaman.GetSpell(core.ActionID{SpellID: healingStreamTopHealID})
	if heal == nil {
		t.Fatal("no Healing Stream heal spell registered")
	}
	shaman.AddStatDynamic(sim, stats.SpellCrit, -shaman.GetStat(stats.SpellCrit))
	shaman.AddStatDynamic(sim, stats.Crit, -shaman.GetStat(stats.Crit))
	shaman.PseudoStats.SchoolBonusCritChance[stats.SchoolIndexNature] += 100 * core.CritRatingPerCritChance

	target := raidMember(sim, 1)
	for i := 0; i < 50; i++ {
		heal.ApplyEffects(sim, target, heal)
	}
	var crits int32
	for _, m := range heal.SpellMetrics {
		crits += m.Crits + m.CritTicks
	}
	if crits != 0 {
		t.Errorf("with 100%% Nature spell crit and no every-table crit, %d of 50 ticks crit, want 0", crits)
	}
}
