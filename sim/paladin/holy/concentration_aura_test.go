package holy

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/healsim"
)

// The site's healer runs Concentration Aura (sim/request's Holy options).
// Its 35% adds to Spiritual Focus's 70%, so a hit mid-cast never costs
// the healer casting time: Holy Light cast back to back under the test
// profile's pulses lands as often as it does on a raid nothing hurts.
// Without the aura the pulses push some casts back.
func TestConcentrationAuraWithSpiritualFocusKeepsEveryCast(t *testing.T) {
	const iterations = 20
	talents := map[string]int{"spiritual_focus": 5}
	// The test profile with its pulses reaching the healer, as the site's
	// heal profile has them (three of the five, the healer among them).
	pulsed := healsim.TestProfile()
	pulsed.PulsesReachHealer = true
	casts := func(aura proto.PaladinAura, model *proto.RaidDamageModel) int32 {
		result := fight{
			rotation: castLoop(holyLightTopRank),
			talents:  talents,
			bonus:    raidGear,
			duration: 60,
			model:    model,
			aura:     aura,
		}.run(t, iterations)
		player := result.RaidMetrics.Parties[0].Players[healsim.HealerIndex]
		return healsim.CastsOf(player, &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: holyLightTopRank}})
	}
	idle := casts(proto.PaladinAura_NoPaladinAura, idleRaid())
	if got := casts(proto.PaladinAura_ConcentrationAura, pulsed); got != idle {
		t.Errorf("under Concentration Aura the pulses left %d Holy Lights of the idle raid's %d; the aura and Spiritual Focus together keep every cast", got, idle)
	}
	if got := casts(proto.PaladinAura_NoPaladinAura, pulsed); got >= idle {
		t.Errorf("without the aura the pulses left %d Holy Lights against the idle raid's %d; Spiritual Focus alone loses some", got, idle)
	}
}
