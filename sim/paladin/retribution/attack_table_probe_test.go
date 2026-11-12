package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Probe: the melee and ranged attack tables of a level 40 Human paladin
// against a level 43 target, and Judgement of Righteousness's hit chance.
func TestProbeJudgementAttackTables(t *testing.T) {
	player := utilitySealPaladin(40, `{"type": "TypeAPL"}`)
	player.Equipment = &proto.EquipmentSpec{Items: []*proto.ItemSpec{{Id: 870}, {Id: 249442}}}
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	target := core.NewDefaultTarget()
	target.Level = 43
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{target}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	unit := &sim.Raid.Parties[0].Players[0].GetCharacter().Unit
	enemy := sim.Encounter.TargetUnits[0]
	for _, ct := range []proto.CastType{proto.CastType_CastTypeMainHand, proto.CastType_CastTypeRanged} {
		at := unit.AttackTables[enemy.UnitIndex][ct]
		if at == nil {
			t.Logf("%v: no table", ct)
			continue
		}
		t.Logf("%v: weapon %v baseMiss %.4f suppression %.4f critSupp %.4f", ct, at.Weapon != nil, at.BaseMissChance, at.HitSuppression, at.MeleeCritSuppression)
	}
	for _, id := range []int32{20283, 10333} {
		sp := unit.GetSpell(core.ActionID{SpellID: id})
		if sp == nil {
			continue
		}
		at := unit.AttackTables[enemy.UnitIndex][sp.CastType]
		t.Logf("spell %d castType %v bonusHit %.2f physicalHit %.4f", id, sp.CastType, sp.BonusHitRating, sp.PhysicalHitChance(at))
	}
	t.Logf("level %d mh %v", unit.Level, unit.AutoAttacks.MH().SwingSpeed)
}
