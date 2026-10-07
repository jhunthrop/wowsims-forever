package mage

import (
	"testing"
	"time"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// newMageAtLevel builds a prepulled mage of the given level with the
// given talent string, so spells can be driven through ApplyEffects.
func newMageAtLevel(t *testing.T, level int32, talentsStr string) (*core.Simulation, *Mage) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassMage,
			Race:               proto.Race_RaceTroll,
			Level:              level,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              core.FullBuffs.Player,
			TalentsString:      talentsStr,
			DistanceFromTarget: 5,
			Rotation:           &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		},
		PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 120,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	agent, ok := sim.Raid.Parties[0].Players[0].(MageAgent)
	if !ok {
		t.Fatal("the raid's first player is not a mage agent")
	}
	return sim, agent.GetMage()
}

func registeredFrostfireRanks(mage *Mage) []int {
	var ranks []int
	for rank, spell := range mage.FrostfireBolt {
		if spell != nil {
			ranks = append(ranks, rank)
		}
	}
	return ranks
}

// The client's three trainable ranks are at levels 40, 50 and 60; the
// generated rank 0 is the level-1 teaching spell and is never cast.
func TestFrostfireBoltRegistersByLevel(t *testing.T) {
	cases := []struct {
		level int32
		want  []int
	}{
		{39, nil},
		{40, []int{1}},
		{50, []int{1, 2}},
		{60, []int{1, 2, 3}},
	}
	for _, tc := range cases {
		_, mage := newMageAtLevel(t, tc.level, ForeverFrostTalents)
		got := registeredFrostfireRanks(mage)
		if len(got) != len(tc.want) {
			t.Errorf("level %d: ranks %v, want %v", tc.level, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("level %d: ranks %v, want %v", tc.level, got, tc.want)
			}
		}
	}
}

func TestFrostfireBoltIsADualSchoolMageSpell(t *testing.T) {
	_, mage := newMageAtLevel(t, 60, ForeverFrostTalents)
	spell := mage.FrostfireBolt[FrostfireBoltRanks]
	if spell == nil {
		t.Fatal("level-60 mage has no top-rank Frostfire Bolt")
	}
	if want := core.SpellSchoolFire | core.SpellSchoolFrost; spell.SpellSchool != want {
		t.Errorf("school = %v, want Fire|Frost", spell.SpellSchool)
	}
	if spell.ClassSpellMask != MageSpellMaskFrostfireBolt {
		t.Errorf("class mask = %#x, want %#x", spell.ClassSpellMask, MageSpellMaskFrostfireBolt)
	}
	if !spell.Flags.Matches(SpellFlagMage) {
		t.Error("Frostfire Bolt is not flagged as a mage spell, so Clearcasting and the school mods skip it")
	}
	if spell.SpellCode != SpellCode_MageFrostfireBolt {
		t.Errorf("spell code = %d, want SpellCode_MageFrostfireBolt", spell.SpellCode)
	}
	if got, want := spell.CastTime(), time.Duration(FrostfireBoltCastTime[FrostfireBoltRanks])*time.Millisecond; got != want {
		t.Errorf("cast time = %v, want the client's %v", got, want)
	}
}

// "This spell will be checked against the lower of the target's Frost
// and Fire resists": with Fire at 0 and Frost high, a pure Frost spell
// is resisted and Frostfire Bolt is not.
func TestFrostfireBoltUsesTheLowerOfTheTwoResists(t *testing.T) {
	sim, mage := newMageAtLevel(t, 60, ForeverFrostTalents)
	target := sim.Encounter.TargetUnits[0]
	// The full-buff debuffs leave both resists negative; set them to
	// exact values rather than adding to whatever they are.
	target.AddStatDynamic(sim, stats.FrostResistance, 300-target.GetStat(stats.FrostResistance))
	target.AddStatDynamic(sim, stats.FireResistance, -target.GetStat(stats.FireResistance))

	frostfire := mage.FrostfireBolt[FrostfireBoltRanks]
	frostbolt := mage.Frostbolt[FrostboltRanks]
	hitChance := func(spell *core.Spell) float64 {
		return mage.AttackTables[target.UnitIndex][spell.CastType].GetBinaryHitChance(spell)
	}
	if got := hitChance(frostfire); got != 1 {
		t.Errorf("Frostfire Bolt binary hit chance %v with Fire resist 0, want 1", got)
	}
	if got := hitChance(frostbolt); got >= 1 {
		t.Errorf("Frostbolt binary hit chance %v against Frost resist 300, want a resisted spell", got)
	}
}

func TestFrostfireBoltDealsDirectDamageAndA9SecondDot(t *testing.T) {
	sim, mage := newMageAtLevel(t, 60, ForeverFrostTalents)
	target := sim.Encounter.TargetUnits[0]
	spell := mage.FrostfireBolt[FrostfireBoltRanks]
	forceOutcome(spell, false)

	castAndLand(sim, target, spell)
	dot := spell.Dot(target)
	if !dot.IsActive() {
		t.Fatal("a landed Frostfire Bolt applied no dot")
	}
	if dot.NumberOfTicks != FrostfireBoltDotTicks || dot.TickLength != 3*time.Second {
		t.Errorf("dot = %d ticks of %v, want %d of 3s", dot.NumberOfTicks, dot.TickLength, FrostfireBoltDotTicks)
	}
	if got := dot.Aura.Duration; got != 9*time.Second {
		t.Errorf("dot duration = %v, want 9s", got)
	}
}

func TestImprovedFireballShortensFrostfireBolt(t *testing.T) {
	talents := talentStringWithRank(t, ForeverFrostTalents, "improved_fireball", 5)
	_, mage := newMageAtLevel(t, 60, talents)
	want := time.Duration(FrostfireBoltCastTime[FrostfireBoltRanks])*time.Millisecond - 500*time.Millisecond
	if got := mage.FrostfireBolt[FrostfireBoltRanks].CastTime(); got != want {
		t.Errorf("cast time with 5/5 Improved Fireball = %v, want %v", got, want)
	}
	// The Fireball half of the same talent, in the same place: one
	// talent, one reduction.
	wantFireball := time.Duration(FireballCastTime[FireballRanks])*time.Millisecond - 500*time.Millisecond
	if got := mage.Fireball[FireballRanks].CastTime(); got != wantFireball {
		t.Errorf("Fireball cast time with 5/5 Improved Fireball = %v, want %v", got, wantFireball)
	}
}

func TestFrostfireBoltFeedsHeatingUp(t *testing.T) {
	sim, mage := newHeatingUpTestMage(t)
	target := sim.Encounter.TargetUnits[0]
	spell := mage.FrostfireBolt[FrostfireBoltRanks]
	forceOutcome(spell, true)
	castAndLand(sim, target, spell)
	if got := heatingUpStacks(mage); got != 1 {
		t.Errorf("a Frostfire Bolt crit gave %d Heating Up stack(s), want 1", got)
	}
}

func TestFrostfireBoltCanTriggerMissileBarrage(t *testing.T) {
	talents := talentStringWithRank(t, ForeverFrostTalents, "missile_barrage", 1)
	sim, mage := newMageAtLevel(t, 60, talents)
	spell := mage.FrostfireBolt[FrostfireBoltRanks]
	for i := 0; i < 200; i++ {
		mage.OnCastComplete(sim, spell)
		if mage.MissileBarrageAura.IsActive() {
			return
		}
	}
	t.Error("200 Frostfire Bolt casts never triggered Missile Barrage at its 20% chance")
}
