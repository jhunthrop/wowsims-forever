package mage

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

const IceBarrierRanks = 4

var IceBarrierSpellId = [IceBarrierRanks + 1]int32{0, 11426, 13031, 13032, 13033}
var IceBarrierManaCost = [IceBarrierRanks + 1]float64{0, 305, 360, 420, 480}
var IceBarrierLevel = [IceBarrierRanks + 1]int{0, 40, 46, 52, 58}

// Ice Barrier (client 1.60.1.70532): "Shield yourself, absorbing [448 +
// (Frost spell power * 0.1)] damage and increasing your Frost damage by
// 3% for 1 min. While shielded, your Frost damage is increased by an
// additional 3% and you do not lose casting time when you take damage."
// Nothing hits a simmed mage, so the absorb is not modeled and a shield
// holds its full minute; the two 3%s stay separate mods so an absorb
// model can end the shielded one early.
const (
	iceBarrierDuration               = time.Minute
	iceBarrierFrostDamagePct         = 3
	iceBarrierShieldedFrostDamagePct = 3
)

func (mage *Mage) registerIceBarrierSpell() {
	if !mage.Talents.IceBarrier {
		return
	}

	mage.IceBarrier = make([]*core.Spell, IceBarrierRanks+1)
	mage.IceBarrierAuras = make([]*core.Aura, IceBarrierRanks+1)
	mage.registerIceBarrierFrostDamage()

	cdTimer := mage.NewTimer()

	for rank := 1; rank <= IceBarrierRanks; rank++ {
		config := mage.newIceBarrierSpellConfig(rank, cdTimer)

		if config.RequiredLevel <= int(mage.Level) {
			mage.IceBarrier[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

func (mage *Mage) newIceBarrierSpellConfig(rank int, cdTimer *core.Timer) core.SpellConfig {
	spellID := IceBarrierSpellId[rank]
	manaCost := IceBarrierManaCost[rank]
	level := IceBarrierLevel[rank]

	cooldown := time.Second * 30

	// The absorb itself is not modeled (see iceBarrierDuration); the aura
	// carries the shielded Frost damage half.
	mage.IceBarrierAuras[rank] = mage.RegisterAura(core.Aura{
		ActionID: core.ActionID{SpellID: spellID},
		Label:    fmt.Sprintf("Ice Barrier (Rank %d)", rank),
		Duration: iceBarrierDuration,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			mage.iceBarrierShieldedMod.Activate()
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			mage.iceBarrierShieldedMod.Deactivate()
		},
	})

	return core.SpellConfig{
		ActionID:        core.ActionID{SpellID: spellID},
		ClassSpellMask:  MageSpellMaskIceBarrier,
		SpellSchool:     core.SpellSchoolFrost,
		Flags:           core.SpellFlagAPL | core.SpellFlagNoOnCastComplete,
		Rank:            rank,
		RequiredLevel:   level,
		RelatedSelfBuff: mage.IceBarrierAuras[rank],
		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    cdTimer,
				Duration: cooldown,
			},
		},
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// One shield at a time: a recast of any rank replaces it.
			if mage.activeBarrier != nil && mage.activeBarrier.IsActive() {
				mage.activeBarrier.Deactivate(sim)
			}
			mage.activeBarrier = mage.IceBarrierAuras[rank]
			mage.activeBarrier.Activate(sim)
			mage.IceBarrierEmpowermentAura.Activate(sim)
		},
	}
}

// registerIceBarrierFrostDamage sets up both Frost damage halves: the
// one-minute empowerment every cast grants, and the shielded extra that
// each rank's shield aura holds while it is up.
func (mage *Mage) registerIceBarrierFrostDamage() {
	empowerment := mage.AddDynamicMod(core.SpellModConfig{
		Kind:      core.SpellMod_DamageDone_Flat,
		ClassMask: MageSpellMaskFrostDamage,
		IntValue:  iceBarrierFrostDamagePct,
	})
	mage.IceBarrierEmpowermentAura = mage.RegisterAura(core.Aura{
		Label:    "Ice Barrier (Frost Damage)",
		ActionID: core.ActionID{SpellID: IceBarrierSpellId[1], Tag: 1},
		Duration: iceBarrierDuration,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			empowerment.Activate()
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			empowerment.Deactivate()
		},
	})
	mage.iceBarrierShieldedMod = mage.AddDynamicMod(core.SpellModConfig{
		Kind:      core.SpellMod_DamageDone_Flat,
		ClassMask: MageSpellMaskFrostDamage,
		IntValue:  iceBarrierShieldedFrostDamagePct,
	})
}
