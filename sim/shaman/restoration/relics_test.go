package restoration

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/shaman"
)

func healingRelicFingerprints(t *testing.T, relicID int32) map[string]core.SpellFingerprint {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:     proto.Class_ClassShaman,
		Race:      proto.Race_RaceTroll,
		Level:     60,
		Equipment: core.RelicEquipment(relicID),
	}, restorationOptions)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)
	return core.SpellFingerprints(&env.Raid.Parties[0].Players[0].GetCharacter().Unit)
}

func changedByHealingRelic(t *testing.T, relicID int32) []core.SpellChange {
	t.Helper()
	changed := core.ChangedSpells(healingRelicFingerprints(t, 0), healingRelicFingerprints(t, relicID))
	if len(changed) == 0 {
		t.Fatalf("relic %d changed no spell", relicID)
	}
	return changed
}

// Totem of Urgency 279249 (client spell 1306448, aura 107 op 7, mask 0x80):
// "Increases the critical strike chance of your Lesser Healing Wave spell
// by 4%". It is an Enchanting relic on Phase 1 loot, and an unregistered
// effect leaves the gear search an empty totem slot.
func TestTotemOfUrgencyRaisesLesserHealingWaveCritOnly(t *testing.T) {
	for _, change := range changedByHealingRelic(t, shaman.TotemOfUrgency) {
		if change.After.ClassSpellMask&shaman.ShamanSpellMaskLesserHealingWave == 0 {
			t.Errorf("%s changed but is not Lesser Healing Wave: %+v", change.Key, change.After)
		}
		if got := change.After.BonusCritRating - change.Before.BonusCritRating; got != 4*core.CritRatingPerCritChance {
			t.Errorf("%s bonus crit moved by %v, want 4%%", change.Key, got)
		}
	}
}
