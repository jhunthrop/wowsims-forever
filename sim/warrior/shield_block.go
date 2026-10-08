package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Shield Block (client spell 2565): "+75" block chance (aura 51) for
// 7000 ms. The two charges are the Forever Deep Dive's "Shield Block has
// 2 charges and 7 s baseline" (research/06-since-announcement.md,
// single-source stream transcription): the client table carries no charge
// column, so the count is unconfirmed. They are the vanilla Improved
// Shield Block talent's full-rank values (+2 sec, +1 charge) made baseline,
// which is the same two numbers.
const (
	shieldBlockDuration      = 7 * time.Second
	shieldBlockCharges       = 2
	shieldBlockChancePercent = 75
)

func (warrior *Warrior) RegisterShieldBlockCD() {
	actionID := core.ActionID{SpellID: ShieldBlockSpellId[0]}
	cooldownDur := time.Duration(ShieldBlockCooldownMS[0]) * time.Millisecond

	warrior.ShieldBlockAura = warrior.RegisterAura(core.Aura{
		Label:    "Shield Block",
		ActionID: actionID,
		// Improved Shield Block is not in the live tree: its duration and
		// charge are baseline (see shieldBlockCharges).
		Duration:  shieldBlockDuration,
		MaxStacks: shieldBlockCharges,

		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			warrior.AddStatDynamic(sim, stats.Block, shieldBlockChancePercent*core.BlockRatingPerBlockChance)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			warrior.AddStatDynamic(sim, stats.Block, -shieldBlockChancePercent*core.BlockRatingPerBlockChance)
		},
		OnSpellHitTaken: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.DidBlock() {
				aura.RemoveStack(sim)
			}
		},
	})

	warrior.ShieldBlock = warrior.RegisterSpell(DefensiveStance, core.SpellConfig{
		ActionID:    actionID,
		SpellSchool: core.SpellSchoolPhysical,

		RequiredLevel: ShieldBlockLevel[0],

		RageCost: core.RageCostOptions{
			Cost: rageCost(ShieldBlockManaCost[0]),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{},
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: cooldownDur,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.PseudoStats.CanBlock
		},

		RelatedSelfBuff: warrior.ShieldBlockAura,

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			warrior.ShieldBlockAura.Activate(sim)
			// A recast while the aura is still up restores the spent charge.
			warrior.ShieldBlockAura.SetStacks(sim, warrior.ShieldBlockAura.MaxStacks)
		},
	})

	warrior.AddMajorCooldown(core.MajorCooldown{
		Spell:    warrior.ShieldBlock.Spell,
		Priority: core.CooldownPriorityDefault,
		Type:     core.CooldownTypeSurvival,
		ShouldActivate: func(s *core.Simulation, c *core.Character) bool {
			// Only castable with manual APL Action
			return false
		},
	})
}
