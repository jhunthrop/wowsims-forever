package elemental

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/shaman"
)

const fireNovaTestTargets = 3

var (
	callOfFlameNode      = elementalNode("call_of_flame")
	improvedFireNovaNode = elementalNode("improved_fire_nova")
	elementalFocusNode   = elementalNode("elemental_focus")
)

func newFireNovaShaman(t *testing.T, level int32, talents string) (*core.Simulation, *shaman.Shaman) {
	t.Helper()

	targets := make([]*proto.Target, fireNovaTestTargets)
	for i := range targets {
		targets[i] = core.DefaultTargetProtoLvl60
	}
	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassShaman,
			Race:          proto.Race_RaceTroll,
			Level:         level,
			Equipment:     &proto.EquipmentSpec{},
			Buffs:         core.FullBuffs.Player,
			TalentsString: talents,
		},
		&proto.Player_ElementalShaman{
			ElementalShaman: &proto.ElementalShaman{Options: &proto.ElementalShaman_Options{}},
		},
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: targets},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	agent, ok := sim.Raid.Parties[0].Players[0].(shaman.ShamanAgent)
	if !ok {
		t.Fatal("the raid's first player is not a shaman agent")
	}
	return sim, agent.GetShaman()
}

func topFireNova(t *testing.T, s *shaman.Shaman) *core.Spell {
	t.Helper()
	spell := s.FireNova[shaman.FireNovaLearnRanks]
	if spell == nil {
		t.Fatal("level-60 shaman has no Fire Nova rank 5")
	}
	return spell
}

func dropSearingTotem(sim *core.Simulation, s *shaman.Shaman) {
	totem := s.SearingTotem[len(s.SearingTotem)-1]
	totem.ApplyEffects(sim, sim.GetTargetUnit(0), totem)
}

// Fire Nova is a trainable with learn rows at 12/22/32/42/52 and no
// vanilla Fire Nova Totem learn row: the ranks are the five client ids
// and the totem registration is gone.
func TestFireNovaRanksFollowTheLearnLevels(t *testing.T) {
	cases := []struct {
		level    int32
		wantRank int
		wantID   int32
	}{
		{level: 12, wantRank: 1, wantID: 408341},
		{level: 21, wantRank: 1, wantID: 408341},
		{level: 22, wantRank: 2, wantID: 408342},
		{level: 32, wantRank: 3, wantID: 408343},
		{level: 42, wantRank: 4, wantID: 408344},
		{level: 60, wantRank: 5, wantID: 408345},
	}
	for _, tc := range cases {
		_, s := newFireNovaShaman(t, tc.level, elementalTalentsWith(nil))
		top := 0
		for rank, spell := range s.FireNova {
			if spell != nil {
				top = rank
			}
		}
		if top != tc.wantRank {
			t.Fatalf("level %d: top Fire Nova rank = %d, want %d", tc.level, top, tc.wantRank)
		}
		if got := s.FireNova[top].ActionID.SpellID; got != tc.wantID {
			t.Errorf("level %d: Fire Nova spell id = %d, want %d", tc.level, got, tc.wantID)
		}
	}

	_, low := newFireNovaShaman(t, 11, elementalTalentsWith(nil))
	for rank, spell := range low.FireNova {
		if spell != nil {
			t.Errorf("level 11: Fire Nova rank %d registered before the first learn level", rank)
		}
	}
}

func TestFireNovaCostsAndRecoversPerTheClient(t *testing.T) {
	_, s := newFireNovaShaman(t, 60, elementalTalentsWith(nil))
	nova := topFireNova(t, s)
	if got, want := nova.DefaultCast.Cost, 520.0; got != want {
		t.Errorf("Fire Nova rank 5 cost = %v, want %v", got, want)
	}
	if got, want := nova.CD.Duration, 10*time.Second; got != want {
		t.Errorf("Fire Nova cooldown = %v, want %v", got, want)
	}
	if got, want := nova.DefaultCast.GCD, 1500*time.Millisecond; got != want {
		t.Errorf("Fire Nova GCD = %v, want %v", got, want)
	}
}

func TestFireNovaNeedsAnActiveFireTotem(t *testing.T) {
	sim, s := newFireNovaShaman(t, 60, elementalTalentsWith(nil))
	nova := topFireNova(t, s)
	target := sim.GetTargetUnit(0)

	if nova.CanCast(sim, target) {
		t.Fatal("Fire Nova castable with no fire totem down")
	}
	dropSearingTotem(sim, s)
	if !nova.CanCast(sim, target) {
		t.Fatal("Fire Nova not castable with Searing Totem down")
	}

	sim.CurrentTime = s.TotemExpirations[shaman.FireTotem] + time.Second
	if nova.CanCast(sim, target) {
		t.Fatal("Fire Nova castable after the fire totem expired")
	}
}

func TestFireNovaHitsEveryEnemy(t *testing.T) {
	sim, s := newFireNovaShaman(t, 60, elementalTalentsWith(nil))
	nova := topFireNova(t, s)
	dropSearingTotem(sim, s)
	nova.ApplyEffects(sim, sim.GetTargetUnit(0), nova)

	blast := s.FireNovaBlast[shaman.FireNovaLearnRanks]
	for i := 0; i < fireNovaTestTargets; i++ {
		metrics := blast.SpellMetrics[sim.GetTargetUnit(int32(i)).UnitIndex]
		if metrics.Hits+metrics.Crits+metrics.Misses != 1 {
			t.Errorf("target %d took %d Fire Nova results, want 1", i, metrics.Hits+metrics.Crits+metrics.Misses)
		}
	}
}

