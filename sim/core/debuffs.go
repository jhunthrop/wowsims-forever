package core

import (
	"math"
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

type DebuffName int32

const (
	// General Buffs
	DemoralizingShout DebuffName = iota
)

// applyDebuffEffects puts the raid's debuffs on target. Forever made Shadow
// Weaving (15258), Improved Scorch (22959), Improved Shadow Bolt (17794,
// all aura 270: damage taken from the caster) and Stormstrike (17364, aura
// 271 on the shaman's own spells) the caster's own, so another raid
// member's copy gives this player nothing: those flags apply no aura, and
// a priest, mage, warlock or shaman gets its own from its talents. Forever
// removed Curse of Shadow (17862 and 17937 have no SpellName or SpellEffect
// row; Curse of the Elements covers every magic school), so its flag applies
// nothing either.
func applyDebuffEffects(target *Unit, targetIdx int, debuffs *proto.Debuffs, raid *proto.Raid) {
	if debuffs.JudgementOfWisdom && targetIdx == 0 {
		MakePermanent(JudgementOfWisdomAura(target))
	}

	if debuffs.JudgementOfLight && targetIdx == 0 {
		MakePermanent(JudgementOfLightAura(target))
	}

	if targetIdx == 0 && debuffs.JudgementOfTheCrusader != proto.TristateEffect_TristateEffectMissing {
		// The raid's paladin judges the Seal of the Crusader rank the raid's
		// level has learned; none before level 6.
		if rank, ok := raidDebuffRank(JudgementOfTheCrusaderRanks, target); ok {
			mult := TernaryFloat64(debuffs.JudgementOfTheCrusader == proto.TristateEffect_TristateEffectImproved, 1.15, 1)
			MakePermanent(JudgementOfTheCrusaderAura(nil, target, rank.Amount, mult, 0))
		}
	}

	if debuffs.CurseOfElements {
		MakePermanent(CurseOfElementsAura(target))
	}

	// debuffs.WintersChill gives the raid nothing: Forever's Winter's Chill
	// (12579, aura 271) raises only the applying mage's own Cone of Cold, and
	// a Frost mage applies its own stacks (sim/mage/talents.go).

	if debuffs.GiftOfArthas {
		MakePermanent(GiftOfArthasAura(target))
	}

	/* if debuffs.Mangle {
		MakePermanent(MangleAura(target, level))
	} */

	if debuffs.CrystalYield {
		MakePermanent(CrystalYieldAura(target))
	}

	// Major Armor Debuffs
	if targetIdx == 0 {
		if debuffs.ExposeArmor != proto.TristateEffect_TristateEffectMissing {
			// Regular and improved remove the same armor: Forever's Improved
			// Expose Armor is energy and combo points, not armor.
			aura := ExposeArmorAura(target)
			SchedulePeriodicDebuffApplication(aura, PeriodicActionOptions{
				Period:   time.Second * 3,
				NumTicks: 1,
				OnAction: func(sim *Simulation) {
					aura.Activate(sim)
				},
			}, raid)
		}

		if debuffs.SunderArmor {
			// Sunder Armor
			aura := SunderArmorAura(target)
			SchedulePeriodicDebuffApplication(aura, PeriodicActionOptions{
				Period:          time.Millisecond * 200,
				NumTicks:        5,
				TickImmediately: true,
				Priority:        ActionPriorityDOT, // High prio so it comes before actual warrior sunders.
				OnAction: func(sim *Simulation) {
					aura.Activate(sim)
					if aura.IsActive() {
						aura.AddStack(sim)
					}
				},
			}, raid)
		}
	}

	if debuffs.CurseOfRecklessness {
		MakePermanent(CurseOfRecklessnessAura(target))
	}

	if debuffs.FaerieFire {
		MakePermanent(FaerieFireAura(target))
	}

	if debuffs.CurseOfWeakness != proto.TristateEffect_TristateEffectMissing {
		MakePermanent(CurseOfWeaknessAura(target, GetTristateValueInt32(debuffs.CurseOfWeakness, 0, 3)))
	}

	if debuffs.DemoralizingRoar != proto.TristateEffect_TristateEffectMissing {
		MakePermanent(DemoralizingRoarAura(target, GetTristateValueInt32(debuffs.DemoralizingRoar, 0, 5)))
	}
	if debuffs.DemoralizingShout != proto.TristateEffect_TristateEffectMissing {
		MakePermanent(DemoralizingShoutAura(target, 0, GetTristateValueInt32(debuffs.DemoralizingShout, 0, 5)))
	}
	if debuffs.HuntersMark != proto.TristateEffect_TristateEffectMissing {
		MakePermanent(HuntersMarkAura(target, GetTristateValueInt32(debuffs.HuntersMark, 0, 5)))
	}

	// Atk spd reduction
	if debuffs.ThunderClap != proto.TristateEffect_TristateEffectMissing {
		// Improved is a raid warrior with Conqueror's Battlegear 5-piece.
		MakePermanent(ThunderClapAura(target, ThunderClapRaidSpellID,
			GetTristateValueInt32(debuffs.ThunderClap, ThunderClapSlowPercent, ThunderClapConquerorSlowPercent)))
	}
	if debuffs.Thunderfury {
		MakePermanent(ThunderfuryASAura(target))
	}

	// Miss
	if debuffs.InsectSwarm && targetIdx == 0 {
		MakePermanent(InsectSwarmAura(target))
	}
	if debuffs.ScorpidSting && targetIdx == 0 {
		MakePermanent(ScorpidStingAura(target))
	}
}

type StormstrikeConfig struct {
	stormstrikeFrequency     float64
	natureAttackersFrequency float64
}

func (character *Character) createStormstrikeConfig(player *proto.Player) {
	character.StormstrikeConfig = StormstrikeConfig{
		stormstrikeFrequency:     player.StormstrikeFrequency,
		natureAttackersFrequency: player.StormstrikeNatureAttackerFrequency,
	}
	// Defaults if not configured
	if character.StormstrikeConfig.stormstrikeFrequency == 0.0 {
		character.StormstrikeConfig.stormstrikeFrequency = 20.0
	}
}

type IsbConfig struct {
	shadowBoltFrequency float64
	casterCrit          float64
	isbWarlocks         int32
}

func (character *Character) createIsbConfig(player *proto.Player) {
	character.IsbConfig = IsbConfig{
		shadowBoltFrequency: player.IsbSbFrequency,
		casterCrit:          player.IsbCrit,
		isbWarlocks:         player.IsbWarlocks,
	}
	//Defaults if not configured
	if character.IsbConfig.shadowBoltFrequency == 0.0 {
		character.IsbConfig.shadowBoltFrequency = 3.0
	}
	if character.IsbConfig.casterCrit == 0.0 {
		character.IsbConfig.casterCrit = 25.0
	}
	if character.IsbConfig.isbWarlocks == 0 {
		character.IsbConfig.isbWarlocks = 1
	}
}

// ImprovedShadowBoltDuration is the Shadow Vulnerability debuff's length:
// SpellDuration index 29, 12000 ms, on spell 17794 (SpellMisc). The spell
// has no SpellAuraOptions row, so it carries no proc charges: the bonus
// lasts the full duration however many spells land.
const ImprovedShadowBoltDuration = 12 * time.Second

var ShadowWeavingSpellIDs = [6]int32{0, 15257, 15331, 15332, 15333, 15334}

func SchedulePeriodicDebuffApplication(aura *Aura, options PeriodicActionOptions, _ *proto.Raid) {
	aura.OnReset = func(aura *Aura, sim *Simulation) {
		aura.Duration = NeverExpires
		StartPeriodicAction(sim, options)
	}
}

const JudgementAuraTag = "Judgement"

// JudgementDuration is the length of every judgement debuff in the
// 1.60.1.70009 client (duration index 31, 40000 ms, on spells 20185 to
// 20355 and the Crusader's); vanilla remembered 10 seconds.
const JudgementDuration = 40 * time.Second

// JudgementProcChance is the chance a qualifying attack triggers a
// Judgement of Wisdom or Light. Assumption, not a client row: the auras
// state ProcChance 100 and the roll lives in the server's script (the
// intermediate spells 1826 and 5373 are Effect 3, dummy), so this is
// vanilla's 50%. The Improved Judgement of Light talent (23564) adds to it.
const JudgementProcChance = 0.5

// Judgement of Wisdom and Light amounts by the judging paladin's level.
// SpellID is the judgement aura; Amount is the mana (Wisdom, energize
// spells 20268 / 20352 / 20353) or the health (Light, heal spells 20267 /
// 20341 / 20342 / 20343) one trigger returns. The raid-debuff form has no
// paladin, so the rank is the one the attacker's own level could judge.
var (
	JudgementOfWisdomRanks = BuffRanks{
		{SpellID: 20186, Level: 38, Amount: 33},
		{SpellID: 20354, Level: 48, Amount: 46},
		{SpellID: 20355, Level: 58, Amount: 59},
	}
	JudgementOfLightRanks = BuffRanks{
		{SpellID: 20185, Level: 30, Amount: 25},
		{SpellID: 20344, Level: 40, Amount: 34},
		{SpellID: 20345, Level: 50, Amount: 49},
		{SpellID: 20346, Level: 60, Amount: 61},
	}
)

// JudgementOfWisdomAura is the debuff on the target: an attack or spell
// used against it has a chance to return mana to the attacker.
func JudgementOfWisdomAura(target *Unit) *Aura {
	actionID := ActionID{SpellID: 20355}

	return target.GetOrRegisterAura(Aura{
		Label:    "Judgement of Wisdom",
		ActionID: actionID,
		Tag:      JudgementAuraTag,
		Duration: JudgementDuration,
		OnSpellHitTaken: func(aura *Aura, sim *Simulation, spell *Spell, result *SpellResult) {
			unit := spell.Unit
			if !unit.HasManaBar() {
				return
			}

			if spell.ProcMask.Matches(ProcMaskEmpty|ProcMaskProc|ProcMaskSpellDamageProc) && !spell.Flags.Matches(SpellFlagNotAProc) {
				return // Phantom spells (Romulo's, Lightning Capacitor, etc.) don't proc JoW.
			}

			if !spell.ProcMask.Matches(ProcMaskDirect) {
				return
			}

			// melee auto attacks don't even need to land
			if !result.Landed() && !spell.ProcMask.Matches(ProcMaskMeleeWhiteHit) {
				return
			}

			if sim.RandomFloat("jow") < JudgementProcChance {
				if unit.JowManaMetrics == nil {
					unit.JowManaMetrics = unit.NewManaMetrics(actionID)
				}
				unit.AddMana(sim, JudgementOfWisdomRanks.At(int(unit.Level)), unit.JowManaMetrics)
			}
		},
	})
}

// JudgementOfLightAura is the debuff on the target: a melee attack that
// lands on it has a chance to heal the attacker.
func JudgementOfLightAura(target *Unit) *Aura {
	actionID := ActionID{SpellID: 20346}

	return target.GetOrRegisterAura(Aura{
		Label:    "Judgement of Light",
		ActionID: actionID,
		Tag:      JudgementAuraTag,
		Duration: JudgementDuration,
		OnSpellHitTaken: func(aura *Aura, sim *Simulation, spell *Spell, result *SpellResult) {
			unit := spell.Unit
			if !unit.HasHealthBar() || !spell.ProcMask.Matches(ProcMaskMelee) || !result.Landed() {
				return
			}
			// A heal on a unit at full health changes nothing, so it is
			// neither rolled nor recorded: a damage sim whose melee attackers
			// are never hurt draws no random numbers for this debuff.
			if unit.CurrentHealth() >= unit.MaxHealth() {
				return
			}
			if sim.RandomFloat("jol") < JudgementProcChance {
				if unit.JolHealthMetrics == nil {
					unit.JolHealthMetrics = unit.NewHealthMetrics(actionID)
				}
				unit.GainHealth(sim, JudgementOfLightRanks.At(int(unit.Level)), unit.JolHealthMetrics)
			}
		},
	})
}

// JudgementOfTheCrusaderRaidBonus is the raid debuff's Holy damage taken:
// rank 6 (20303), aura 14 at 161 in the Forever client (Era's was 140).
const JudgementOfTheCrusaderRaidBonus = 161

// JudgementOfTheCrusaderAura adds bonus Holy damage taken (the judging
// Seal rank's amount), scaled by mult and raised by extraBonus.
func JudgementOfTheCrusaderAura(caster *Unit, target *Unit, bonus float64, mult float64, extraBonus float64) *Aura {
	var spellId int32 = 20303

	bonus *= mult
	bonus += extraBonus

	return target.GetOrRegisterAura(Aura{
		Label:    "Judgement of the Crusader",
		ActionID: ActionID{SpellID: spellId},
		Tag:      JudgementAuraTag,
		// The client gives Judgement of the Crusader a 40 s debuff (spellconst
		// duration_ms 40000 at every rank); vanilla remembered 10 s.
		Duration: 40 * time.Second,

		OnGain: func(aura *Aura, sim *Simulation) {
			aura.Unit.PseudoStats.SchoolBonusDamageTaken[stats.SchoolIndexHoly] += bonus
		},
		OnExpire: func(aura *Aura, sim *Simulation) {
			aura.Unit.PseudoStats.SchoolBonusDamageTaken[stats.SchoolIndexHoly] -= bonus
		},
		OnSpellHitTaken: func(aura *Aura, sim *Simulation, spell *Spell, result *SpellResult) {
			if spell.Unit != caster { // caster is nil for permanent auras
				return
			}
			if result.Landed() && spell.ProcMask.Matches(ProcMaskMelee) {
				aura.Refresh(sim)
			}
		},
	})
}

// CurseOfElementsAura is the raid-debuff form of the Forever curse at the
// rank the raid's level has learned (CurseOfTheElementsDamageRanks): at 50
// and above (1311680) every magic school takes 10% more damage and 75
// resistance is removed (aura 87 and aura 22 over misc value 126).
func CurseOfElementsAura(target *Unit) *Aura {
	damage, _ := raidDebuffRank(CurseOfTheElementsDamageRanks, target)
	resist, _ := raidDebuffRank(CurseOfTheElementsResistanceRanks, target)
	resistance := resist.Amount
	dmgMod := 1 + damage.Amount/100

	aura := target.GetOrRegisterAura(Aura{
		Label: "Curse of Elements",
		// Not the client's 1311680: the conformance report matches an aura on
		// the target by the spell's sibling ids, and this permanent one would
		// hide the castable curse's own five-minute duration.
		ActionID: ActionID{SpellID: 11722},
		Duration: time.Minute * 5,
	})
	CurseOfElementsEffects(aura, dmgMod, resistance)
	return aura
}

// curseOfElementsSchools are the magic schools Curse of the Elements' aura 87
// names (misc value 126).
var curseOfElementsSchools = []stats.SchoolIndex{
	stats.SchoolIndexArcane, stats.SchoolIndexFire, stats.SchoolIndexFrost,
	stats.SchoolIndexHoly, stats.SchoolIndexNature, stats.SchoolIndexShadow,
}

// CurseOfElementsEffects puts one Curse of the Elements rank on aura: every
// magic school takes damageTakenMultiplier and every school loses resistance.
// Every copy of the curse (the raid's debuff and a warlock's own rank) shares
// these exclusive categories, so the target carries only the strongest.
func CurseOfElementsEffects(aura *Aura, damageTakenMultiplier, resistance float64) {
	for _, school := range curseOfElementsSchools {
		spellSchoolDamageEffect(aura, school, damageTakenMultiplier, 0.0, false)
	}
	// One effect: it lowers every school's resistance at once.
	spellSchoolResistanceEffect(aura, stats.SchoolIndexFire, resistance, 0.0, false)
}

// CurseOfElementsOn is the aura whose Curse of the Elements is in force on
// target, or nil.
func CurseOfElementsOn(target *Unit) *Aura {
	category := target.ExclusiveEffectManager.GetExclusiveEffectCategory("spellDamage" + strconv.Itoa(int(stats.SchoolIndexShadow)))
	return category.GetActiveAura()
}

func CurseOfShadowAura(target *Unit) *Aura {
	resistance := 75.0
	dmgMod := 1.1

	aura := target.GetOrRegisterAura(Aura{
		Label:    "Curse of Shadow",
		ActionID: ActionID{SpellID: 17937},
		Duration: time.Minute * 5,
	})
	spellSchoolDamageEffect(aura, stats.SchoolIndexArcane, dmgMod, 0.0, false)
	spellSchoolDamageEffect(aura, stats.SchoolIndexShadow, dmgMod, 0.0, false)

	spellSchoolResistanceEffect(aura, stats.SchoolIndexArcane, resistance, 0.0, false)
	spellSchoolResistanceEffect(aura, stats.SchoolIndexShadow, resistance, 0.0, false)

	return aura
}

func spellSchoolDamageEffect(aura *Aura, school stats.SchoolIndex, multiplier float64, extraPriority float64, exclusive bool) *ExclusiveEffect {
	return aura.NewExclusiveEffect("spellDamage"+strconv.Itoa(int(school)), exclusive, ExclusiveEffect{
		Priority: multiplier + extraPriority,
		OnGain: func(ee *ExclusiveEffect, sim *Simulation) {
			ee.Aura.Unit.PseudoStats.SchoolDamageTakenMultiplier[school] *= multiplier
		},
		OnExpire: func(ee *ExclusiveEffect, sim *Simulation) {
			ee.Aura.Unit.PseudoStats.SchoolDamageTakenMultiplier[school] /= multiplier
		},
	})
}

func spellSchoolResistanceEffect(aura *Aura, school stats.SchoolIndex, amount float64, extraPriority float64, exclusive bool) *ExclusiveEffect {
	return aura.NewExclusiveEffect("resistance"+strconv.Itoa(int(school)), exclusive, ExclusiveEffect{
		Priority: amount + extraPriority,
		OnGain: func(ee *ExclusiveEffect, sim *Simulation) {
			aura.Unit.AddResistancesDynamic(sim, -amount)
		},
		OnExpire: func(ee *ExclusiveEffect, sim *Simulation) {
			aura.Unit.AddResistancesDynamic(sim, amount)
		},
	})
}

func GiftOfArthasAura(target *Unit) *Aura {
	return target.GetOrRegisterAura(Aura{
		Label:    "Gift of Arthas",
		ActionID: ActionID{SpellID: 11374},
		Duration: time.Minute * 3,
		OnGain: func(aura *Aura, sim *Simulation) {
			aura.Unit.PseudoStats.SchoolBonusDamageTaken[stats.SchoolIndexPhysical] += 8
		},
		OnExpire: func(aura *Aura, sim *Simulation) {
			aura.Unit.PseudoStats.SchoolBonusDamageTaken[stats.SchoolIndexPhysical] -= 8
		},
	})
}

func HemorrhageAura(target *Unit) *Aura {
	debuffBonusDamage := 7.0

	spellID := int32(17348)

	return target.GetOrRegisterAura(Aura{
		Label:     "Hemorrhage",
		ActionID:  ActionID{SpellID: spellID},
		Duration:  time.Second * 15,
		MaxStacks: 30,
		OnGain: func(aura *Aura, sim *Simulation) {
			aura.Unit.PseudoStats.SchoolBonusDamageTaken[stats.SchoolIndexPhysical] += debuffBonusDamage
		},
		OnExpire: func(aura *Aura, sim *Simulation) {
			aura.Unit.PseudoStats.SchoolBonusDamageTaken[stats.SchoolIndexPhysical] -= debuffBonusDamage
		},
		OnSpellHitTaken: func(aura *Aura, sim *Simulation, spell *Spell, result *SpellResult) {
			if spell.SpellSchool != SpellSchoolPhysical {
				return
			}
			if !result.Landed() || result.Damage == 0 {
				return
			}
			// TODO find out which abilities are actually affected
			aura.RemoveStack(sim)
		},
	})
}

func MangleAura(target *Unit) *Aura {
	return bleedDamageAura(target, Aura{
		Label:    "Mangle",
		ActionID: ActionID{SpellID: 409828},
		Duration: time.Minute,
	}, 1.3)
}

// Bleed Damage Multiplier category
const BleedEffectCategory = "BleedDamage"

func bleedDamageAura(target *Unit, config Aura, multiplier float64) *Aura {
	aura := target.GetOrRegisterAura(config)
	aura.NewExclusiveEffect(BleedEffectCategory, true, ExclusiveEffect{
		Priority: multiplier,
		OnGain: func(ee *ExclusiveEffect, sim *Simulation) {
			ee.Aura.Unit.PseudoStats.BleedDamageTakenMultiplier *= multiplier
		},
		OnExpire: func(ee *ExclusiveEffect, sim *Simulation) {
			ee.Aura.Unit.PseudoStats.BleedDamageTakenMultiplier /= multiplier
		},
	})
	return aura
}

const SpellFirePowerEffectCategory = "spellFirePowerdebuff"

func ImprovedScorchAura(target *Unit) *Aura {
	aura := target.GetOrRegisterAura(Aura{
		Label:     "Improved Scorch",
		ActionID:  ActionID{SpellID: 12873},
		Duration:  time.Second * 30,
		MaxStacks: 5,
		OnStacksChange: func(aura *Aura, sim *Simulation, oldStacks int32, newStacks int32) {
			aura.Unit.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexFire] /= 1 + .03*float64(oldStacks)
			aura.Unit.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexFire] *= 1 + .03*float64(newStacks)
		},
	})

	return aura
}

