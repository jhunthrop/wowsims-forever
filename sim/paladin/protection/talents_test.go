package protection

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

const (
	spellSealOfFury       = 20423
	spellSealOfFuryProc   = 20418
	spellJudgement        = 20271
	spellHolyStrike       = 10333
	spellHammerRighteous  = 407632
	spellHolyShield       = 20928
	spellHolyShieldProc   = 20957
	spellTemplarsBulwark  = 1311015
	spellDivineProtection = 5573
	spellDivineShield     = 1020
	spellSwiftJudgement   = 1310994
	spellJudgementOfFury  = 20414
	spellConsecration     = 20924

	auraRedoubt          = 20134
	auraIronCreed        = 1311033
	auraSealOfFuryShield = 1311648
	auraHolyShield       = 20928
	auraForbearance      = 25771

	tolerance = 1e-6
)

func near(a, b, within float64) bool { return math.Abs(a-b) <= within }

func withoutShield(t testing.TB) *proto.EquipmentSpec {
	t.Helper()
	gear := core.GetGearSet(gearDir, harnessGearSet).GearSet
	equipment := &proto.EquipmentSpec{Items: append([]*proto.ItemSpec(nil), gear.Items...)}
	const offHand = 15
	equipment.Items[offHand] = &proto.ItemSpec{}
	return equipment
}

// Anticipation: "Increases your Defense Skill by 4" a rank, 20 at rank 5.
func TestAnticipationAddsFourDefensePerRank(t *testing.T) {
	base := shortRun{}.start(t).tank.GetStat(stats.Defense)
	for rank := 1; rank <= 5; rank++ {
		got := shortRun{Talents: map[string]int{"anticipation": rank}}.start(t).tank.GetStat(stats.Defense) - base
		if !near(got, float64(4*rank), tolerance) {
			t.Errorf("Anticipation %d: +%v defense, want +%d", rank, got, 4*rank)
		}
	}
}

// Toughness: "Increases your armor value from items by 2%" a rank, so a
// helm of 526 armor is 52.6 armor richer at rank 5.
func TestToughnessAddsTwoPercentOfItemArmorPerRank(t *testing.T) {
	const helmArmor = 526 // Enchanted Thorium Helm
	helm := &proto.EquipmentSpec{Items: []*proto.ItemSpec{{Id: 12620}}}

	base := shortRun{Equipment: helm}.start(t).tank.GetStat(stats.Armor)
	got := shortRun{Equipment: helm, Talents: map[string]int{"toughness": 5}}.start(t).tank.GetStat(stats.Armor) - base
	if !near(got, 0.10*helmArmor, 0.01) {
		t.Errorf("Toughness 5: +%v armor, want +%v", got, 0.10*helmArmor)
	}
}

// Precision: "Improves your chance to hit by 1%" a rank.
func TestPrecisionAddsOnePercentHitPerRank(t *testing.T) {
	base := shortRun{}.start(t).tank.GetStat(stats.Hit)
	got := shortRun{Talents: map[string]int{"precision": 3}}.start(t).tank.GetStat(stats.Hit) - base
	if !near(got, 3*core.HitRatingPerHitChance, tolerance) {
		t.Errorf("Precision 3: +%v hit, want +%v", got, 3*core.HitRatingPerHitChance)
	}
}

// Sacred Duty: "Increases your total Stamina by 2% and reduces the
// cooldown of your Divine Shield, Divine Protection, and Templar's
// Bulwark spells by 30 sec", 4% and 60 sec at rank 2.
func TestSacredDutyAddsStaminaAndShortensTheDefensiveCooldowns(t *testing.T) {
	base := shortRun{Talents: map[string]int{"templars_bulwark": 1}}.start(t)
	talented := shortRun{Talents: map[string]int{"templars_bulwark": 1, "sacred_duty": 2}}.start(t)

	ratio := talented.tank.GetStat(stats.Stamina) / base.tank.GetStat(stats.Stamina)
	if !near(ratio, 1.04, 1e-3) {
		t.Errorf("Sacred Duty 2 scales Stamina by %v, want 1.04", ratio)
	}

	for _, spellID := range []int32{spellTemplarsBulwark, spellDivineProtection, spellDivineShield} {
		if got, want := base.spellOf(t, spellID).CD.Duration, 5*time.Minute; got != want {
			t.Errorf("spell %d cooldown = %v, want %v", spellID, got, want)
		}
		if got, want := talented.spellOf(t, spellID).CD.Duration, 4*time.Minute; got != want {
			t.Errorf("spell %d cooldown with Sacred Duty 2 = %v, want %v", spellID, got, want)
		}
	}
}

