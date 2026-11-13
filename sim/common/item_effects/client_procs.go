package item_effects

// Item effects the published BiS sets wear that the engine did not
// implement (sim audit round 8, items lane). Every amount, school, period
// and cooldown is the 70707 client's own row: ItemEffect for the trigger and
// cooldown, SpellEffect for the amount (EffectBasePointsF as the centre,
// Variance as the whole width of the roll), SpellMisc for the school. Each
// one matches the server's tooltip on Wowhead Forever
// (nether.wowhead.com/forever/tooltip/spell/<id>, read 2026-11-02).
//
// The client states no proc rate for a "chance on hit": the item proc rate
// lives on the server. These take core.UnstatedProcsPerMinute, the rate the
// round-8 beta-log fits back for rows of this shape. Black Menace, Verigan's
// Fist and the other single-roll weapons live in common's clientWeaponProcs.

import (
	"time"

	"github.com/wowsims/classic/sim/common/itemhelpers"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

const (
	ThunderbrewsBootFlask      = 744
	GlovesOfHolyMight          = 867
	CloakOfFlames              = 3475
	SporidCape                 = 6629
	FireHardenedHauberk        = 6972
	BrutalHauberk              = 7133
	AegisOfTheScarletCommander = 7726
	ToxicRevenger              = 9453
	WallOfTheDead              = 1979
	SpireOfHakkar              = 10844
	DormantHeartOfTheMountain  = 249473
	MindtapTalisman            = 18371
	RazorGauntlets             = 18326
)

// Gloves of Holy Might's equip trigger, client 1.60.1.70938 spell 1330855:
// ProcChance 15 and ProcCategoryRecovery 20000.
const (
	glovesOfHolyMightTrigger      = 1330855
	glovesOfHolyMightProcChance   = 0.15
	glovesOfHolyMightProcRecovery = 20 * time.Second
)

// eternalPowerHeals are every rank of the four heals Dormant Heart of the
// Mountain names, from the 70707 trainables (spellranks.json): Greater Heal,
// Healing Touch, Healing Wave and Holy Light.
var eternalPowerHeals = map[int32]bool{
	2060: true, 10963: true, 10964: true, 10965: true, 25314: true,
	5185: true, 5186: true, 5187: true, 5188: true, 5189: true, 6778: true, 8903: true, 9758: true, 9888: true, 9889: true, 25297: true,
	331: true, 332: true, 547: true, 913: true, 939: true, 959: true, 8005: true, 10395: true, 10396: true, 25357: true,
	635: true, 639: true, 647: true, 1026: true, 1042: true, 3472: true, 10328: true, 10329: true, 25292: true,
}

// clientRoll is a SpellEffect row's roll: centre x (1 -/+ variance/2).
func clientRoll(sim *core.Simulation, centre, variance float64) float64 {
	if variance == 0 {
		return centre
	}
	return sim.Roll(centre*(1-variance/2), centre*(1+variance/2))
}

func init() {
	// 11790: "Deals 14 Nature damage every 3 sec to any enemy in an 8 yard
	// radius around the caster for 12 sec."
	itemhelpers.CreateWeaponProcSpell(ToxicRevenger, "Toxic Revenger", core.UnstatedProcsPerMinute, func(character *core.Character) *core.Spell {
		return character.GetOrRegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{SpellID: 11790},
			SpellSchool: core.SpellSchoolNature,
			DefenseType: core.DefenseTypeMagic,
			ProcMask:    core.ProcMaskEmpty,

			DamageMultiplier: 1,
			ThreatMultiplier: 1,

			Dot: core.DotConfig{
				Aura:          core.Aura{Label: "Poison Cloud (Toxic Revenger)"},
				TickLength:    3 * time.Second,
				NumberOfTicks: 4,
				OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
					dot.Snapshot(target, 14, isRollover)
				},
				OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
					dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
				},
			},

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				for _, aoeTarget := range sim.Encounter.TargetUnits {
					spell.Dot(aoeTarget).Apply(sim)
				}
			},
		})
	})

	// 1325229, "Smite your target for 101 to 123 Holy damage. Damage is
	// tripled against Undead." 112 with variance 0.2. Its
	// SpellCategories.DefenseType is 3 (ranged), so it rolls the ranged
	// table: melee hit and crit, double damage on a crit.
	//
	// The trigger is Blizzard's 9 November 2026 fix ("Gloves of Holy Might
	// now work as intended"), native in client 1.60.1.70938: ItemEffect
	// 238916 equips 1330855, aura 42 at ProcChance 15 with
	// ProcCategoryRecovery 20000 and ProcTypeMask 69636 (0x11004: a melee
	// auto attack, a harmful spell of no class, a harmful magic spell). The
	// site's 70790 row was a chance on any melee hit. A melee special or a
	// ranged attack never procs it.
	core.NewItemEffect(GlovesOfHolyMight, func(agent core.Agent) {
		character := agent.GetCharacter()
		smite := character.GetOrRegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{SpellID: 1325229},
			SpellSchool: core.SpellSchoolHoly,
			DefenseType: core.DefenseTypeRanged,
			ProcMask:    core.ProcMaskEmpty,
			Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,

			DamageMultiplier: 1,
			ThreatMultiplier: 1,

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				damage := clientRoll(sim, 112, 0.20000000298)
				if target.MobType == proto.MobType_MobTypeUndead {
					damage *= 3
				}
				spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeRangedHitAndCrit)
			},
		})
		core.MakeProcTriggerAura(&character.Unit, core.ProcTrigger{
			Name:              "Gloves of Holy Might Proc",
			ActionID:          core.ActionID{SpellID: glovesOfHolyMightTrigger},
			Callback:          core.CallbackOnSpellHitDealt,
			Outcome:           core.OutcomeLanded,
			ProcMask:          core.ProcMaskMeleeWhiteHit | core.ProcMaskSpellDamage,
			SpellFlagsExclude: core.SpellFlagSuppressEquipProcs,
			ProcChance:        glovesOfHolyMightProcChance,
			ICD:               glovesOfHolyMightProcRecovery,
			Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
				smite.Cast(sim, result.Target)
			},
		})
	})

	// ItemEffect 97235: 12257 on a 5 min cooldown. 12257 is aura 23, a
	// periodic trigger every 1 sec for 5 sec (duration 28), casting 12278
	// each second: a direct 75 Fire spell (DefenseType 1) that rolls spell
	// hit and crit on its own. "Deals 75 Fire damage every 1 sec for 5 sec
	// to all enemies in front of you."
	core.NewItemEffect(ThunderbrewsBootFlask, func(agent core.Agent) {
		character := agent.GetCharacter()
		breath := character.RegisterSpell(core.SpellConfig{
			ActionID:         core.ActionID{SpellID: 12278},
			SpellSchool:      core.SpellSchoolFire,
			DefenseType:      core.DefenseTypeMagic,
			ProcMask:         core.ProcMaskEmpty,
			Flags:            core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.CalcAndDealDamage(sim, target, 75, spell.OutcomeMagicHitAndCrit)
			},
		})
		spell := character.RegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{SpellID: 12257},
			SpellSchool: core.SpellSchoolFire,
			DefenseType: core.DefenseTypeMagic,
			ProcMask:    core.ProcMaskEmpty,
			Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagOffensiveEquipment,

			Cast: core.CastConfig{
				CD: core.Cooldown{Timer: character.NewTimer(), Duration: 5 * time.Minute},
			},

			Dot: core.DotConfig{
				Aura:          core.Aura{Label: "Breath of Fire"},
				TickLength:    time.Second,
				NumberOfTicks: 5,
				OnTick: func(sim *core.Simulation, target *core.Unit, _ *core.Dot) {
					breath.Cast(sim, target)
				},
			},

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				for _, aoeTarget := range sim.Encounter.TargetUnits {
					spell.Dot(aoeTarget).Apply(sim)
				}
			},
		})
		character.AddMajorCooldown(core.MajorCooldown{Spell: spell, Type: core.CooldownTypeDPS})
	})

	// Forever's ranged hit on four classic-db items: Trueaim Gauntlets links
	// to 1301127 "Increased Hit Chance - Ranged - 03" (aura 54, 3), and
	// Infernal Trickster Leggings, Marksman Bands and Stoneshatter to 1294769
	// "- 01" (aura 54, 1), each restricted to ranged weapons
	// (SpellEquippedItems subclass mask 262156), so the hit is the ranged
	// weapon's. The site's simdb rows carry none of it.
	// The restriction holds: a wearer with no bow, gun, crossbow or thrown
	// weapon (a paladin with a libram, whose Judgements and Hammer of
	// Wrath are ranged attacks) gets nothing from them.
	for item, hit := range map[int32]float64{13255: 3, 17754: 1, 18296: 1, 18388: 1} {
		core.NewClientItemEffect(item, func(agent core.Agent) {
			character := agent.GetCharacter()
			if !wieldsRangedWeapon(character) {
				return
			}
			character.AddBonusRangedHitRating(hit * core.HitRatingPerHitChance)
		})
	}

	// ItemEffect 98263 and 98669: 9174, energize 300 rage tenths, on a 1 hour
	// cooldown and the 15 sec category the Hearts of the Mountain share
	// (SpellCategory 1141): "Use: Increase Rage by 30."
	for _, hauberk := range []int32{FireHardenedHauberk, BrutalHauberk} {
		core.NewItemEffect(hauberk, func(agent core.Agent) {
			character := agent.GetCharacter()
			if !character.HasRageBar() {
				return
			}
			actionID := core.ActionID{SpellID: 9174}
			metrics := character.NewRageMetrics(actionID)
			spell := character.RegisterSpell(core.SpellConfig{
				ActionID: actionID,
				Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagOffensiveEquipment,
				Cast: core.CastConfig{
					CD: core.Cooldown{Timer: character.NewTimer(), Duration: time.Hour},
					SharedCD: core.Cooldown{
						Timer:    character.GetOffensiveTrinketCD(),
						Duration: 15 * time.Second,
					},
				},
				ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
					character.AddRage(sim, 30, metrics)
				},
			})
			character.AddMajorCooldown(core.MajorCooldown{
				Spell: spell,
				Type:  core.CooldownTypeDPS,
				ShouldActivate: func(_ *core.Simulation, character *core.Character) bool {
					return character.CurrentRage() <= core.MaxRage-30
				},
			})
		})
	}

	// Spire of Hakkar: 1299948 is Kindling Stave's row shape (aura 42 on a
	// harmful spell, ProcChance 100, no SpellProcsPerMinute row, 10 ms
	// recovery) triggering 1299943, a leech of 28 every 2 sec for 8 sec
	// (duration 31), school mask 1: "steal 112 life from the target enemy
	// over 8 sec."
	core.NewItemEffect(SpireOfHakkar, func(agent core.Agent) {
		character := agent.GetCharacter()
		siphon := character.RegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{SpellID: 1299943},
			SpellSchool: core.SpellSchoolPhysical,
			DefenseType: core.DefenseTypeMagic,
			ProcMask:    core.ProcMaskEmpty,
			Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,

			DamageMultiplier: 1,
			ThreatMultiplier: 1,

			Dot: core.DotConfig{
				Aura:          core.Aura{Label: "Blood Siphon"},
				TickLength:    2 * time.Second,
				NumberOfTicks: 4,
				OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
					dot.Snapshot(target, 28, isRollover)
				},
				OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
					dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
				},
			},
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.Dot(target).Apply(sim)
			},
		})
		core.MakeHarmfulSpellProcAura(&character.Unit, core.HarmfulSpellProc{
			Name:     "Blood Siphon",
			ActionID: core.ActionID{SpellID: 1299948},
			PPM:      core.UnstatedProcsPerMinute,
			ICD:      10 * time.Millisecond,
			Handler:  func(sim *core.Simulation, target *core.Unit) { siphon.Cast(sim, target) },
		})
	})

	// Wall of the Dead: 19409 procs 18828 on 3% of the hits taken
	// (SpellAuraOptions ProcChance 3), aura 674 armour 150 for 20 sec
	// (duration 18): "a 3% chance to encase the caster in bone, increasing
	// armor by 150 for 20 sec."
	core.NewItemEffect(WallOfTheDead, func(agent core.Agent) {
		character := agent.GetCharacter()
		boneArmor := character.NewTemporaryStatsAura("Bone Armor", core.ActionID{SpellID: 18828}, stats.Stats{stats.BonusArmor: 150}, 20*time.Second)
		core.MakeProcTriggerAura(&character.Unit, core.ProcTrigger{
			Name:       "Wall of the Dead",
			Callback:   core.CallbackOnSpellHitTaken,
			ProcMask:   core.ProcMaskMelee,
			Outcome:    core.OutcomeLanded,
			ProcChance: 0.03,
			Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
				boneArmor.Activate(sim)
			},
		})
	})

	// 1293539 is aura 43 (damage on a proc), 14 Holy with variance 0.2, on
	// a block, with a 5 sec cooldown (SpellCooldowns RecoveryTime 5000):
	// "5 sec cooldown. Deals 13 to 15 Holy damage every time you block."
	registerStruckDamage(AegisOfTheScarletCommander, 1293539, core.SpellSchoolHoly, 14, 0.20000000298, core.OutcomeBlock, 5*time.Second)

	// Damage shields, aura 15: 21142 "Deals 7 Fire damage to anyone who
	// strikes you with a melee attack", 1291698 "inflicts 1 Nature damage to
	// the attacker".
	registerStruckDamage(CloakOfFlames, 21142, core.SpellSchoolFire, 7, 0, core.OutcomeLanded, 0)
	registerStruckDamage(SporidCape, 1291698, core.SpellSchoolNature, 1, 0, core.OutcomeLanded, 0)
	// Forever's Razor Gauntlets: 1302193, aura 15, "When struck in combat,
	// inflicts 7 Nature damage to the attacker" (Era's was 3 Arcane).
	registerStruckDamage(RazorGauntlets, 1302193, core.SpellSchoolNature, 7, 0, core.OutcomeLanded, 0)

	registerMindtap()

	// Dormant Heart of the Mountain: equip 1249118 is aura 42 on a helpful
	// spell (ProcTypeMask 0x4000), ProcChance 100, no recovery, triggering
	// 1249119 for 15 sec (duration 1): aura 135 +40 healing and aura 13 +13
	// damage, school mask 126. The server limits it to the four heals its
	// text names; the rows carry no class mask. "Your casts of Greater
	// Heal, Healing Touch, Healing Wave, or Holy Light in combat grant up
	// to 40 increased healing and up to 13 increased damage for 15 sec."
	core.NewClientItemEffect(DormantHeartOfTheMountain, func(agent core.Agent) {
		character := agent.GetCharacter()
		eternalPower := character.NewTemporaryStatsAura("Eternal Power", core.ActionID{SpellID: 1249119},
			stats.Stats{stats.HealingPower: 40, stats.SpellDamage: 13}, 15*time.Second)
		core.MakeProcTriggerAura(&character.Unit, core.ProcTrigger{
			Name:     "Dormant Heart of the Mountain",
			Callback: core.CallbackOnCastComplete,
			Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
				if eternalPowerHeals[spell.SpellID] {
					eternalPower.Activate(sim)
				}
			},
		})
	})
}

