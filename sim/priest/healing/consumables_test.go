package healing

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/healsim"
)

// TestHealerDrinksItsManaConsumables holds the healer's major-cooldown
// pass to using the mana potion and the Demonic Rune it carries: both are
// self-cast, so a friendly current target (the tank) must not stop them.
func TestHealerDrinksItsManaConsumables(t *testing.T) {
	for _, c := range rotationCases {
		t.Run(c.name, func(t *testing.T) {
			player := healer(60, c.talents, &proto.HealingPriest_Options{UseInnerFire: true}, loadRotation(t, c.file))
			player.Consumes = healsim.ManaConsumables()
			result := core.RunRaidSim(healsim.Request(player, healsim.TestProfile(), rotationSeconds, rotationIterations))
			if result.Error != nil {
				t.Fatal(result.Error.Message)
			}
			metrics := result.RaidMetrics.Parties[0].Players[healsim.HealerIndex]
			for _, name := range healsim.UnusedManaConsumables(metrics) {
				t.Errorf("the healer never used its %s", name)
			}
		})
	}
}

// innerFocusSpellID is the Inner Focus talent's spell.
const innerFocusSpellID = 14751

// TestHealerUsesInnerFocus holds the major-cooldown pass to casting Inner
// Focus for a healer whose current target is a friend: it is a self-buff.
func TestHealerUsesInnerFocus(t *testing.T) {
	player := healer(60, HolyTalents, &proto.HealingPriest_Options{UseInnerFire: true}, loadRotation(t, "forever_holy"))
	result := core.RunRaidSim(healsim.Request(player, healsim.TestProfile(), rotationSeconds, rotationIterations))
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}
	metrics := result.RaidMetrics.Parties[0].Players[healsim.HealerIndex]
	var casts int32
	for _, action := range metrics.Actions {
		if action.Id.GetSpellId() == innerFocusSpellID {
			for _, target := range action.Targets {
				casts += target.Casts
			}
		}
	}
	if casts == 0 {
		t.Error("the healer never cast Inner Focus")
	}
}