// Shield Specialization: "Increases the amount of damage absorbed by your
// shield by 10%", 30% at rank 3. The absorb is the shield's block value,
// which the multiplier scales (Strength's share of block value is not
// "damage absorbed by your shield").
func TestShieldSpecializationRaisesTheShieldsAbsorb(t *testing.T) {
	base := shortRun{}.start(t).tank
	talented := shortRun{Talents: map[string]int{"shield_specialization": 3}}.start(t).tank

	if got, want := talented.PseudoStats.BlockValueMultiplier-base.PseudoStats.BlockValueMultiplier, 0.3; !near(got, want, tolerance) {
		t.Errorf("Shield Specialization 3 raises the block value multiplier by %v, want %v", got, want)
	}
	if talented.BlockValue() <= base.BlockValue() {
		t.Errorf("block value %v with the talent, %v without", talented.BlockValue(), base.BlockValue())
	}
}

// Shield Specialization: "gives your blocks a 100% chance to restore 6% of
// your maximum Mana. May only occur once every 3 sec." (rank 3).
func TestShieldSpecializationBlocksRestoreMana(t *testing.T) {
	f := shortRun{Talents: map[string]int{"shield_specialization": 3}}.start(t)
	spend := f.tank.MaxMana() * 0.5
	f.tank.SpendMana(f.sim, spend, f.tank.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionPotion}))

	restores := 6.0 / 100 * f.tank.MaxMana()
	before := f.tank.CurrentMana()
	f.hit(core.OutcomeBlock, 100)
	if got := f.tank.CurrentMana() - before; !near(got, restores, 1) {
		t.Errorf("a block restored %v mana, want %v", got, restores)
	}

	before = f.tank.CurrentMana()
	f.hit(core.OutcomeBlock, 100)
	if got := f.tank.CurrentMana() - before; got > 1 {
		t.Errorf("a second block inside the 3 s window restored %v mana, want none", got)
	}

	f.advance(3100 * time.Millisecond)
	before = f.tank.CurrentMana()
	f.hit(core.OutcomeBlock, 100)
	if got := f.tank.CurrentMana() - before; !near(got, restores, 50) {
		t.Errorf("a block after the window restored %v mana, want about %v", got, restores)
	}
}

// Shield Specialization restores nothing on a hit that is not blocked.
func TestShieldSpecializationIgnoresUnblockedHits(t *testing.T) {
	f := shortRun{Talents: map[string]int{"shield_specialization": 3}}.start(t)
	f.tank.SpendMana(f.sim, f.tank.MaxMana()*0.5, f.tank.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionPotion}))
	before := f.tank.CurrentMana()
	f.hit(core.OutcomeHit, 100)
	if got := f.tank.CurrentMana() - before; got > 1 {
		t.Errorf("an unblocked hit restored %v mana", got)
	}
}

// Redoubt: "Damaging melee attacks against you have a 10% chance to
// increase your chance to block by 4%. Lasts 10 sec or 5 blocks.", 20% at
// rank 5: any melee hit may give it, and five blocks end it.
func TestRedoubtBlockBonusEndsAfterFiveBlocks(t *testing.T) {
	f := shortRun{Talents: map[string]int{"redoubt": 5}}.start(t)
	aura := f.tank.GetAuraByID(core.ActionID{SpellID: auraRedoubt})
	base := f.tank.GetStat(stats.Block)

	aura.Activate(f.sim)
	aura.SetStacks(f.sim, 5)
	if got := f.tank.GetStat(stats.Block) - base; !near(got, 20, tolerance) {
		t.Fatalf("Redoubt 5 gives +%v block, want +20", got)
	}

	// A spell hit cannot give Redoubt again, so only the blocks spend it.
	for block := 1; block <= 5; block++ {
		if !aura.IsActive() {
			t.Fatalf("Redoubt ended after %d blocks, want 5", block-1)
		}
		f.hitWith(f.magic, core.OutcomeBlock, 100)
	}
	if aura.IsActive() {
		t.Errorf("Redoubt is still up after five blocks")
	}
	if got := f.tank.GetStat(stats.Block) - base; !near(got, 0, tolerance) {
		t.Errorf("block is still +%v after Redoubt ended", got)
	}
}

