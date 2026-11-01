package hunter

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// newLevel60HunterWithFullBuffs builds a level-60 hunter with every raid,
// party and individual buff the test presets carry, so a pet that still
// receives player buffs shows it in its stat sheet.
func newLevel60HunterWithFullBuffs(t *testing.T) *Hunter {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              core.FullBuffs.Player,
			DistanceFromTarget: 25,
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)
	built, ok := env.Raid.Parties[0].Players[0].(*Hunter)
	if !ok || built.pet == nil {
		t.Fatal("player 0 did not build as a *Hunter with a pet")
	}
	return built
}

// Forever reworked hunter pets: they "can no longer receive player buffs
// that boost their stats" (Blizzard's hunter deep dive, 2026-09-30). The
// pet's stat sheet before it inherits anything is its own base: Strength
// 136, Agility 100, and 136 x 2 - 20 = 252 attack power.
func TestHunterPetReceivesNoPlayerBuffs(t *testing.T) {
	pet := newLevel60HunterWithFullBuffs(t).pet
	got := pet.GetStats()
	if got[stats.Strength] != 136 || got[stats.Agility] != 100 {
		t.Errorf("pet Strength/Agility = %.1f/%.1f, want the unbuffed 136/100", got[stats.Strength], got[stats.Agility])
	}
	if math.Abs(got[stats.AttackPower]-252) > 1e-6 {
		t.Errorf("pet attack power = %.1f, want the unbuffed 252", got[stats.AttackPower])
	}
}

// A Forever pet inherits 10% of the hunter's attack power, whichever of
// melee and ranged is higher, and 100% of the hunter's crit chance
// (client Hunter Pet Scaling 415429 carries the channels, aura 99 and
// aura 52, with server-computed amounts; the figures are from the
// Warcraft Tavern Forever hunter and pet guides and MMOGAH).
func TestHunterPetInheritsTenPercentAttackPowerAndAllCrit(t *testing.T) {
	hunter := newLevel60HunterWithFullBuffs(t)
	inherit := hunter.makeStatInheritance()

	ranged := stats.Stats{stats.AttackPower: 1000, stats.RangedAttackPower: 2000, stats.Crit: 25 * core.CritRatingPerCritChance}
	hunter.petAttackPowerSource = higherAttackPower(ranged)
	got := inherit(ranged)
	if got[stats.AttackPower] != 200 {
		t.Errorf("ranged hunter: pet inherits %.1f attack power, want 10%% of 2000 ranged = 200", got[stats.AttackPower])
	}
	if got[stats.Crit] != 25*core.CritRatingPerCritChance {
		t.Errorf("pet inherits %.2f crit, want all of the hunter's %.2f", got[stats.Crit], float64(25*core.CritRatingPerCritChance))
	}

	melee := stats.Stats{stats.AttackPower: 1800, stats.RangedAttackPower: 900}
	hunter.petAttackPowerSource = higherAttackPower(melee)
	if got := inherit(melee); got[stats.AttackPower] != 180 {
		t.Errorf("melee hunter: pet inherits %.1f attack power, want 10%% of 1800 melee = 180", got[stats.AttackPower])
	}
}

// Hunter Pet Scaling (415429) carries the melee and ranged hit channels
// (aura 54 and 55, server-computed amounts), the same channels as Warlock
// Pet Scaling (416189), whose demons inherit the warlock's hit. wowsims/sod,
// on the same client spell, gives the pet all of the hunter's hit
// (sim/hunter/pet.go makeStatInheritance). Sim-audit round 4, physical B3.
func TestHunterPetInheritsTheHuntersHit(t *testing.T) {
	hunter := newLevel60HunterWithFullBuffs(t)
	inherit := hunter.makeStatInheritance()
	owner := stats.Stats{stats.RangedAttackPower: 2000, stats.Hit: 6 * core.HitRatingPerHitChance}
	hunter.petAttackPowerSource = higherAttackPower(owner)
	if got := inherit(owner)[stats.Hit]; got != 6*core.HitRatingPerHitChance {
		t.Errorf("pet inherits %.2f hit, want all of the hunter's %.2f", got, float64(6*core.HitRatingPerHitChance))
	}
}