const SpellCritEffectCategory = "spellcritdebuff"

// WintersChillMaxStacks is the most stacks a Winter's Chill debuff holds.
const WintersChillMaxStacks = 5

// WintersChillAura is Forever's Winter's Chill (1.60.1.70532): a stack
// counter on the target that the applying mage's own next Cone of Cold
// reads and spends ("increasing the target's damage taken from your next
// Cone of Cold by 5% and it gains an additional 7% of your spell damage,
// stacking up to 5 times" - sim/mage/cone_of_cold.go). Vanilla's 2%
// Frost crit taken a stack is gone from the text, so the aura itself
// changes no number on the target. The id is the client's own debuff
// (12579, the spell the talent applies); rotations gate Cone of Cold on it.
func WintersChillAura(target *Unit) *Aura {
	aura := target.GetOrRegisterAura(Aura{
		Label:     "Winter's Chill",
		ActionID:  ActionID{SpellID: 12579},
		Duration:  time.Second * 15,
		MaxStacks: WintersChillMaxStacks,
	})

	// effect = aura.NewExclusiveEffect(SpellCritEffectCategory, true, ExclusiveEffect{
	// 	Priority: 0,
	// 	OnGain: func(ee *ExclusiveEffect, sim *Simulation) {
	// 		ee.Aura.Unit.PseudoStats.BonusSpellCritRatingTaken += ee.Priority * CritRatingPerCritChance
	// 	},
	// 	OnExpire: func(ee *ExclusiveEffect, sim *Simulation) {
	// 		ee.Aura.Unit.PseudoStats.BonusSpellCritRatingTaken -= ee.Priority * CritRatingPerCritChance
	// 	},
	// })
	return aura
}

