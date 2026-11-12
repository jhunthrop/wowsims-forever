package restoration

import (
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/healsim"
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

// Tidal Totem 272431 (client spell 1291076, aura 108 op 14, mask 0x40):
// "Reduces the Mana cost of your Healing Wave ability by 5%". A Pix Xizzix
// vendor relic in open content; its generated mod sat unregistered, so the
// gear search left a Restoration shaman's totem slot empty.
func TestTidalTotemCutsHealingWavesCostOnly(t *testing.T) {
	for _, change := range changedByHealingRelic(t, shaman.TidalTotem) {
		if change.After.ClassSpellMask&shaman.ShamanSpellMaskHealingWave == 0 {
			t.Errorf("%s changed but is not Healing Wave: %+v", change.Key, change.After)
		}
		if got := change.Before.CostMultiplier - change.After.CostMultiplier; got != 5 {
			t.Errorf("%s cost multiplier moved by %d points, want 5 (5%% cheaper)", change.Key, got)
		}
	}
}

// Totem of Sustaining 23200 (Scholomance): "Increases the maximum number of
// charges on your Water Shield by 1", Forever's redesign of the totem. A
// fourth globe is one more 2% of the mana pool a cast.
func TestTotemOfSustainingGivesWaterShieldAFourthGlobe(t *testing.T) {
	stacks := func(relicID int32) int32 {
		player := core.WithSpec(&proto.Player{
			Class:         proto.Class_ClassShaman,
			Race:          proto.Race_RaceTroll,
			Level:         60,
			Equipment:     core.RelicEquipment(relicID),
			TalentsString: restoTalents(map[int]int{posWaterShield: 1}),
		}, restorationOptions)
		req := healsim.Request(player, quietRaid(), 60, 1)
		sim := core.NewSim(req, simsignals.CreateSignals())
		sim.Reset()
		sim.PrePull()
		healer := sim.Raid.Parties[0].Players[healsim.HealerIndex].(*RestorationShaman)
		healer.WaterShield.Cast(sim, &healer.Unit)
		return healer.ActiveShieldAura.GetStacks()
	}
	if got := stacks(0); got != 3 {
		t.Errorf("Water Shield starts with %d globes, want 3", got)
	}
	if got := stacks(shaman.TotemOfSustaining); got != 4 {
		t.Errorf("Water Shield with Totem of Sustaining starts with %d globes, want 4", got)
	}
}