// registerStruckDamage deals damage to a melee attacker whose swing at the
// wearer lands with one of outcomes, at most once per cooldown (the proc
// spell's SpellCooldowns RecoveryTime; 0 for none).
func registerStruckDamage(itemID int32, spellID int32, school core.SpellSchool, centre, variance float64, outcomes core.HitOutcome, cooldown time.Duration) {
	core.NewItemEffect(itemID, func(agent core.Agent) {
		character := agent.GetCharacter()
		var cd core.Cooldown
		if cooldown > 0 {
			cd = core.Cooldown{Timer: character.NewTimer(), Duration: cooldown}
		}
		spell := character.RegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{SpellID: spellID},
			SpellSchool: school,
			DefenseType: core.DefenseTypeMagic,
			ProcMask:    core.ProcMaskEmpty,
			Flags:       core.SpellFlagBinary | core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,
			Cast:        core.CastConfig{CD: cd},

			DamageMultiplier: 1,
			ThreatMultiplier: 1,

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.CalcAndDealDamage(sim, target, clientRoll(sim, centre, variance), spell.OutcomeAlwaysHit)
			},
		})
		core.MakeProcTriggerAura(&character.Unit, core.ProcTrigger{
			Name:     spell.ActionID.String() + " (struck)",
			Callback: core.CallbackOnSpellHitTaken,
			ProcMask: core.ProcMaskMelee,
			Outcome:  outcomes,
			Handler: func(sim *core.Simulation, attack *core.Spell, _ *core.SpellResult) {
				if spell.CD.Timer == nil || spell.CD.IsReady(sim) {
					spell.Cast(sim, attack.Unit)
				}
			},
		})
	})
}