func TestRedoubtProcsOnDamagingMeleeHits(t *testing.T) {
	boss := rapidBoss(0.5)
	with := shortRun{Talents: map[string]int{"redoubt": 5}, Boss: boss, Seconds: 60, Iterations: 20}.metrics(t)
	without := shortRun{Boss: boss, Seconds: 60, Iterations: 20}.metrics(t)

	if got := auraProcs(with, auraRedoubt); got < 1 {
		t.Errorf("Redoubt procs %v times a minute against a boss swinging every half second, want several", got)
	}
	if got := auraProcs(without, auraRedoubt); got != 0 {
		t.Errorf("an untalented tank procs Redoubt %v times", got)
	}
}

// Reckoning: "a 20% chance to gain an extra attack after being the victim
// of a non-periodic critical strike", 100% at rank 5.
func TestReckoningGrantsAnExtraAttackWhenCritted(t *testing.T) {
	extraAttacks := func(talents map[string]int, outcome core.HitOutcome) int {
		f := shortRun{Talents: talents}.start(t)
		lines := f.logLines()
		f.hit(outcome, 100)
		return countContaining(lines(), "extra main-hand attack from {SpellID: 20178}")
	}

	if got := extraAttacks(map[string]int{"reckoning": 5}, core.OutcomeCrit); got != 1 {
		t.Errorf("a crit with Reckoning 5 granted %d extra attacks, want 1", got)
	}
	if got := extraAttacks(nil, core.OutcomeCrit); got != 0 {
		t.Errorf("a crit without Reckoning granted %d extra attacks", got)
	}
	if got := extraAttacks(map[string]int{"reckoning": 5}, core.OutcomeHit); got != 0 {
		t.Errorf("a plain hit granted %d extra attacks, want none", got)
	}
}

// Improved Righteous Fury: "While Righteous Fury is active, all damage
// taken is reduced by 2%", 6% at rank 3.
func TestImprovedRighteousFuryReducesAllDamageTaken(t *testing.T) {
	talents := map[string]int{"improved_righteous_fury": 3}

	on := shortRun{Talents: talents}.start(t).tank
	if got := on.PseudoStats.DamageTakenMultiplier; !near(got, 0.94, tolerance) {
		t.Errorf("Righteous Fury with Improved Righteous Fury 3 takes %v of the damage, want 0.94", got)
	}

	off := shortRun{Talents: talents, Options: &proto.PaladinOptions{}}.start(t).tank
	if got := off.PseudoStats.DamageTakenMultiplier; !near(got, 1, tolerance) {
		t.Errorf("without Righteous Fury the damage taken multiplier is %v, want 1", got)
	}
}

// Righteous Fury: +60% threat on Holy damage, and not on anything else.
// Holy Shield's own "20% additional threat" multiplies on top, and Iron
// Creed's 5% a rank on Holy Strike.
func TestRighteousFuryRaisesHolyThreatOnly(t *testing.T) {
	talents := map[string]int{"holy_shield": 1, "iron_creed": 5, "swift_judgement": 1}
	f := shortRun{Talents: talents}.start(t)

	for _, tc := range []struct {
		name    string
		spellID int32
		want    float64
	}{
		{"Seal of Fury proc", spellSealOfFuryProc, 1.6},
		{"Judgement of Fury", spellJudgementOfFury, 1.6},
		{"Hammer of the Righteous", spellHammerRighteous, 1.6},
		{"Holy Strike with Iron Creed 5", spellHolyStrike, 1.6 * 1.25},
		{"Holy Shield damage", spellHolyShieldProc, 1.6 * 1.2},
	} {
		if got := f.spellOf(t, tc.spellID).ThreatMultiplier; !near(got, tc.want, tolerance) {
			t.Errorf("%s threat multiplier = %v, want %v", tc.name, got, tc.want)
		}
	}
	if got := f.tank.AutoAttacks.MHAuto().ThreatMultiplier; !near(got, 1, tolerance) {
		t.Errorf("the white swing's threat multiplier = %v, want 1: Righteous Fury raises Holy only", got)
	}

	off := shortRun{Talents: talents, Options: &proto.PaladinOptions{}}.start(t)
	if got := off.spellOf(t, spellHammerRighteous).ThreatMultiplier; !near(got, 1, tolerance) {
		t.Errorf("Hammer of the Righteous without Righteous Fury has threat multiplier %v, want 1", got)
	}
}

