package shaman

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Mana Tide Totem (talent 104728, learned in three ranks) restores mana to
// the shaman's group for a few seconds. The totem lasts 13 s and the
// restore ticks every 3 s, so four ticks land; the client's per-tick
// amount is the generated ManaTide row (spells 16191, 17355, 17360).
const (
	manaTideTotemDuration = 13 * time.Second
	manaTideTotemTickGap  = 3 * time.Second
	manaTideTotemTicks    = 4
	manaTideTotemCooldown = 5 * time.Minute
)

func (shaman *Shaman) registerManaTideTotemSpell() {
	if !shaman.Talents.ManaTideTotem {
		return
	}

	// The ranks are one spell to the client, so they share the cooldown.
	cooldown := core.Cooldown{Timer: shaman.NewTimer(), Duration: manaTideTotemCooldown}
	shaman.ManaTideTotem = make([]*core.Spell, ManaTideTotemRanks+1)
	for rank := 1; rank <= ManaTideTotemRanks; rank++ {
		if ManaTideTotemLevel[rank] <= int(shaman.Level) {
			shaman.ManaTideTotem[rank] = shaman.RegisterSpell(shaman.newManaTideTotemConfig(rank, cooldown))
		}
	}
}

func (shaman *Shaman) newManaTideTotemConfig(rank int, cooldown core.Cooldown) core.SpellConfig {
	actionID := core.ActionID{SpellID: ManaTideTotemSpellId[rank]}
	manaPerTick := ManaTideBaseDamage[rank][0]

	members := shaman.Party.Players
	metrics := make([]*core.ResourceMetrics, len(members))
	for i, member := range members {
		if unit := &member.GetCharacter().Unit; unit.HasManaBar() {
			metrics[i] = unit.NewManaMetrics(actionID)
		}
	}

	var ticker *core.PendingAction
	aura := shaman.RegisterAura(core.Aura{
		Label:    fmt.Sprintf("Mana Tide Totem (Rank %d)", rank),
		ActionID: actionID,
		Duration: manaTideTotemDuration,
		OnGain: func(_ *core.Aura, sim *core.Simulation) {
			ticker = core.StartPeriodicAction(sim, core.PeriodicActionOptions{
				Period:   manaTideTotemTickGap,
				NumTicks: manaTideTotemTicks,
				OnAction: func(sim *core.Simulation) {
					for i, member := range members {
						if metrics[i] != nil {
							member.GetCharacter().AddMana(sim, manaPerTick, metrics[i])
						}
					}
				},
			})
		},
		OnExpire: func(_ *core.Aura, sim *core.Simulation) {
			ticker.Cancel(sim)
		},
	})

	spell := shaman.newTotemSpellConfig(ManaTideTotemManaCost[rank], actionID.SpellID)
	spell.RequiredLevel = ManaTideTotemLevel[rank]
	spell.Rank = rank
	spell.RelatedSelfBuff = aura
	spell.Cast.CD = cooldown
	spell.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
		shaman.dropWaterTotem(sim, spell, manaTideTotemDuration, aura.Deactivate)
		aura.Activate(sim)
	}
	return spell
}
