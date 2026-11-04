package hunter

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Summon Hawk's second half, "continuing its assault for 18 sec. Only 2
// hawks can be active at once." The client gives it no data: every rank's
// effect 1 triggers 1312639, a dummy with no amount, period or creature.
// The hawk is server-side, so its kit comes from the beta combat logs
// (foreverlogs.gg reports 2695-2764, 48 encounters, 8 Beast Mastery
// hunters at 25-35; round 16 hunter.md H2, round 18 hunter.md H1):
//   - a "Hawk" pet unit, owned by the hunter, appears a median 1.07 s after
//     the cast (quartiles 1.02-1.17 s) and auto attacks in melee;
//   - it swings every 2.26 s (median 2247-2273 ms at every level) and Rapid
//     Fire hastes it (1.62 s inside Rapid Fire);
//   - a summon on a target that lives lands eight swings (the mode; 718
//     swings, 0 glancing);
//   - it crits 0-1% below Ferocity and 10.6-11.6% with it, so it takes
//     Ferocity and none of the hunter's crit; the talent texts name "your
//     pets and hawks" for Ferocity and Unleashed Fury;
//   - every rank triggers the same 1312639, and the rank-1 swing grows with
//     the hunter's level: non-crit means 8.4, 9.1, 10.1, 9.4, 10.4, 12.8 at
//     25, 26, 27, 28, 30, 35.
//
// The level-60 swing is a projection. hawkSwingDamagePerLevel is the
// floor of the defensible readings: the level-35 talented mean (12.8 on
// level-35 mobs, Unleashed Fury 5) taken as unarmored and proportional to
// level, like a hunter pet's weapon (hunterPetDamageLevelScale), 12.8 /
// 1.15 / 35 = 0.32 a level. The pooled fit through the origin gives 0.35,
// and any armor on the beta mobs pushes the raw swing higher. It takes no
// attack power from the hunter (the logs cannot show any), so gear does
// not grow it. The owner's in-game check at 60 confirms or replaces it.
const (
	hawkArrivalDelay        = 1070 * time.Millisecond
	hawkAssaultDuration     = 18 * time.Second
	hawkSwingSpeed          = 2.26
	hawkMaxActive           = 2
	hawkSwingDamagePerLevel = 0.32
	hawkSwingSpread         = 0.15 // the logs' non-crit range, about +/-15% of the mean
	hawkAssaultSpellID      = 1312639
	// hawkPool is one more hawk than can be out, so a fresh summon never
	// reuses the hawk it just replaced (whose swing timer is still running).
	hawkPool = hawkMaxActive + 1
)

type Hawk struct {
	core.Pet

	hunter     *Hunter
	summonedAt time.Duration
	retiredAt  time.Duration
	hasted     bool
}

func (hunter *Hunter) newHawk(index int) *Hawk {
	hawk := &Hawk{
		Pet:    core.NewPet(fmt.Sprintf("Hawk %d", index+1), &hunter.Character, stats.Stats{}, hawkInheritsNothing, false, true),
		hunter: hunter,
	}

	mean := hawkSwingDamagePerLevel * float64(hunter.Level)
	hawk.EnableAutoAttacks(hawk, core.AutoAttackOptions{
		MainHand: core.Weapon{
			BaseDamageMin: mean * (1 - hawkSwingSpread),
			BaseDamageMax: mean * (1 + hawkSwingSpread),
			SwingSpeed:    hawkSwingSpeed,
			SpellSchool:   core.SpellSchoolPhysical,
		},
		AutoSwingMelee: true,
	})

	hawk.AddStat(stats.Crit, core.CritRatingPerCritChance*ferocityCritPerRank*float64(hunter.Talents.Ferocity))
	hawk.PseudoStats.DamageDealtMultiplier *= 1 + unleashedFuryPerRank*float64(hunter.Talents.UnleashedFury)

	hawk.OnPetEnable = func(sim *core.Simulation) {
		if hunter.RapidFireAura.IsActive() {
			hawk.setHasted(sim, true)
		}
	}
	hawk.OnPetDisable = func(sim *core.Simulation) {
		hawk.setHasted(sim, false)
		hawk.retiredAt = sim.CurrentTime
		hunter.updateHawkCount(sim)
	}

	hunter.AddPet(hawk)
	return hawk
}

