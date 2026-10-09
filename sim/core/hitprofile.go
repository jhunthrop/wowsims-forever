package core

import (
	"errors"
	"math"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// hitStepPoints is the half-width, in hit points (one point is one percent
// of hit chance, HitRatingPerHitChance being 1), of the stat-weight sweep's
// hit step.
const hitStepPoints = 1.0

// HitProfile is where a physical attacker's hit stands against its first
// target, in the engine's own units (percent points of hit chance).
//
// The DPS-versus-hit curve is flat below Suppression (PhysicalHitChance
// floors at max(hit - HitSuppression, 0)), rises with full slope up to
// SpecialCap, where special attacks stop missing, keeps a smaller slope
// from white swings alone up to WhiteCap (the dual wield miss penalty
// moves white swings' cap 19 points past the specials'), and is flat above.
type HitProfile struct {
	// Physical is true when the player auto-attacks with a weapon (melee
	// or ranged). A caster's hit has no suppression floor and its cap
	// depends on school bonuses the profile does not model.
	Physical bool
	// Hit is the player's total hit, in percent points.
	Hit float64
	// Suppression is the dead zone below which hit does nothing.
	Suppression float64
	// SpecialCap and WhiteCap are the hit values (percent points) at which
	// special attacks, and white swings, can no longer miss.
	SpecialCap, WhiteCap float64
	// Spell is true when the player casts damaging spells at its target.
	// Spell hit is a separate table from the weapon's: see SpellProfile.
	Spell bool
	// SpellProfile is where the player's spell hit stands, set when Spell is.
	SpellProfile SpellHitProfile
	// DualWielding is true when white swings carry the dual wield miss
	// penalty, so WhiteCap is above SpecialCap.
	DualWielding bool
	// Melee is true when the player auto-attacks in melee, the only swing a
	// target can dodge or parry. The three fields below are set only then.
	Melee bool
	// Expertise is the player's total expertise, in percent points: the
	// dodge and parry chance its melee attacks lose.
	Expertise float64
	// DodgeChance and ParryChance are the target's chances (percent
	// points) to dodge and to parry this attacker, before Expertise. The
	// dodge chance is net of the target's own dodge reduction. A target
	// only parries an attacker it faces: a tank's figure applies, a damage
	// dealer standing behind the target does not meet it.
	DodgeChance, ParryChance float64
}

// ToDodgeCap is the expertise still worth anything against the target's dodge.
func (p HitProfile) ToDodgeCap() float64 { return math.Max(p.DodgeChance-p.Expertise, 0) }

// ToParryCap is the expertise still worth anything against the target's parry.
func (p HitProfile) ToParryCap() float64 { return math.Max(p.ParryChance-p.Expertise, 0) }

// ToSpecialCap is the hit still worth full value to special attacks.
func (p HitProfile) ToSpecialCap() float64 { return math.Max(p.SpecialCap-p.Hit, 0) }

// ToWhiteCap is the hit still worth anything to white swings.
func (p HitProfile) ToWhiteCap() float64 { return math.Max(p.WhiteCap-p.Hit, 0) }

// hitStepWindow returns the hit offsets, relative to the player's current
// hit, between which the sweep measures hit's marginal DPS value: a two
// sided step where neither side is floored or capped, the live part of it
// otherwise. ok is false where the stat is worth nothing (past the cap),
// so the sweep leaves it unmeasured instead of reporting noise.
func hitStepWindow(p HitProfile) (low, high float64, ok bool) {
	if !p.Physical {
		return -hitStepPoints, hitStepPoints, true
	}
	if p.Hit > p.WhiteCap {
		return 0, 0, false
	}
	lowEdge := math.Max(p.Hit-hitStepPoints, p.Suppression)
	highEdge := math.Min(p.Hit+hitStepPoints, p.WhiteCap)
	if p.Hit < p.Suppression {
		highEdge = math.Min(p.Suppression+hitStepPoints, p.WhiteCap)
	}
	if highEdge <= lowEdge {
		return 0, 0, false
	}
	return lowEdge - p.Hit, highEdge - p.Hit, true
}

var errNoHitProfilePlayer = errors.New("the request has no player to profile")

// ComputeHitProfile builds the request's environment, without simulating,
// and reads the player's hit and the miss table it faces. Hit comes from
// the request as the sweep will run it: gear, talents, buffs and bonus
// stats included.
func ComputeHitProfile(swr *proto.StatWeightsRequest) (HitProfile, error) {
	if swr.GetPlayer().GetSpec() == nil {
		return HitProfile{}, errNoHitProfilePlayer
	}
	raid := SinglePlayerRaidProto(swr.Player, swr.PartyBuffs, swr.RaidBuffs, swr.Debuffs)
	env, _, _ := NewEnvironment(raid, swr.Encounter, false)
	character := env.Raid.Parties[0].Players[0].GetCharacter()
	profile := HitProfile{Hit: character.GetStat(stats.Hit) / HitRatingPerHitChance}
	if len(env.Encounter.TargetUnits) > 0 {
		profile.SpellProfile, profile.Spell = computeSpellHitProfile(&character.Unit, env.Encounter.TargetUnits[0])
	}

	aa := character.AutoAttacks
	profile.Physical = aa.AutoSwingMelee || aa.AutoSwingRanged
	if !profile.Physical || len(env.Encounter.TargetUnits) == 0 {
		return profile, nil
	}
	slot := proto.CastType_CastTypeMainHand
	if aa.AutoSwingRanged && !aa.AutoSwingMelee {
		slot = proto.CastType_CastTypeRanged
	}
	table := character.AttackTables[env.Encounter.TargetUnits[0].UnitIndex][slot]
	if table == nil {
		return profile, nil
	}
	profile.Melee = aa.AutoSwingMelee
	if profile.Melee {
		profile.Expertise = character.GetStat(stats.Expertise)
		profile.DodgeChance = math.Max(table.BaseDodgeChance-env.Encounter.TargetUnits[0].PseudoStats.DodgeReduction, 0) * 100
		profile.ParryChance = table.BaseParryChance * 100
	}
	profile.Suppression = table.HitSuppression * 100
	profile.SpecialCap = (table.BaseMissChance + table.HitSuppression) * 100
	profile.WhiteCap = profile.SpecialCap
	if aa.IsDualWielding && !character.PseudoStats.DisableDWMissPenalty {
		profile.DualWielding = true
		profile.WhiteCap += activeAttackTable.DualWieldMissPenalty * 100
	}
	return profile, nil
}