// Mindtap Talisman's Forever use, ItemEffect 235401: 1302205 Mindtap on a 7
// min cooldown, aura 24 restoring 125 mana every 1 sec for 10 sec, and
// 1302206 Tapped Mind, aura 29 -60 Intellect for 2 min. "Tap your mind,
// restoring 1250 mana over 10 sec, but reducing Intellect by 60 for 2 min."
// Its 10 sec category 2554 is shared with other trinkets' uses, none of
// which a published set pairs with it, so it is not modelled. It is used
// once the mana it restores and the mana the lost Intellect takes from the
// pool both fit.
func registerMindtap() {
	const (
		manaPerTick     = 125.0
		ticks           = 10
		intellectLost   = 60.0
		tappedMindLasts = 2 * time.Minute
		// A player's mana per point of Intellect (core.EnableManaBar).
		manaPerIntellect = 15.0
	)
	core.NewItemEffect(MindtapTalisman, func(agent core.Agent) {
		character := agent.GetCharacter()
		actionID := core.ActionID{SpellID: 1302205}
		manaMetrics := character.NewManaMetrics(actionID)
		tappedMind := character.NewTemporaryStatsAura("Tapped Mind", core.ActionID{SpellID: 1302206},
			stats.Stats{stats.Intellect: -intellectLost}, tappedMindLasts)
		spell := character.RegisterSpell(core.SpellConfig{
			ActionID: actionID,
			ProcMask: core.ProcMaskEmpty,
			Flags:    core.SpellFlagNoOnCastComplete,
			Cast: core.CastConfig{
				CD: core.Cooldown{Timer: character.NewTimer(), Duration: 7 * time.Minute},
			},
			ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
				tappedMind.Activate(sim)
				core.StartPeriodicAction(sim, core.PeriodicActionOptions{
					Period:   time.Second,
					NumTicks: ticks,
					Priority: core.ActionPriorityAuto,
					OnAction: func(sim *core.Simulation) {
						character.AddMana(sim, manaPerTick, manaMetrics)
					},
				})
			},
		})
		character.AddMajorCooldown(core.MajorCooldown{
			Spell: spell,
			Type:  core.CooldownTypeMana,
			ShouldActivate: func(_ *core.Simulation, character *core.Character) bool {
				return character.MaxMana()-character.CurrentMana() >= manaPerTick*ticks+intellectLost*manaPerIntellect
			},
		})
	})
}

// wieldsRangedWeapon reports whether the ranged slot holds a weapon the
// client's ranged-weapon restriction (subclass mask 262156: bows, guns,
// crossbows and thrown) accepts; a relic, a wand or an empty slot does not.
func wieldsRangedWeapon(character *core.Character) bool {
	switch character.Ranged().RangedWeaponType {
	case proto.RangedWeaponType_RangedWeaponTypeBow, proto.RangedWeaponType_RangedWeaponTypeGun,
		proto.RangedWeaponType_RangedWeaponTypeCrossbow, proto.RangedWeaponType_RangedWeaponTypeThrown:
		return true
	}
	return false
}