var majorArmorReductionEffectCategory = "MajorArmorReduction"
var minorArmorReductionEffectCategory = "MinorArmorReduction"

// SunderArmorAura strips the armor of the rank the raid's level has
// learned (SunderArmorRanks) per stack.
func SunderArmorAura(target *Unit) *Aura {
	rank, _ := raidDebuffRank(SunderArmorRanks, target)
	arpen := rank.Amount

	var effect *ExclusiveEffect
	aura := target.GetOrRegisterAura(Aura{
		Label:     "Sunder Armor",
		ActionID:  ActionID{SpellID: rank.SpellID},
		Duration:  time.Second * 30,
		MaxStacks: 5,
		OnStacksChange: func(aura *Aura, sim *Simulation, oldStacks int32, newStacks int32) {
			effect.SetPriority(sim, arpen*float64(newStacks))
		},
	})

	effect = aura.NewExclusiveEffect(majorArmorReductionEffectCategory, true, ExclusiveEffect{
		Priority: 0,
		OnGain: func(ee *ExclusiveEffect, sim *Simulation) {
			aura.Unit.AddStatDynamic(sim, stats.Armor, -ee.Priority)
		},
		OnExpire: func(ee *ExclusiveEffect, sim *Simulation) {
			aura.Unit.AddStatDynamic(sim, stats.Armor, ee.Priority)
		},
	})

	return aura
}

