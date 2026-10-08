package healing

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/healsim"
)

// The cooldowns the major-cooldown pass is held to for a healer whose
// current target is a friend.
const (
	berserkingSpellID           int32 = 26297
	talismanOfEphemeralPowerID  int32 = 18820 // use: spell power
	earthstrikeID               int32 = 21180 // use: attack power only
	cooldownRunIterationsPerRun       = 10
)

func cooldownRun(t *testing.T, race proto.Race, trinket int32) *proto.UnitMetrics {
	t.Helper()
	player := healer(60, HolyTalents, &proto.HealingPriest_Options{UseInnerFire: true}, loadRotation(t, "forever_holy"))
	player.Race = race
	if trinket != 0 {
		player.Equipment = &proto.EquipmentSpec{Items: []*proto.ItemSpec{{Id: trinket}}}
	}
	result := core.RunRaidSim(healsim.Request(player, healsim.TestProfile(), rotationSeconds, cooldownRunIterationsPerRun))
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}
	return result.RaidMetrics.Parties[0].Players[healsim.HealerIndex]
}

// TestHealerFiresItsRacial holds the pass to casting Berserking, a haste
// self-buff, with a friend as the current target.
func TestHealerFiresItsRacial(t *testing.T) {
	metrics := cooldownRun(t, proto.Race_RaceTroll, 0)
	if healsim.CastsOf(metrics, &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: berserkingSpellID}}) == 0 {
		t.Error("the troll healer never used Berserking")
	}
}

// TestHealerFiresASpellPowerTrinket holds the pass to using a spell power
// trinket, and to leaving an attack-power-only trinket in the bag.
func TestHealerFiresASpellPowerTrinket(t *testing.T) {
	use := func(id int32) int32 {
		metrics := cooldownRun(t, proto.Race_RaceHuman, id)
		return healsim.CastsOf(metrics, &proto.ActionID{RawId: &proto.ActionID_ItemId{ItemId: id}})
	}
	if use(talismanOfEphemeralPowerID) == 0 {
		t.Error("the healer never used its spell power trinket")
	}
	if got := use(earthstrikeID); got != 0 {
		t.Errorf("the healer used an attack-power-only trinket %d times", got)
	}
}