func TestFireNovaTalentMultipliers(t *testing.T) {
	_, bare := newFireNovaShaman(t, 60, elementalTalentsWith(nil))
	baseline := bare.FireNovaBlast[shaman.FireNovaLearnRanks]

	_, cof := newFireNovaShaman(t, 60, elementalTalentsWith(map[int]int{callOfFlameNode: 3}))
	if got, want := cof.FireNovaBlast[shaman.FireNovaLearnRanks].DamageMultiplier, baseline.DamageMultiplier+0.15; !near(got, want) {
		t.Errorf("Call of Flame 3/3 multiplier = %v, want %v", got, want)
	}

	_, ifn := newFireNovaShaman(t, 60, elementalTalentsWith(map[int]int{improvedFireNovaNode: 2}))
	if got, want := ifn.FireNovaBlast[shaman.FireNovaLearnRanks].DamageMultiplier, baseline.DamageMultiplier+0.20; !near(got, want) {
		t.Errorf("Improved Fire Nova 2/2 multiplier = %v, want %v", got, want)
	}
	if got, want := topFireNova(t, ifn).CD.Duration, 8*time.Second; got != want {
		t.Errorf("Improved Fire Nova 2/2 cooldown = %v, want %v", got, want)
	}

	_, both := newFireNovaShaman(t, 60, elementalTalentsWith(map[int]int{callOfFlameNode: 3, improvedFireNovaNode: 2}))
	if got, want := both.FireNovaBlast[shaman.FireNovaLearnRanks].DamageMultiplier, baseline.DamageMultiplier+0.35; !near(got, want) {
		t.Errorf("Call of Flame + Improved Fire Nova multiplier = %v, want the additive %v", got, want)
	}
}

// The client's Concussion text names Lightning Bolt, Chain Lightning and
// Earth Shock only; Elemental Fury names Fire spells.
func TestFireNovaConcussionAndFury(t *testing.T) {
	_, bare := newFireNovaShaman(t, 60, elementalTalentsWith(nil))
	base := bare.FireNovaBlast[shaman.FireNovaLearnRanks]

	_, conc := newFireNovaShaman(t, 60, elementalTalentsWith(map[int]int{concussionNode: 5}))
	if got := conc.FireNovaBlast[shaman.FireNovaLearnRanks].DamageMultiplierAdditive; !near(got, base.DamageMultiplierAdditive) {
		t.Errorf("Concussion moved Fire Nova's additive multiplier to %v", got)
	}

	_, fury := newFireNovaShaman(t, 60, elementalTalentsWith(map[int]int{elementalFuryNode: 5}))
	if got, want := fury.FireNovaBlast[shaman.FireNovaLearnRanks].CritDamageBonus, base.CritDamageBonus+1.0; !near(got, want) {
		t.Errorf("Elemental Fury 5/5 Fire Nova crit damage bonus = %v, want %v", got, want)
	}
}

// Elemental Focus: Clearcasting makes the next damaging cast free, and a
// Fire Nova cast is one of those casts.
func TestFireNovaSharesClearcasting(t *testing.T) {
	sim, s := newFireNovaShaman(t, 60, elementalTalentsWith(map[int]int{elementalFocusNode: 1}))
	nova := topFireNova(t, s)
	if got := nova.Cost.GetCurrentCost(); got != 520 {
		t.Fatalf("Fire Nova cost before Clearcasting = %v, want 520", got)
	}
	s.ClearcastingAura.Activate(sim)
	s.ClearcastingAura.SetStacks(sim, s.ClearcastingAura.MaxStacks)
	if got := nova.Cost.GetCurrentCost(); got != 0 {
		t.Errorf("Fire Nova cost under Clearcasting = %v, want 0", got)
	}
}

// Mana Spring Totem restores "10 mana every 2 seconds" at rank 4 for
// five minutes (spell 10497's tooltip and 300 s duration). The APL cast
// used to only mark the water slot occupied; the restore is now a real
// MP5 aura, so a solo shaman benefits from dropping it.
func TestManaSpringTotemRestoresManaForFiveMinutes(t *testing.T) {
	sim, s := newFireNovaShaman(t, 60, elementalTalentsWith(nil))
	spring := s.ManaSpringTotem[shaman.ManaSpringTotemRanks]
	if spring == nil {
		t.Fatal("level-60 shaman has no Mana Spring Totem rank 4")
	}

	before := s.GetStat(stats.MP5)
	spring.ApplyEffects(sim, sim.GetTargetUnit(0), spring)
	if got, want := s.GetStat(stats.MP5)-before, 25.0; !near(got, want) {
		t.Errorf("Mana Spring Totem rank 4 MP5 = +%v, want +%v (10 mana per 2 s)", got, want)
	}
	if got, want := s.TotemExpirations[shaman.WaterTotem], sim.CurrentTime+5*time.Minute; got != want {
		t.Errorf("Mana Spring Totem expires at %v, want %v (client duration 300 s)", got, want)
	}
}