// ExposeArmorAura is the shared Expose Armor debuff. The raid's copy is a
// five-point Expose Armor of the rank the raid's level has learned
// (ExposeArmorRanks); a rogue's own cast sets the priority from its combo
// points.
func ExposeArmorAura(target *Unit) *Aura {
	rank, _ := raidDebuffRank(ExposeArmorRanks, target)
	arpen := rank.Amount * exposeArmorRaidComboPoints

	aura := target.GetOrRegisterAura(Aura{
		Label:    "ExposeArmor",
		ActionID: ActionID{SpellID: rank.SpellID},
		Duration: time.Second * 30,
	})

	aura.NewExclusiveEffect(majorArmorReductionEffectCategory, true, ExclusiveEffect{
		Priority: arpen,
		OnGain: func(ee *ExclusiveEffect, sim *Simulation) {
			aura.Unit.AddStatDynamic(sim, stats.Armor, -ee.Priority)
		},
		OnExpire: func(ee *ExclusiveEffect, sim *Simulation) {
			aura.Unit.AddStatDynamic(sim, stats.Armor, ee.Priority)
		},
	})

	return aura
}

// CurseOfRecklessnessAura strips the armor of the rank the raid's level has
// learned (CurseOfRecklessnessRanks). Forever's curse gives the target no
// attack power (Era's added 90).
func CurseOfRecklessnessAura(target *Unit) *Aura {
	rank, _ := raidDebuffRank(CurseOfRecklessnessRanks, target)
	arpen := rank.Amount

	aura := target.GetOrRegisterAura(Aura{
		Label:    "Curse of Recklessness",
		ActionID: ActionID{SpellID: rank.SpellID},
		Duration: time.Minute * 2,
		OnGain: func(aura *Aura, sim *Simulation) {
			aura.Unit.AddStatDynamic(sim, stats.Armor, -arpen)
		},
		OnExpire: func(aura *Aura, sim *Simulation) {
			aura.Unit.AddStatDynamic(sim, stats.Armor, arpen)
		},
	})
	return aura
}