// Instrument of Law: "reduces all threat you generate by 10% while
// Righteous Fury is not active", 20% at rank 2.
func TestInstrumentOfLawLowersThreatWithoutRighteousFury(t *testing.T) {
	talents := map[string]int{"instrument_of_law": 2}

	off := shortRun{Talents: talents, Options: &proto.PaladinOptions{}}.start(t).tank
	if got := off.PseudoStats.ThreatMultiplier; !near(got, 0.8, tolerance) {
		t.Errorf("threat multiplier without Righteous Fury = %v, want 0.8", got)
	}
	on := shortRun{Talents: talents}.start(t).tank
	if got := on.PseudoStats.ThreatMultiplier; !near(got, 1, tolerance) {
		t.Errorf("threat multiplier with Righteous Fury = %v, want 1", got)
	}
}

// Iron Creed: "While Righteous Fury is active, Holy Strike also reduces
// your damage taken by 2% for 6 sec." (10% at rank 5).
func TestIronCreedReducesDamageTakenAfterHolyStrike(t *testing.T) {
	f := shortRun{Talents: map[string]int{"iron_creed": 5}}.start(t)
	holyStrike := f.spellOf(t, spellHolyStrike)
	creed := f.tank.GetAuraByID(core.ActionID{SpellID: auraIronCreed})
	if creed == nil {
		t.Fatal("Iron Creed's damage-taken aura is not registered")
	}

	before := f.tank.PseudoStats.DamageTakenMultiplier
	holyStrike.ApplyEffects(f.sim, f.boss, holyStrike)
	if !creed.IsActive() {
		t.Fatal("Holy Strike did not open Iron Creed")
	}
	if got := f.tank.PseudoStats.DamageTakenMultiplier / before; !near(got, 0.9, tolerance) {
		t.Errorf("Iron Creed 5 takes %v of the damage, want 0.9", got)
	}
	if got, want := creed.Duration, 6*time.Second; got != want {
		t.Errorf("Iron Creed lasts %v, want %v", got, want)
	}

	f.advance(6100 * time.Millisecond)
	if creed.IsActive() {
		t.Error("Iron Creed outlasted its 6 seconds")
	}
	if got := f.tank.PseudoStats.DamageTakenMultiplier; !near(got, before, tolerance) {
		t.Errorf("damage taken stays at %v after Iron Creed, want %v", got, before)
	}

	withoutFury := shortRun{Talents: map[string]int{"iron_creed": 5}, Options: &proto.PaladinOptions{}}.start(t)
	if withoutFury.tank.GetAuraByID(core.ActionID{SpellID: auraIronCreed}) != nil {
		t.Error("Iron Creed's damage reduction exists without Righteous Fury")
	}
}

// One-Handed Weapon Specialization: "Increases the damage you deal with
// one-handed melee weapons by 3%", 7% and 10% at ranks 2 and 3.
func TestOneHandedWeaponSpecializationRaisesWeaponAttacks(t *testing.T) {
	base := shortRun{}.start(t).spellOf(t, spellHammerRighteous).DamageMultiplier
	for rank, bonus := range map[int]float64{1: 0.03, 2: 0.07, 3: 0.10} {
		got := shortRun{Talents: map[string]int{"one_handed_weapon_specialization": rank}}.start(t).spellOf(t, spellHammerRighteous).DamageMultiplier
		if !near(got/base, 1+bonus, tolerance) {
			t.Errorf("rank %d scales Hammer of the Righteous by %v, want %v", rank, got/base, 1+bonus)
		}
	}
}