func hawkInheritsNothing(stats.Stats) stats.Stats {
	return stats.Stats{}
}

func (hawk *Hawk) GetPet() *core.Pet                        { return &hawk.Pet }
func (hawk *Hawk) Initialize()                              {}
func (hawk *Hawk) Reset(_ *core.Simulation)                 {}
func (hawk *Hawk) ExecuteCustomRotation(_ *core.Simulation) {}

// setHasted applies or removes Rapid Fire's 40% on the hawk's swing.
func (hawk *Hawk) setHasted(sim *core.Simulation, hasted bool) {
	if hawk.hasted == hasted {
		return
	}
	hawk.hasted = hasted
	if hasted {
		hawk.MultiplyAttackSpeed(sim, rapidFireHaste)
	} else {
		hawk.MultiplyAttackSpeed(sim, 1/rapidFireHaste)
	}
}

// registerHawks builds the hawk pets. It runs in NewHunter, where pets must
// be added, for a hunter who knows Summon Hawk.
func (hunter *Hunter) registerHawks() {
	if !hunter.Talents.SummonHawk || hunter.Level < int32(SummonHawkLevel[1]) {
		return
	}
	for i := 0; i < hawkPool; i++ {
		hunter.hawks = append(hunter.hawks, hunter.newHawk(i))
	}
}

// registerHawkAssaultAura is the rotation's view of the hawks: its stacks
// are the hawks out.
func (hunter *Hunter) registerHawkAssaultAura() {
	if len(hunter.hawks) == 0 {
		return
	}
	hunter.HawkAssaultAura = hunter.RegisterAura(core.Aura{
		Label:     "Hawk Assault",
		ActionID:  core.ActionID{SpellID: hawkAssaultSpellID},
		Duration:  core.NeverExpires,
		MaxStacks: hawkMaxActive,
	})
}

func (hunter *Hunter) updateHawkCount(sim *core.Simulation) {
	if hunter.HawkAssaultAura == nil {
		return
	}
	n := int32(0)
	for _, hawk := range hunter.hawks {
		if hawk.IsEnabled() {
			n++
		}
	}
	if n == 0 {
		hunter.HawkAssaultAura.Deactivate(sim)
		return
	}
	hunter.HawkAssaultAura.Activate(sim)
	hunter.HawkAssaultAura.SetStacks(sim, n)
}

// sendHawk lands a hawk hawkArrivalDelay after the cast. With two out, the
// oldest leaves for the new one.
func (hunter *Hunter) sendHawk(sim *core.Simulation) {
	if len(hunter.hawks) == 0 {
		return
	}
	core.StartDelayedAction(sim, core.DelayedActionOptions{
		DoAt: sim.CurrentTime + hawkArrivalDelay,
		OnAction: func(sim *core.Simulation) {
			var oldest, free *Hawk
			active := 0
			for _, hawk := range hunter.hawks {
				if hawk.IsEnabled() {
					active++
					if oldest == nil || hawk.summonedAt < oldest.summonedAt {
						oldest = hawk
					}
				} else if free == nil || hawk.retiredAt < free.retiredAt {
					free = hawk
				}
			}
			if active >= hawkMaxActive {
				oldest.Disable(sim)
			}
			free.summonedAt = sim.CurrentTime
			free.EnableWithTimeout(sim, free, hawkAssaultDuration)
			hunter.updateHawkCount(sim)
		},
	})
}

func (hunter *Hunter) activeHawk() *Hawk {
	for _, hawk := range hunter.hawks {
		if hawk.IsEnabled() {
			return hawk
		}
	}
	return nil
}

// hasteHawks follows Rapid Fire onto the hawks out.
func (hunter *Hunter) hasteHawks(sim *core.Simulation, hasted bool) {
	for _, hawk := range hunter.hawks {
		if hawk.IsEnabled() {
			hawk.setHasted(sim, hasted)
		}
	}
}