// Decreases the armor of the target by X for 40 sec.
// Improved: Your Faerie Fire and Faerie Fire (Feral) also increase the chance for all attacks to hit that target by 1% for 40 sec.
func FaerieFireAura(target *Unit) *Aura {
	return faerieFireAuraInternal(target, "Faerie Fire", 9907)
}

func FaerieFireFeralAura(target *Unit) *Aura {
	return faerieFireAuraInternal(target, "Faerie Fire (Feral)", 17392)
}

// faerieFireAuraInternal strips the armor of the Faerie Fire rank the raid's
// level has learned (FaerieFireRanks).
func faerieFireAuraInternal(target *Unit, label string, spellID int32) *Aura {
	rank, _ := raidDebuffRank(FaerieFireRanks, target)
	return MinorArmorReductionAura(target, label, spellID, rank.Amount, time.Second*40)
}

// MinorArmorReductionAura is an armor reduction in the minor category,
// the one Faerie Fire takes: of the sources in it only the strongest
// applies, and it stacks with the major category (Sunder Armor, Expose
// Armor).
func MinorArmorReductionAura(target *Unit, label string, spellID int32, armorReduction float64, duration time.Duration) *Aura {
	aura := target.GetOrRegisterAura(Aura{
		Label:    label,
		ActionID: ActionID{SpellID: spellID},
		Duration: duration,
	})

	aura.NewExclusiveEffect(minorArmorReductionEffectCategory, true, ExclusiveEffect{
		Priority: armorReduction,
		OnGain: func(ee *ExclusiveEffect, sim *Simulation) {
			ee.Aura.Unit.AddStatDynamic(sim, stats.Armor, -armorReduction)
		},
		OnExpire: func(ee *ExclusiveEffect, sim *Simulation) {
			ee.Aura.Unit.AddStatDynamic(sim, stats.Armor, armorReduction)
		},
	})

	return aura
}

