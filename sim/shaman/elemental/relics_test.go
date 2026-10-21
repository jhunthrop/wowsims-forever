package elemental

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/shaman"
)

// relicTalents is Maelstrom Weapon 5/5 and nothing else: the talent the
// Totem of the Storm test needs, and inert for the other two.
func relicTalents(t *testing.T) string {
	t.Helper()
	str, err := core.TalentsStringFromRanks((&proto.ShamanTalents{}).ProtoReflect(), shaman.TalentTreeSizes, map[string]int{"maelstrom_weapon": 5})
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func relicPlayer(t *testing.T, relicID int32) *proto.Player {
	t.Helper()
	return core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassShaman,
		Race:          proto.Race_RaceTroll,
		Level:         60,
		Equipment:     core.RelicEquipment(relicID),
		TalentsString: relicTalents(t),
	}, PlayerOptionsAdaptive)
}

func shamanSpellFingerprints(t *testing.T, relicID int32) map[string]core.SpellFingerprint {
	t.Helper()
	raid := core.SinglePlayerRaidProto(relicPlayer(t, relicID), core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)
	return core.SpellFingerprints(&env.Raid.Parties[0].Players[0].GetCharacter().Unit)
}

func changedByRelic(t *testing.T, relicID int32) []core.SpellChange {
	t.Helper()
	changed := core.ChangedSpells(shamanSpellFingerprints(t, 0), shamanSpellFingerprints(t, relicID))
	if len(changed) == 0 {
		t.Fatalf("relic %d changed no spell", relicID)
	}
	return changed
}

// Totem of Thunder (client spell 461295): Lightning Bolt crit chance +1%.
func TestTotemOfThunderRaisesLightningBoltCritOnly(t *testing.T) {
	for _, change := range changedByRelic(t, shaman.TotemOfThunder) {
		if change.After.ClassSpellMask&shaman.ShamanSpellMaskLightningBolt == 0 {
			t.Errorf("%s changed but is not Lightning Bolt: %+v", change.Key, change.After)
		}
		if got := change.After.BonusCritRating - change.Before.BonusCritRating; got != 1*core.CritRatingPerCritChance {
			t.Errorf("%s bonus crit moved by %v, want 1%%", change.Key, got)
		}
	}
}

// Burning Totem (client spell 1291077): Flame Shock lasts 3 seconds longer,
// one more tick of its 3 second period, on every rank.
func TestBurningTotemExtendsFlameShockByOneTickOnly(t *testing.T) {
	for _, change := range changedByRelic(t, shaman.BurningTotem) {
		if change.After.ClassSpellMask&shaman.ShamanSpellMaskFlameShock == 0 {
			t.Errorf("%s changed but is not Flame Shock: %+v", change.Key, change.After)
		}
		if got := change.After.DotTicks - change.Before.DotTicks; got != 1 {
			t.Errorf("%s dot ticks moved by %d, want 1", change.Key, got)
		}
	}
}

// Totem of the Storm (client spell 1291078): Lightning Bolt can trigger
// Maelstrom Weapon, at half the chance a melee hit has. The chance itself is
// pinned in sim/shaman/relics_test.go; this checks the totem arms it.
func TestTotemOfTheStormArmsLightningBoltsMaelstromWeaponChance(t *testing.T) {
	chance := func(relicID int32) float64 {
		raid := core.SinglePlayerRaidProto(relicPlayer(t, relicID), core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
		env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)
		return env.Raid.Parties[0].Players[0].(shaman.ShamanAgent).GetShaman().LightningBoltMaelstromChance
	}
	if got := chance(0); got != 0 {
		t.Errorf("without the totem Lightning Bolt's chance is %v, want 0 (it cannot trigger the talent)", got)
	}
	if got := chance(shaman.TotemOfTheStormForever); got <= 0 {
		t.Errorf("with the totem Lightning Bolt's chance is %v, want it armed", got)
	}
}
