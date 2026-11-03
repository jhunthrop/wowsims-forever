package core

import (
	"time"
)

const CharacterMaxLevel = 60

// EffectiveCharacterLevel resolves a proto.Player.Level value the way
// NewCharacter builds a character: 1..CharacterMaxLevel is used as given,
// and anything else - 0 (the field's zero value, so every request that
// predates this field, and every hand-built test fixture, still means
// "unset"), negative, or above CharacterMaxLevel - becomes
// CharacterMaxLevel, the sim's long-standing default. Every level-aware
// lookup in this package (base stats, attack power, spell crit per
// intellect) resolves through this so an out-of-range or zero-value level
// never becomes a table miss.
func EffectiveCharacterLevel(level int32) int32 {
	if level < 1 || level > CharacterMaxLevel {
		return CharacterMaxLevel
	}
	return level
}

const GCDMin = time.Second * 1
const GCDDefault = time.Millisecond * 1500
const SpellBatchWindow = time.Millisecond * 10

const DefaultAttackPowerPerDPS = 14.0
const ArmorPenPerPercentArmor = 13.99

const MaxMeleeAttackDistance = 5
const MinRangedAttackDistance = 12

const MissDodgeParryBlockCritChancePerDefense = 0.04

const DefenseRatingToChanceReduction = (1.0 / DefenseRatingPerDefense) * MissDodgeParryBlockCritChancePerDefense / 100

// EnemyAutoAttackAPCoefficient is the share of a creature's rolled weapon
// damage each point of its attack power adds: a swing is weapon damage x
// (1 + attack power x 0.15873%), so a level 63 boss's ~270 attack power is
// about 30% of its swing and Demoralizing Shout takes a share of the whole
// swing away. Classic testing ("Boss Attacks and You", Blizzard Classic
// forums, 2020-03-03) states the coefficient; vmangos
// Creature::UpdateDamagePhysical (weapon damage x (0.7 + 0.3 x AP / database
// AP), level 63 warrior-class database AP 268) gives the same slope. The
// Wrath fork's 1 / (14 x 177) of the minimum roll made 270 attack power
// under 9% of the swing.
const EnemyAutoAttackAPCoefficient = 1.0 / 630.0

const AverageMagicPartialResistPerLevelMultiplier = 0.02

// IDs for items used in core
const (
	ItemIDBraidedEterniumChain  = 24114
	ItemIDChainOfTheTwilightOwl = 24121
	ItemIDEyeOfTheNight         = 24116
	ItemIDJadePendantOfBlasting = 20966
	ItemIDTheLightningCapacitor = 28785
)

type Hand bool

const MainHand Hand = true
const OffHand Hand = false

type DefenseType byte

const (
	DefenseTypeNone DefenseType = iota
	DefenseTypeMagic
	DefenseTypeMelee
	DefenseTypeRanged

	DefenseTypeLen
)