func CurseOfWeaknessAura(target *Unit, points int32) *Aura {
	modDmgReduction := -31.0

	modDmgReduction *= []float64{1, 1.06, 1.13, 1.20}[points]
	modDmgReduction = math.Floor(modDmgReduction)

	aura := target.GetOrRegisterAura(Aura{
		Label:    "Curse of Weakness" + strconv.Itoa(int(points)),
		ActionID: ActionID{SpellID: 11708},
		Duration: time.Minute * 2,
		OnGain: func(aura *Aura, sim *Simulation) {
			aura.Unit.PseudoStats.BonusPhysicalDamage += modDmgReduction
		},
		OnExpire: func(aura *Aura, sim *Simulation) {
			aura.Unit.PseudoStats.BonusPhysicalDamage -= modDmgReduction
		},
	})
	return aura
}

const HuntersMarkAuraTag = "HuntersMark"

// HuntersMarkTopRankSpellID and HuntersMarkTopRankRangedAttackPower are
// Hunter's Mark rank 4 as a trainer teaches it (SkillLineAbility row 7913)
// and the 1.60.1.70707 SpellEffect states it, matching Wowhead Forever's
// tooltip: 143 ranged attack power. The 110 used before is 1213268, a
// Season of Discovery copy of the rank that no trainer teaches.
const (
	HuntersMarkTopRankSpellID           int32   = 14325
	HuntersMarkTopRankRangedAttackPower float64 = 143
)

