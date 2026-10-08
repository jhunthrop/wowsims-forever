package restoration

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/healsim"
)

const (
	rotationFightSeconds = 180
	rotationIterations   = 50
	// maxOverheal is the most of its healing a sensible rotation wastes
	// against the test profile.
	maxOverheal = 0.8
)

// The spells the written rotation casts, by id.
const (
	healingStreamTotemID = 10463
	lesserHealingWaveID  = 10468
	chainHealID          = 10623
	healingWaveID        = 25357
	riptideID            = 1239243
	naturesSwiftnessID   = 16188
	manaTideTotemID      = 17359
	waterShieldID        = 408510
)

var rotationSpells = []int32{
	healingStreamTotemID, lesserHealingWaveID, chainHealID, healingWaveID,
	riptideID, naturesSwiftnessID, manaTideTotemID, waterShieldID,
}

func writtenRotation() *proto.APLRotation {
	return core.GetAplRotation("../../../ui/restoration_shaman/apls", "forever_restoration").Rotation
}

func rotationRequest() *proto.RaidSimRequest {
	player := newHealerPlayer(60, FullTalents, writtenRotation())
	return healsim.Request(player, healsim.TestProfile(), rotationFightSeconds, rotationIterations)
}

func TestWrittenRotationHealsWithoutWastingItAll(t *testing.T) {
	summary, err := healsim.Run(rotationRequest())
	if err != nil {
		t.Fatal(err)
	}

	if summary.EffectiveHPS <= 0 {
		t.Fatalf("effective HPS = %v, want a healer that heals", summary.EffectiveHPS)
	}
	if summary.OverhealPct < 0 || summary.OverhealPct >= maxOverheal {
		t.Errorf("overheal = %.2f, want 0 to %.2f", summary.OverhealPct, maxOverheal)
	}
	if summary.RawHPS < summary.EffectiveHPS {
		t.Errorf("raw HPS %v is below effective HPS %v", summary.RawHPS, summary.EffectiveHPS)
	}
}

func TestWrittenRotationCastsEverySpellItNames(t *testing.T) {
	summary, err := healsim.Run(rotationRequest())
	if err != nil {
		t.Fatal(err)
	}

	for _, id := range rotationSpells {
		if summary.Casts[id] == 0 {
			t.Errorf("the rotation never cast spell %d in %d runs of %d s", id, rotationIterations, rotationFightSeconds)
		}
	}
}

// TestWrittenRotationSpendsWhatItsCastsCost checks the mana books: for every
// spell with a mana cost, the mana spent is its casts times its cost.
func TestWrittenRotationSpendsWhatItsCastsCost(t *testing.T) {
	req := rotationRequest()
	result := core.RunRaidSim(req)
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}
	metrics := result.RaidMetrics.Parties[0].Players[healsim.HealerIndex]
	_, healer := newHealer(t, 60, FullTalents)

	casts := map[int32]int32{}
	for _, action := range metrics.Actions {
		for _, target := range action.Targets {
			casts[action.Id.GetSpellId()] += target.Casts
		}
	}
	checked := 0
	for _, resource := range metrics.Resources {
		id := resource.Id.GetSpellId()
		if resource.Type != proto.ResourceType_ResourceTypeMana || resource.Gain >= 0 || id == 0 {
			continue
		}
		spell := healer.GetSpell(core.ActionID{SpellID: id})
		want := float64(casts[id]) * spell.Cost.GetCurrentCost()
		if math.Abs(-resource.Gain-want) > 1e-6*math.Max(1, want) {
			t.Errorf("spell %d spent %.2f mana over %d casts, want %.2f", id, -resource.Gain, casts[id], want)
		}
		checked++
	}
	if checked < len(rotationSpells)-2 {
		t.Errorf("checked the mana of only %d spells", checked)
	}
}

// The written rotation opens with the autocast line, so a shaman that
// carries a Major Mana Potion and a Demonic Rune drinks both: they are
// self-cast, and the healer's current target is a friend.
func TestWrittenRotationUsesTheManaConsumables(t *testing.T) {
	player := newHealerPlayer(60, FullTalents, writtenRotation())
	player.Consumes = healsim.ManaConsumables()
	result := core.RunRaidSim(healsim.Request(player, healsim.TestProfile(), rotationFightSeconds, rotationIterations))
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}
	metrics := result.RaidMetrics.Parties[0].Players[healsim.HealerIndex]
	for _, name := range healsim.UnusedManaConsumables(metrics) {
		t.Errorf("the shaman never used its %s", name)
	}
}