// Holy Shield: "Increases chance to block by 30% for 10 sec, and deals 110
// Holy damage for each attack blocked while active. ... Each block expends
// a charge. 4 charges."
func TestHolyShieldCostsACharge(t *testing.T) {
	f := shortRun{Talents: map[string]int{"holy_shield": 1}}.start(t)
	shield := f.spellOf(t, spellHolyShield)
	aura := f.tank.GetAuraByID(core.ActionID{SpellID: auraHolyShield})
	damage := f.spellOf(t, spellHolyShieldProc)
	base := f.tank.GetStat(stats.Block)

	shield.ApplyEffects(f.sim, f.boss, shield)
	if got := aura.GetStacks(); got != 4 {
		t.Fatalf("Holy Shield starts with %d charges, want 4", got)
	}
	if got := f.tank.GetStat(stats.Block) - base; !near(got, 30, tolerance) {
		t.Errorf("Holy Shield gives +%v block, want +30", got)
	}

	for block := 1; block <= 4; block++ {
		f.hit(core.OutcomeBlock, 100)
		if got, want := damage.SpellMetrics[f.boss.UnitIndex].Casts, int32(block); got != want {
			t.Errorf("after %d blocks Holy Shield has dealt damage %d times", block, got)
		}
	}
	if aura.IsActive() {
		t.Error("Holy Shield outlived its four charges")
	}
	if got := f.tank.GetStat(stats.Block) - base; !near(got, 0, tolerance) {
		t.Errorf("block is still +%v after Holy Shield ended", got)
	}
}

func TestHolyShieldRecastRestoresTheCharges(t *testing.T) {
	f := shortRun{Talents: map[string]int{"holy_shield": 1}}.start(t)
	shield := f.spellOf(t, spellHolyShield)
	aura := f.tank.GetAuraByID(core.ActionID{SpellID: auraHolyShield})

	shield.ApplyEffects(f.sim, f.boss, shield)
	f.hit(core.OutcomeBlock, 100)
	f.hit(core.OutcomeBlock, 100)
	if got := aura.GetStacks(); got != 2 {
		t.Fatalf("Holy Shield has %d charges after two blocks, want 2", got)
	}
	shield.ApplyEffects(f.sim, f.boss, shield)
	if got := aura.GetStacks(); got != 4 {
		t.Errorf("a recast leaves %d charges, want 4", got)
	}
}

// Templar's Bulwark: "an absorb shield equal to 100% of your maximum
// health for 8 sec. Applies Forbearance for 1 min."
func TestTemplarsBulwarkAbsorbsAWholeHealthBar(t *testing.T) {
	f := shortRun{Talents: map[string]int{"templars_bulwark": 1}}.start(t)
	bulwark := f.spellOf(t, spellTemplarsBulwark)
	aura := f.tank.GetAuraByID(core.ActionID{SpellID: spellTemplarsBulwark})

	bulwark.ApplyEffects(f.sim, f.boss, bulwark)
	if !aura.IsActive() {
		t.Fatal("Templar's Bulwark did not start")
	}
	if got, want := aura.Duration, 8*time.Second; got != want {
		t.Errorf("Templar's Bulwark lasts %v, want %v", got, want)
	}

	healthBefore := f.tank.CurrentHealth()
	if result := f.hit(core.OutcomeHit, 1000); result.Damage != 0 {
		t.Errorf("a hit inside the shield dealt %v", result.Damage)
	}
	if f.tank.CurrentHealth() != healthBefore {
		t.Errorf("health fell from %v to %v inside the shield", healthBefore, f.tank.CurrentHealth())
	}

	result := f.hit(core.OutcomeHit, 100000)
	if result.Damage <= 0 {
		t.Fatal("a hit far larger than the shield dealt nothing")
	}
	if aura.IsActive() {
		t.Error("Templar's Bulwark outlived its pool")
	}
	soaked := bulwark.SpellMetrics[f.tank.UnitIndex].TotalShielding
	if !near(soaked, f.tank.MaxHealth(), 1) {
		t.Errorf("the shield soaked %v, want the maximum health %v", soaked, f.tank.MaxHealth())
	}
}

func TestTemplarsBulwarkEndsAfterEightSeconds(t *testing.T) {
	f := shortRun{Talents: map[string]int{"templars_bulwark": 1}}.start(t)
	bulwark := f.spellOf(t, spellTemplarsBulwark)
	bulwark.ApplyEffects(f.sim, f.boss, bulwark)

	f.advance(8100 * time.Millisecond)
	if result := f.hit(core.OutcomeHit, 1000); result.Damage <= 0 {
		t.Error("the shield still absorbs after its 8 seconds")
	}
}