// HuntersMarkAura is the raid debuff: the rank the raid's level has learned
// (HuntersMarkRanks).
func HuntersMarkAura(target *Unit, points int32) *Aura {
	rank, _ := raidDebuffRank(HuntersMarkRanks, target)
	return HuntersMarkAuraOfRank(target, rank.SpellID, rank.Amount, points)
}

// HuntersMarkAuraOfRank is the debuff a hunter's own Hunter's Mark rank
// applies: spellID is that rank's spell, baseBonus its ranged attack power.
func HuntersMarkAuraOfRank(target *Unit, spellID int32, baseBonus float64, points int32) *Aura {
	bonus := baseBonus * (1 + 0.03*float64(points))

	aura := target.GetOrRegisterAura(Aura{
		Label:    "HuntersMark-" + strconv.Itoa(int(bonus)),
		Tag:      HuntersMarkAuraTag,
		ActionID: ActionID{SpellID: spellID},
		Duration: time.Minute * 2,
	})

	aura.NewExclusiveEffect("HuntersMark", true, ExclusiveEffect{
		Priority: bonus,
		OnGain: func(ee *ExclusiveEffect, sim *Simulation) {
			for _, unit := range sim.Environment.Raid.AllUnits {
				if rangedAttackTable := unit.AttackTables[target.UnitIndex][proto.CastType_CastTypeRanged]; rangedAttackTable != nil {
					rangedAttackTable.BonusAttackPowerTaken += bonus
				}
			}
		},
		OnExpire: func(ee *ExclusiveEffect, sim *Simulation) {
			for _, unit := range sim.Environment.Raid.AllUnits {
				if rangedAttackTable := unit.AttackTables[target.UnitIndex][proto.CastType_CastTypeRanged]; rangedAttackTable != nil {
					rangedAttackTable.BonusAttackPowerTaken -= bonus
				}
			}
		},
	})

	return aura
}

func ExposeWeaknessAura(target *Unit) *Aura {
	bonus := 450.0

	aura := target.GetOrRegisterAura(Aura{
		ActionID: ActionID{SpellID: 23577},
		Label:    "Expose Weakness",
		Duration: time.Second * 7,
		OnGain: func(aura *Aura, sim *Simulation) {
			for _, unit := range sim.Environment.Raid.AllUnits {
				if rangedAttackTable := unit.AttackTables[target.UnitIndex][proto.CastType_CastTypeRanged]; rangedAttackTable != nil {
					rangedAttackTable.BonusAttackPowerTaken += bonus
				}
			}
		},
		OnExpire: func(aura *Aura, sim *Simulation) {
			for _, unit := range sim.Environment.Raid.AllUnits {
				if rangedAttackTable := unit.AttackTables[target.UnitIndex][proto.CastType_CastTypeRanged]; rangedAttackTable != nil {
					rangedAttackTable.BonusAttackPowerTaken -= bonus
				}
			}
		},
	})

	return aura
}

func DemoralizingRoarAura(target *Unit, points int32) *Aura {
	baseAPReduction := 138.0

	aura := target.GetOrRegisterAura(Aura{
		Label:    "DemoralizingRoar-" + strconv.Itoa(int(points)),
		ActionID: ActionID{SpellID: 9898},
		Duration: time.Second * 30,
	})
	apReductionEffect(aura, math.Floor(baseAPReduction*(1+0.08*float64(points))))
	return aura
}

const DemoralizingShoutRanks = 5

var DemoralizingShoutSpellId = [DemoralizingShoutRanks + 1]int32{0, 1160, 6190, 11554, 11555, 11556}
var DemoralizingShoutBaseAP = [DemoralizingShoutRanks + 1]float64{0, 45, 56, 76, 111, 146}
var DemoralizingShoutLevel = [DemoralizingShoutRanks + 1]int{0, 14, 24, 34, 44, 54}

func DemoralizingShoutAura(target *Unit, boomingVoicePts int32, impDemoShoutPts int32) *Aura {
	rank := int32(5)
	return DemoralizingShoutAuraFor(target,
		"DemoralizingShout-"+strconv.Itoa(int(impDemoShoutPts)),
		DemoralizingShoutSpellId[rank],
		math.Floor(DemoralizingShoutBaseAP[rank]*(1+0.08*float64(impDemoShoutPts))),
		time.Duration(float64(time.Second*30)*(1+0.1*float64(boomingVoicePts))))
}

