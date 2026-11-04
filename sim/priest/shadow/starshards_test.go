package shadow

import (
	"testing"
	"time"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/priest"
)

// Starshards (Night Elf priests) is on a 30 second cooldown shared by every
// rank: SpellCooldowns CategoryRecoveryTime 30000 on every rank (10797,
// 19296 to 19305; category 2522), and the Forever tooltip "30 sec cooldown"
// (https://nether.wowhead.com/forever/tooltip/spell/19305). The engine had
// none, so a line casting it would chain it back to back.
func TestStarshardsIsOnAThirtySecondCooldownSharedByItsRanks(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassPriest,
		Race:          proto.Race_RaceNightElf,
		Level:         60,
		Equipment:     &proto.EquipmentSpec{},
		Buffs:         core.FullBuffs.Player,
		TalentsString: P1Talents,
	}, PlayerOptionsBasic)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	encounter := &proto.Encounter{Duration: 10, Targets: []*proto.Target{core.NewDefaultTarget()}}
	env, _, _ := core.NewEnvironment(raid, encounter, true)
	spriest, ok := env.Raid.Parties[0].Players[0].(*ShadowPriest)
	if !ok {
		t.Fatal("not a shadow priest")
	}
	top := spriest.Starshards[priest.StarshardsRanks][0]
	if top == nil {
		t.Fatal("a level-60 Night Elf has no Starshards rank 7")
	}
	if got := top.CD.Duration; got != 30*time.Second {
		t.Errorf("Starshards cooldown %v, want 30s", got)
	}
	for rank := 1; rank <= priest.StarshardsRanks; rank++ {
		for _, spell := range spriest.Starshards[rank] {
			if spell != nil && spell.CD.Timer != top.CD.Timer {
				t.Errorf("Starshards rank %d (%v) does not share the top rank's cooldown", rank, spell.ActionID)
			}
		}
	}
}
