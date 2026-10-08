package core

import (
	"time"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// The raid damage model gives a healer something to heal. The fake raid
// members (Raid.target_dummies) are players with a health bar; the
// model's tank takes melee hits and its other members take raid-wide
// pulses, all as plain health loss after mitigation, with active
// absorb shields soaking up their share first. Without a model the fake
// members stay at full health and every heal overheals completely.

// Randomness labels, so a recorded run replays.
const (
	raidDamageSpreadLabel = "Raid Damage Spread"
	raidDamageStartLabel  = "Raid Damage Start"
	raidDamagePickLabel   = "Raid Damage Pulse Pick"
)

// raidDamageDummies is the fake raid members in raid order. The last is
// the tank, the rest are the members pulses land on.
func (raid *Raid) raidDamageDummies() (tank *TargetDummy, members []*TargetDummy) {
	var dummies []*TargetDummy
	for _, party := range raid.Parties {
		for _, player := range party.Players {
			if dummy, ok := player.(*TargetDummy); ok {
				dummies = append(dummies, dummy)
			}
		}
	}
	if len(dummies) == 0 {
		return nil, nil
	}
	return dummies[len(dummies)-1], dummies[:len(dummies)-1]
}

// applyRaidDamageModel gives every fake member a health bar and starts
// the tank's swings and the raid's pulses on every reset. A nil model
// leaves the fake members as they were.
func (raid *Raid) applyRaidDamageModel(model *proto.RaidDamageModel) {
	if model == nil {
		return
	}
	tank, members := raid.raidDamageDummies()
	if tank == nil {
		return
	}
	for _, member := range members {
		enableRaidDamageHealth(member, model.MemberHealth)
	}
	enableRaidDamageHealth(tank, model.TankHealth)

	units := make([]*Unit, len(members))
	for i, member := range members {
		units[i] = &member.Unit
	}
	tank.RegisterResetEffect(func(sim *Simulation) {
		if model.TankHitDamage > 0 && model.TankSwingSeconds > 0 {
			scheduleRaidDamage(sim, model.TankSwingSeconds, func(sim *Simulation) {
				dealRaidDamage(sim, &tank.Unit, model.TankHitDamage, model.DamageSpread, true)
			})
		}
		if model.PulseDamage > 0 && model.PulseIntervalSeconds > 0 && len(units) > 0 {
			picked := make([]*Unit, len(units))
			scheduleRaidDamage(sim, model.PulseIntervalSeconds, func(sim *Simulation) {
				for _, unit := range pickPulseTargets(sim, units, picked, int(model.PulseMembers)) {
					dealRaidDamage(sim, unit, model.PulseDamage, model.DamageSpread, false)
				}
			})
		}
	})
}

func enableRaidDamageHealth(dummy *TargetDummy, health float64) {
	dummy.AddStat(stats.Health, health)
	dummy.EnableHealthBar()
}

// scheduleRaidDamage runs onTick every interval seconds from a random
// point inside the first interval, so the tank's swings and the pulses
// do not line up on one tick.
func scheduleRaidDamage(sim *Simulation, intervalSeconds float64, onTick func(*Simulation)) {
	interval := DurationFromSeconds(intervalSeconds)
	pa := &PendingAction{
		NextActionAt: time.Duration(sim.RandomFloat(raidDamageStartLabel) * float64(interval)),
	}
	pa.OnAction = func(sim *Simulation) {
		onTick(sim)
		pa.NextActionAt = sim.CurrentTime + interval
		sim.AddPendingAction(pa)
	}
	sim.AddPendingAction(pa)
}

// pickPulseTargets chooses count of units at random without repeats;
// count 0 or more than there are means all of them. scratch is reused.
func pickPulseTargets(sim *Simulation, units []*Unit, scratch []*Unit, count int) []*Unit {
	if count <= 0 || count >= len(units) {
		return units
	}
	copy(scratch, units)
	for i := 0; i < count; i++ {
		j := i + int(sim.RandomFloat(raidDamagePickLabel)*float64(len(scratch)-i))
		scratch[i], scratch[j] = scratch[j], scratch[i]
	}
	return scratch[:count]
}

// RaidDamageListener is told each time the damage model hurts a fake
// member: the unit, the damage that got through its absorbs, and whether
// the unit is the model's tank. It is how a healer's effect that "is
// cancelled by being attacked" (a Lightwell renew) or that fires when a
// member is hurt (a member clicking the Lightwell) hears of the damage,
// which is plain health loss and no spell.
type RaidDamageListener func(sim *Simulation, unit *Unit, damage float64, isTank bool)

// OnRaidDamage registers a listener for the rest of the sim. Register
// during initialization, once.
func (raid *Raid) OnRaidDamage(listener RaidDamageListener) {
	raid.raidDamageListeners = append(raid.raidDamageListeners, listener)
}

// dealRaidDamage takes amount (rolled within +/- spread) off a fake
// member's health after its absorb shields, then tells the listeners.
func dealRaidDamage(sim *Simulation, unit *Unit, amount, spread float64, isTank bool) {
	amount *= 1 + spread*(2*sim.RandomFloat(raidDamageSpreadLabel)-1)
	amount = unit.AbsorbDamage(sim, amount)
	if amount > 0 {
		unit.RemoveHealth(sim, amount)
		sim.Raid.notifyRaidDamage(sim, unit, amount, isTank)
	}
}

func (raid *Raid) notifyRaidDamage(sim *Simulation, unit *Unit, damage float64, isTank bool) {
	for _, listener := range raid.raidDamageListeners {
		listener(sim, unit, damage, isTank)
	}
}

// fakeRaidMembers is how many fake raid members a healing request adds:
// four fill the healer's party and the fifth, the tank, lands in the next.
const fakeRaidMembers = 5

// AddHealingFakeRaid gives raid the fake members a healing sim needs when
// model is set: a second, empty party for the tank to stand in, the five
// fake members and the model itself. A nil model leaves raid alone.
func AddHealingFakeRaid(raid *proto.Raid, model *proto.RaidDamageModel) {
	if model == nil {
		return
	}
	raid.Parties = append(raid.Parties, &proto.Party{})
	raid.TargetDummies = fakeRaidMembers
	raid.RaidDamageModel = model
}