// DemoralizingShoutAuraFor is the attack power reduction a Demoralizing
// Shout puts on a target, for an explicit label, spell id, reduction and
// duration, so a caster that knows the client's own rank numbers (the
// warrior spec) is not tied to rank 5's vanilla ones above.
func DemoralizingShoutAuraFor(target *Unit, label string, spellID int32, apReduction float64, duration time.Duration) *Aura {
	aura := target.GetOrRegisterAura(Aura{
		Label:    label,
		ActionID: ActionID{SpellID: spellID},
		Duration: duration,
	})
	apReductionEffect(aura, apReduction)
	return aura
}

func VindicationAura(target *Unit, points int32, _ int32) *Aura {
	aura := target.GetOrRegisterAura(Aura{
		Label:    "Vindication",
		ActionID: ActionID{SpellID: 26016},
		Duration: time.Second * 10,
	})
	return aura
}

func apReductionEffect(aura *Aura, apReduction float64) *ExclusiveEffect {
	statReduction := stats.Stats{stats.AttackPower: -apReduction}
	return aura.NewExclusiveEffect("APReduction", false, ExclusiveEffect{
		Priority: apReduction,
		OnGain: func(ee *ExclusiveEffect, sim *Simulation) {
			ee.Aura.Unit.AddStatsDynamic(sim, statReduction)
		},
		OnExpire: func(ee *ExclusiveEffect, sim *Simulation) {
			ee.Aura.Unit.AddStatsDynamic(sim, statReduction.Invert())
		},
	})
}

// Thunder Clap's slow, the trained family (6343 to 11581, SkillLineAbility
// 6073 to 6078): aura 319 at -20 on every rank, "increasing the time
// between their attacks by 20%"; Conqueror's Battlegear 5-piece raises it
// by 50%. The raid debuff is a level-60 warrior's rank 6.
const (
	ThunderClapRaidSpellID          int32 = 11581
	ThunderClapSlowPercent          int32 = 20
	ThunderClapConquerorSlowPercent int32 = 30
)

func ThunderClapAura(target *Unit, spellID int32, atkSpeedReductionPercent int32) *Aura {
	aura := target.GetOrRegisterAura(Aura{
		Label:    "ThunderClap-" + strconv.Itoa(int(atkSpeedReductionPercent)),
		ActionID: ActionID{SpellID: spellID},
		Duration: time.Second * 30,
	})
	AtkSpeedReductionEffect(aura, 1+0.01*float64(atkSpeedReductionPercent))
	return aura
}

func ThunderfuryASAura(target *Unit) *Aura {
	aura := target.GetOrRegisterAura(Aura{
		Label:    "Thunderfury",
		ActionID: ActionID{SpellID: 21992},
		Duration: time.Second * 12,
	})
	AtkSpeedReductionEffect(aura, 1.2)
	return aura
}

func AtkSpeedReductionEffect(aura *Aura, speedMultiplier float64) *ExclusiveEffect {
	return aura.NewExclusiveEffect("AtkSpdReduction", false, ExclusiveEffect{
		Priority: speedMultiplier,
		OnGain: func(ee *ExclusiveEffect, sim *Simulation) {
			ee.Aura.Unit.MultiplyAttackSpeed(sim, 1/speedMultiplier)
		},
		OnExpire: func(ee *ExclusiveEffect, sim *Simulation) {
			ee.Aura.Unit.MultiplyAttackSpeed(sim, speedMultiplier)
		},
	})
}

func InsectSwarmAura(target *Unit) *Aura {
	aura := target.GetOrRegisterAura(Aura{
		Label:    "InsectSwarmMiss",
		ActionID: ActionID{SpellID: 24977},
		Duration: time.Second * 12,
	})
	increasedMissEffect(aura, 0.02)
	return aura
}

func ScorpidStingAura(target *Unit) *Aura {
	aura := target.GetOrRegisterAura(Aura{
		Label:    "Scorpid Sting",
		ActionID: ActionID{SpellID: 3043},
		Duration: time.Second * 20,
	})
	return aura
}

func increasedMissEffect(aura *Aura, increasedMissChance float64) *ExclusiveEffect {
	return aura.NewExclusiveEffect("IncreasedMiss", false, ExclusiveEffect{
		Priority: increasedMissChance,
		OnGain: func(ee *ExclusiveEffect, sim *Simulation) {
			ee.Aura.Unit.PseudoStats.IncreasedMissChance += increasedMissChance
		},
		OnExpire: func(ee *ExclusiveEffect, sim *Simulation) {
			ee.Aura.Unit.PseudoStats.IncreasedMissChance -= increasedMissChance
		},
	})
}

func CrystalYieldAura(target *Unit) *Aura {
	return target.GetOrRegisterAura(Aura{
		Label:    "Crystal Yield",
		ActionID: ActionID{SpellID: 15235},
		Duration: 2 * time.Minute,
		OnGain: func(aura *Aura, sim *Simulation) {
			aura.Unit.stats[stats.Armor] -= 200
		},
		OnExpire: func(aura *Aura, sim *Simulation) {
			aura.Unit.stats[stats.Armor] += 200
		},
	})
}
