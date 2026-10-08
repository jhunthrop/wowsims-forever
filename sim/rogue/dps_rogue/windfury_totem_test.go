package dpsrogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	googleproto "google.golang.org/protobuf/proto"
)

// The 1.60.1.70009 client's Windfury Totem (spells 8512 to 10614) is an aura
// on every party member, not a weapon enchant: the rogue's poisons sit on
// the weapon and the totem's extra attacks come on top. The raid buff
// WindfuryTotem is how a request states that.
func raidBuffsWithWindfuryTotem() *proto.RaidBuffs {
	buffs := googleproto.Clone(core.FullBuffs.Raid).(*proto.RaidBuffs)
	buffs.WindfuryTotem = true
	return buffs
}

func TestWindfuryTotemRaidBuffGivesTheRogueTheExtraAttackAura(t *testing.T) {
	_, without := buildRogueWithRaidBuffs(t, "", core.FullBuffs.Raid)
	if without.GetAura("Windfury") != nil {
		t.Fatal("a rogue without the Windfury Totem buff has a Windfury aura")
	}
	_, with := buildRogueWithRaidBuffs(t, "", raidBuffsWithWindfuryTotem())
	if with.GetAura("Windfury") == nil {
		t.Fatal("the Windfury Totem raid buff did not register the Windfury aura on the rogue")
	}
}
