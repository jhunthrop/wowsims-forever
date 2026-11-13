package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Technically 4 different items
const ManaGemRanks = 4

var ManaGemItemId = [ManaGemRanks + 1]int32{0, 5514, 5513, 8007, 8008}
var ManaGemManaRestored = [ManaGemRanks + 1][]float64{{0, 0}, {375, 425}, {550, 650}, {775, 925}, {1000, 1200}}

// ManaGemLevel is the level the trainer teaches each Conjure spell
// (trainables 759 Mana Agate 28, 3552 Mana Jade 38, 10053 Mana Citrine 48,
// 10054 Mana Ruby 58, client 1.60.1.70790): a mage has a gem only once it
// can make one. The Mana Agate item (5514) says required level 23 where
// the other three gems say their conjure level; reading the item here gave
// a mage at 23 to 27 a gem five levels before the spell.
var ManaGemLevel = [ManaGemRanks + 1]int{0, 28, 38, 48, 58}

// The four gems and the runes share one item category (ItemEffect
// SpellCategoryID 1153, 120000 ms), so every use of one holds all of them
// for two minutes and a 180 s raid fight fits two. A mage spends its
// biggest gem first and a smaller one only once every bigger gem is gone;
// with a Demonic Rune (900 to 1500) in the conjured slot, which restores
// more than every gem below the Ruby and is there for every use, the
// gems below the biggest stay in the bag.
func (mage *Mage) manaGemWaits(rank int) bool {
	for bigger := rank + 1; bigger <= ManaGemRanks; bigger++ {
		if mage.ManaGem[bigger] != nil && mage.manaGemsLeft[bigger] > 0 {
			return true
		}
	}
	return rank < mage.biggestManaGem() && mage.Consumes.DefaultConjured == proto.Conjured_ConjuredDemonicRune
}

// biggestManaGem is the highest gem rank the mage can make at its level.
func (mage *Mage) biggestManaGem() int {
	for rank := ManaGemRanks; rank >= 1; rank-- {
		if mage.ManaGem[rank] != nil {
			return rank
		}
	}
	return 0
}

func (mage *Mage) registerManaGemCD() {
	mage.ManaGem = make([]*core.Spell, ManaGemRanks+1)

	for rank := 1; rank <= ManaGemRanks; rank++ {
		config := mage.newManaGemCooldown(rank)

		if config.RequiredLevel <= int(mage.Level) {
			mage.ManaGem[rank] = mage.RegisterSpell(config)
			minMana, maxMana := ManaGemManaRestored[rank][0], ManaGemManaRestored[rank][1]

			mage.AddMajorCooldown(core.MajorCooldown{
				Spell:    mage.ManaGem[rank],
				Priority: int32(minMana),
				Type:     core.CooldownTypeMana,
				ShouldActivate: func(sim *core.Simulation, character *core.Character) bool {
					if mage.manaGemWaits(rank) {
						return false
					}
					// Only pop if we have less than the max mana provided by the potion minus 1mp5 tick.
					totalRegen := character.ManaRegenPerSecondWhileCasting() * 2
					return (character.MaxMana()-(character.CurrentMana()+totalRegen) >= maxMana)
				},
			})
		}
	}
}

func (mage *Mage) newManaGemCooldown(rank int) core.SpellConfig {
	itemID := ManaGemItemId[rank]
	manaRestoredLow := ManaGemManaRestored[rank][0]
	manaRestoredHigh := ManaGemManaRestored[rank][1]

	actionID := core.ActionID{ItemID: itemID}
	manaMetrics := mage.NewManaMetrics(actionID)

	mage.RegisterResetEffect(func(sim *core.Simulation) {
		mage.manaGemsLeft[rank] = 1
	})

	return core.SpellConfig{
		ActionID:       actionID,
		ClassSpellMask: MageSpellMaskManaGem,
		Flags:          core.SpellFlagNoOnCastComplete | core.SpellFlagAPL,

		RequiredLevel: ManaGemLevel[rank],

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    mage.GetConjuredCD(),
				Duration: time.Minute * 2,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			mage.AddMana(sim, sim.Roll(manaRestoredLow, manaRestoredHigh), manaMetrics)

			mage.manaGemsLeft[rank]--
			if mage.manaGemsLeft[rank] == 0 {
				// Disable this cooldown since we're out of emeralds.
				mage.GetMajorCooldown(actionID).Disable()
			}
		},

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return mage.manaGemsLeft[rank] != 0
		},
	}
}
