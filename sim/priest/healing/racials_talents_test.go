package healing

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// The priest racial heals carry class masks the talents name (client
// 1.60.1.70707 SpellClassOptions): Divine Grace (1277370-1277378) is
// 0x1000000/0/0/0, named by Holy Specialization eff0, Twin Disciplines eff0
// and Spiritual Healing eff0; Contingency Plan's heal over time
// (1277456-1277460) is 0x20/0/0/0, named by Spiritual Healing eff0 and both
// Twin Disciplines effects. Discipline's published build (human) and Holy's
// (gnome) take all three talents.
const racialTalents = "025005031330011530-03505003030000000"

func TestTheTalentsReachThePriestRacialHeals(t *testing.T) {
	_, plain := racialPriest(t, proto.Race_RaceHuman, 60)
	sim, talented := agentSim(t, func() *proto.Player {
		p := healer(60, racialTalents, &proto.HealingPriest_Options{}, &proto.APLRotation{})
		p.Race = proto.Race_RaceHuman
		return p
	}())
	_ = sim
	if got, base := talented.DivineGrace.DamageMultiplier, plain.DivineGrace.DamageMultiplier; got <= base {
		t.Errorf("Divine Grace's multiplier is %.3f with Twin Disciplines and Spiritual Healing, %.3f without", got, base)
	}
	if got, base := talented.DivineGrace.BonusCritRating, plain.DivineGrace.BonusCritRating; got <= base {
		t.Errorf("Divine Grace's crit rating is %.1f with Holy Specialization, %.1f without", got, base)
	}

	_, plainGnome := racialPriest(t, proto.Race_RaceGnome, 60)
	_, gnome := agentSim(t, func() *proto.Player {
		p := healer(60, racialTalents, &proto.HealingPriest_Options{}, &proto.APLRotation{})
		p.Race = proto.Race_RaceGnome
		return p
	}())
	if got, base := gnome.ContingencyHot.DamageMultiplier, plainGnome.ContingencyHot.DamageMultiplier; got <= base {
		t.Errorf("Contingency Plan's heal over time multiplier is %.3f with Twin Disciplines and Spiritual Healing, %.3f without", got, base)
	}
}