// Divine Shield, Divine Protection and Templar's Bulwark share Forbearance.
func TestForbearanceBlocksTheOtherDefensives(t *testing.T) {
	f := shortRun{Talents: map[string]int{"templars_bulwark": 1}}.start(t)
	shield := f.spellOf(t, spellDivineShield)
	bulwark := f.spellOf(t, spellTemplarsBulwark)

	if !shield.CanCast(f.sim, f.boss) {
		t.Fatal("Divine Shield cannot be cast before anything else is")
	}
	bulwark.ApplyEffects(f.sim, f.boss, bulwark)
	if !f.tank.GetAuraByID(core.ActionID{SpellID: auraForbearance}).IsActive() {
		t.Fatal("Templar's Bulwark did not apply Forbearance")
	}
	if shield.CanCast(f.sim, f.boss) {
		t.Error("Divine Shield can be cast under Forbearance")
	}
}

// Divine Protection: immune to every school for its duration, and the
// paladin cannot attack.
func TestDivineProtectionMakesTheTankImmune(t *testing.T) {
	f := shortRun{}.start(t)
	protection := f.spellOf(t, spellDivineProtection)
	protection.ApplyEffects(f.sim, f.boss, protection)

	if result := f.hit(core.OutcomeHit, 5000); result.Damage != 0 {
		t.Errorf("a physical hit dealt %v to an immune tank", result.Damage)
	}
	if result := f.hitWith(f.magic, core.OutcomeHit, 5000); result.Damage != 0 {
		t.Errorf("a Holy hit dealt %v to an immune tank", result.Damage)
	}

	f.advance(8100 * time.Millisecond)
	if result := f.hit(core.OutcomeHit, 5000); result.Damage <= 0 {
		t.Error("the tank is still immune after Divine Protection's 8 seconds")
	}
}

// Divine Shield: immune, and the paladin deals 50% less damage meanwhile.
func TestDivineShieldHalvesDamageDealt(t *testing.T) {
	f := shortRun{}.start(t)
	shield := f.spellOf(t, spellDivineShield)
	before := f.tank.PseudoStats.DamageDealtMultiplier

	shield.ApplyEffects(f.sim, f.boss, shield)
	if got := f.tank.PseudoStats.DamageDealtMultiplier / before; !near(got, 0.5, tolerance) {
		t.Errorf("Divine Shield scales damage dealt by %v, want 0.5", got)
	}
	if result := f.hit(core.OutcomeHit, 5000); result.Damage != 0 {
		t.Errorf("a hit dealt %v to a tank inside Divine Shield", result.Damage)
	}

	f.advance(12100 * time.Millisecond)
	if got := f.tank.PseudoStats.DamageDealtMultiplier; !near(got, before, tolerance) {
		t.Errorf("damage dealt stays at %v after Divine Shield, want %v", got, before)
	}
}

// Seal of Fury: "a small absorb worth half the damage with a shield
// equipped", and Improved Seal of Fury restores mana when it is fully
// absorbed, 15% more for each level the attacker stands above the paladin.
func TestSealOfFuryShieldIsHalfTheProcDamage(t *testing.T) {
	f := shortRun{}.start(t)
	proc := f.spellOf(t, spellSealOfFuryProc)
	shield := f.tank.GetAuraByID(core.ActionID{SpellID: auraSealOfFuryShield})

	proc.ApplyEffects(f.sim, f.boss, proc)
	dealt := proc.SpellMetrics[f.boss.UnitIndex].TotalDamage
	if dealt <= 0 {
		t.Fatal("the Seal of Fury proc dealt no damage")
	}
	if !shield.IsActive() {
		t.Fatal("the proc did not raise the shield")
	}

	f.hit(core.OutcomeHit, 3000)
	if got := proc.SpellMetrics[f.tank.UnitIndex].TotalShielding; !near(got, dealt/2, 1e-6) {
		t.Errorf("the shield soaked %v of a proc that dealt %v, want half", got, dealt)
	}
	if shield.IsActive() {
		t.Error("a hit larger than the shield left it standing")
	}
}

func TestSealOfFuryRaisesNoShieldWithoutAShield(t *testing.T) {
	f := shortRun{Equipment: withoutShield(t)}.start(t)
	proc := f.spellOf(t, spellSealOfFuryProc)

	proc.ApplyEffects(f.sim, f.boss, proc)
	if f.tank.GetAuraByID(core.ActionID{SpellID: auraSealOfFuryShield}).IsActive() {
		t.Error("Seal of Fury shielded a tank with no shield equipped")
	}
}

