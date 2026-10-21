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

// Updated based on formulas supplied by InDebt on WoWSims Discord
const EnemyAutoAttackAPCoefficient = 1.0 / (14.0 * 177.0)

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