func TestImprovedSealOfFuryRestoresManaWhenTheShieldIsSpent(t *testing.T) {
	const bossLevelsAbove = 3.0 // the harness boss is level 63
	f := shortRun{Talents: map[string]int{"improved_seal_of_fury": 1}}.start(t)
	proc := f.spellOf(t, spellSealOfFuryProc)
	f.tank.SpendMana(f.sim, f.tank.MaxMana()*0.5, f.tank.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionPotion}))

	proc.ApplyEffects(f.sim, f.boss, proc)
	before := f.tank.CurrentMana()
	f.hit(core.OutcomeHit, 3000)

	want := 0.01 * f.tank.MaxMana() * (1 + 0.15*bossLevelsAbove)
	if got := f.tank.CurrentMana() - before; !near(got, want, 1) {
		t.Errorf("the spent shield restored %v mana, want %v", got, want)
	}

	// Nothing to spend, nothing restored.
	before = f.tank.CurrentMana()
	f.hit(core.OutcomeHit, 3000)
	if got := f.tank.CurrentMana() - before; got > 1 {
		t.Errorf("a hit with no shield up restored %v mana", got)
	}
}

func TestWithoutImprovedSealOfFuryTheShieldRestoresNothing(t *testing.T) {
	f := shortRun{}.start(t)
	proc := f.spellOf(t, spellSealOfFuryProc)
	f.tank.SpendMana(f.sim, f.tank.MaxMana()*0.5, f.tank.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionPotion}))

	proc.ApplyEffects(f.sim, f.boss, proc)
	before := f.tank.CurrentMana()
	f.hit(core.OutcomeHit, 3000)
	if got := f.tank.CurrentMana() - before; got > 1 {
		t.Errorf("the spent shield restored %v mana without the talent", got)
	}
}

// Judgement no longer consumes the seal (Blizzard's Deep Dive recap).
func TestJudgementKeepsTheSealUp(t *testing.T) {
	rotation := rotationOf(
		[]string{prepullCast(spellSealOfFury, "2")},
		[]string{cast(spellJudgement)})
	metrics := shortRun{Rotation: rotation, Seconds: 12}.metrics(t)

	if got := actionCasts(metrics, spellJudgement); got < 2 {
		t.Fatalf("Judgement was cast %v times in 12 seconds, want at least 2", got)
	}
	if got := auraUptime(metrics, spellSealOfFury); !near(got, 12, 0.01) {
		t.Errorf("Seal of Fury was up for %v of 12 seconds, want all of it", got)
	}
}

// Swift Judgement: "Finishes the remaining cooldown on your Judgement
// ability and reduces the Mana cost of your next Judgement by 100%."
func TestSwiftJudgementResetsJudgementAndMakesTheNextOneFree(t *testing.T) {
	rotation := rotationOf(
		[]string{prepullCast(spellSealOfFury, "2")},
		[]string{cast(spellJudgement), cast(spellSwiftJudgement)})
	metrics := shortRun{Talents: map[string]int{"swift_judgement": 1}, Rotation: rotation, Seconds: 5}.metrics(t)

	if got := actionCasts(metrics, spellSwiftJudgement); got != 1 {
		t.Fatalf("Swift Judgement was cast %v times, want once", got)
	}
	if got := actionCasts(metrics, spellJudgement); got != 2 {
		// The first Judgement, then the one Swift Judgement made ready; the
		// ten second cooldown holds a third past the five second fight.
		t.Errorf("Judgement was cast %v times, want 2", got)
	}
	if got, want := resourceEvents(metrics, spellJudgement), 1.0; got != want {
		t.Errorf("Judgement spent mana %v times, want %v: the cast Swift Judgement made ready is free", got, want)
	}
}

// Hammer of the Righteous strikes up to three targets.
func TestHammerOfTheRighteousStrikesThreeTargets(t *testing.T) {
	rotation := rotationOf(nil, []string{cast(spellHammerRighteous)})
	metrics := shortRun{Rotation: rotation, Targets: 5, Seconds: 3, Iterations: 10}.metrics(t)

	var struck int
	for _, action := range metrics.Actions {
		if action.Id.GetSpellId() != spellHammerRighteous {
			continue
		}
		for _, target := range action.Targets {
			if target.Damage > 0 {
				struck++
			}
		}
	}
	if struck != 3 {
		t.Errorf("Hammer of the Righteous damaged %d targets of 5, want 3", struck)
	}
}
