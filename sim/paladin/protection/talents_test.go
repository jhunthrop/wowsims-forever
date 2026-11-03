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
	spellSealOfFury      = 20423
	spellSealOfFuryProc  = 20418
	spellJudgement       = 20271
	spellHolyStrike      = 10333
	spellHammerRighteous = 407632
	spellHolyShield      = 20928
	// spellHolyShieldRank4 is the level 60 rank client 1.60.1.70753 adds; the
	// reference rotation casts it.
	spellHolyShieldRank4  = 1239570
	spellHolyShieldProc   = 20957
	spellTemplarsBulwark  = 1311015
	spellDivineProtection = 5573
	spellDivineShield     = 1020
	spellSwiftJudgement   = 1310994
	spellJudgementOfFury  = 20414
	spellConsecration     = 20924

	auraRedoubt          = 20128
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

// Sacred Duty (client 1.60.1.70707): "Reduces the cooldown of your Divine
// Shield, Divine Protection, and Templar's Bulwark spells by 30 sec, and
// reduces the duration of Forbearance by 5 sec", 60 and 10 sec at rank 2.
// Its Stamina is gone.
func TestSacredDutyShortensTheDefensiveCooldownsAndForbearance(t *testing.T) {
	base := shortRun{Talents: map[string]int{"templars_bulwark": 1}}.start(t)
	talented := shortRun{Talents: map[string]int{"templars_bulwark": 1, "sacred_duty": 2}}.start(t)

	if got, want := talented.tank.GetStat(stats.Stamina), base.tank.GetStat(stats.Stamina); !near(got, want, tolerance) {
		t.Errorf("Sacred Duty 2 changes Stamina from %v to %v, want no change", want, got)
	}
	forbearance := func(f *fight) time.Duration {
		return f.tank.GetAuraByID(core.ActionID{SpellID: auraForbearance}).Duration
	}
	if got, want := forbearance(talented), forbearance(base)-10*time.Second; got != want {
		t.Errorf("Forbearance with Sacred Duty 2 lasts %v, want %v", got, want)
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
// shield by 15%", 30% at rank 2 (client 1.60.1.70707). The absorb is the shield's block value,
// which the multiplier scales (Strength's share of block value is not
// "damage absorbed by your shield").
func TestShieldSpecializationRaisesTheShieldsAbsorb(t *testing.T) {
	base := shortRun{}.start(t).tank
	talented := shortRun{Talents: map[string]int{"shield_specialization": 2}}.start(t).tank

	if got, want := talented.PseudoStats.BlockValueMultiplier-base.PseudoStats.BlockValueMultiplier, 0.3; !near(got, want, tolerance) {
		t.Errorf("Shield Specialization 2 raises the block value multiplier by %v, want %v", got, want)
	}
	if talented.BlockValue() <= base.BlockValue() {
		t.Errorf("block value %v with the talent, %v without", talented.BlockValue(), base.BlockValue())
	}
}

// Shield Specialization: "gives your blocks a 100% chance to restore 4% of
// your maximum Mana. May only occur once every 3 sec." (rank 2).
func TestShieldSpecializationBlocksRestoreMana(t *testing.T) {
	f := shortRun{Talents: map[string]int{"shield_specialization": 2}}.start(t)
	spend := f.tank.MaxMana() * 0.5
	f.tank.SpendMana(f.sim, spend, f.tank.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionPotion}))

	restores := 4.0 / 100 * f.tank.MaxMana() // 1.60.1.70425: "restore 4% of your maximum Mana" (6% before)
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
	f := shortRun{Talents: map[string]int{"shield_specialization": 2}}.start(t)
	f.tank.SpendMana(f.sim, f.tank.MaxMana()*0.5, f.tank.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionPotion}))
	before := f.tank.CurrentMana()
	f.hit(core.OutcomeHit, 100)
	if got := f.tank.CurrentMana() - before; got > 1 {
		t.Errorf("an unblocked hit restored %v mana", got)
	}
}

// Redoubt: "Damaging melee attacks against you have a 10% chance to
// increase your chance to block by 6%. Lasts 10 sec or 5 blocks.", 30% at
// rank 5 (client 1.60.1.70707): any melee hit may give it, and five blocks
// end it.
func TestRedoubtBlockBonusEndsAfterFiveBlocks(t *testing.T) {
	f := shortRun{Talents: map[string]int{"redoubt": 5}}.start(t)
	aura := f.tank.GetAuraByID(core.ActionID{SpellID: auraRedoubt})
	base := f.tank.GetStat(stats.Block)

	aura.Activate(f.sim)
	aura.SetStacks(f.sim, 5)
	if got := f.tank.GetStat(stats.Block) - base; !near(got, 30, tolerance) {
		t.Fatalf("Redoubt 5 gives +%v block, want +30", got)
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
// taken is reduced by 3%", 9% at rank 3 (client 1.60.1.70707).
func TestImprovedRighteousFuryReducesAllDamageTaken(t *testing.T) {
	talents := map[string]int{"improved_righteous_fury": 3}

	on := shortRun{Talents: talents}.start(t).tank
	if got := on.PseudoStats.DamageTakenMultiplier; !near(got, 0.91, tolerance) {
		t.Errorf("Righteous Fury with Improved Righteous Fury 3 takes %v of the damage, want 0.91", got)
	}

	off := shortRun{Talents: talents, Options: &proto.PaladinOptions{}}.start(t).tank
	if got := off.PseudoStats.DamageTakenMultiplier; !near(got, 1, tolerance) {
		t.Errorf("without Righteous Fury the damage taken multiplier is %v, want 1", got)
	}
}

// Righteous Fury: +33% on all threat and +90% more on Holy damage. The
// shipped 70707 row has +90% on Holy (effect 0, aura 10, school 2); the
// hotfix push of 31 October adds effect 4 (1367626: aura 10, +33%, every
// school), and Wowhead Forever's live text reads "Increases all threat
// generated by 33%, and all threat generated by your Holy attacks by an
// additional 90%". The two threat effects multiply, as every aura 10 does.
// Holy Shield's own "20% additional threat" multiplies on top, and Iron
// Creed's 10% a rank on Holy Strike.
func TestRighteousFuryRaisesAllThreatAndHolyMore(t *testing.T) {
	talents := map[string]int{"holy_shield": 1, "iron_creed": 3, "swift_judgement": 1}
	f := shortRun{Talents: talents}.start(t)

	for _, tc := range []struct {
		name    string
		spellID int32
		want    float64
	}{
		{"Seal of Fury proc", spellSealOfFuryProc, 1.9},
		{"Judgement of Fury", spellJudgementOfFury, 1.9},
		{"Hammer of the Righteous", spellHammerRighteous, 1.9},
		{"Holy Strike with Iron Creed 3", spellHolyStrike, 1.9 * 1.3},
		{"Holy Shield damage", spellHolyShieldProc, 1.9 * 1.2},
	} {
		if got := f.spellOf(t, tc.spellID).ThreatMultiplier; !near(got, tc.want, tolerance) {
			t.Errorf("%s threat multiplier = %v, want %v", tc.name, got, tc.want)
		}
	}
	if got := f.tank.AutoAttacks.MHAuto().ThreatMultiplier; !near(got, 1, tolerance) {
		t.Errorf("the white swing's own threat multiplier = %v, want 1: the 33%% is on the unit", got)
	}
	if got := f.tank.PseudoStats.ThreatMultiplier; !near(got, 1.33, tolerance) {
		t.Errorf("the paladin's threat multiplier = %v, want 1.33 (all threat +33%%)", got)
	}

	off := shortRun{Talents: talents, Options: &proto.PaladinOptions{}}.start(t)
	if got := off.spellOf(t, spellHammerRighteous).ThreatMultiplier; !near(got, 1, tolerance) {
		t.Errorf("Hammer of the Righteous without Righteous Fury has threat multiplier %v, want 1", got)
	}
	if got := off.tank.PseudoStats.ThreatMultiplier; !near(got, 1, tolerance) {
		t.Errorf("without Righteous Fury the paladin's threat multiplier is %v, want 1", got)
	}
}

func TestIronCreedReducesDamageTakenAfterHolyStrike(t *testing.T) {
	f := shortRun{Talents: map[string]int{"iron_creed": 3}}.start(t)
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
	if got := f.tank.PseudoStats.DamageTakenMultiplier / before; !near(got, 0.94, tolerance) {
		t.Errorf("Iron Creed 3 takes %v of the damage, want 0.94", got)
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

	withoutFury := shortRun{Talents: map[string]int{"iron_creed": 3}, Options: &proto.PaladinOptions{}}.start(t)
	if withoutFury.tank.GetAuraByID(core.ActionID{SpellID: auraIronCreed}) != nil {
		t.Error("Iron Creed's damage reduction exists without Righteous Fury")
	}
}

// Holy Shield: "Increases chance to block by 30% for 10 sec, and deals 75
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
// equipped".
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

// Swift Judgement (client 1.60.1.70707): "Unleash the energy of a Seal
// spell upon an enemy. Does not consume the Seal. ... Does not share a
// cooldown with your Judgement ability." It is a second Judgement of the
// Seal that is up: it leaves Judgement's cooldown alone and costs nothing.
func TestSwiftJudgementJudgesTheSealOnItsOwnCooldown(t *testing.T) {
	rotation := rotationOf(
		[]string{prepullCast(spellSealOfFury, "2")},
		[]string{cast(spellJudgement), cast(spellSwiftJudgement)})
	metrics := shortRun{Talents: map[string]int{"swift_judgement": 1}, Rotation: rotation, Seconds: 5}.metrics(t)

	if got := actionCasts(metrics, spellSwiftJudgement); got != 1 {
		t.Fatalf("Swift Judgement was cast %v times, want once", got)
	}
	if got := actionCasts(metrics, spellJudgement); got != 1 {
		t.Errorf("Judgement was cast %v times in 5 seconds, want 1: Swift Judgement does not reset it", got)
	}
	if got := actionCasts(metrics, spellJudgementOfFury); got != 2 {
		t.Errorf("Judgement of Fury landed %v times, want 2 (Judgement and Swift Judgement)", got)
	}
	if got := resourceEvents(metrics, spellSwiftJudgement); got != 0 {
		t.Errorf("Swift Judgement spent mana %v times, want none", got)
	}
}

// Shieldsworn (client 1.60.1.70707): Holy Strike's cooldown drops by 4
// seconds, and with a shield on it fires Holy Shield's block damage while
// Holy Shield is up, without spending a charge.
func TestShieldswornShortensHolyStrikeAndStrikesWithTheShield(t *testing.T) {
	talents := map[string]int{"holy_shield": 1, "shieldsworn": 1}
	base := shortRun{Talents: map[string]int{"holy_shield": 1}}.start(t)
	if got, want := base.spellOf(t, spellHolyStrike).CD.Duration, 10*time.Second; got != want {
		t.Fatalf("Holy Strike's cooldown without Shieldsworn is %v, want %v", got, want)
	}

	f := shortRun{Talents: talents}.start(t)
	holyStrike := f.spellOf(t, spellHolyStrike)
	if got, want := holyStrike.CD.Duration, 6*time.Second; got != want {
		t.Errorf("Holy Strike's cooldown with Shieldsworn is %v, want %v", got, want)
	}
	shield := f.spellOf(t, spellHolyShield)
	damage := f.spellOf(t, spellHolyShieldProc)
	aura := f.tank.GetAuraByID(core.ActionID{SpellID: auraHolyShield})

	holyStrike.ApplyEffects(f.sim, f.boss, holyStrike)
	if got := damage.SpellMetrics[f.boss.UnitIndex].Casts; got != 0 {
		t.Errorf("Holy Strike fired Holy Shield %d times with Holy Shield down", got)
	}
	shield.ApplyEffects(f.sim, f.boss, shield)
	holyStrike.ApplyEffects(f.sim, f.boss, holyStrike)
	if got := damage.SpellMetrics[f.boss.UnitIndex].Casts; got != 1 {
		t.Errorf("Holy Strike fired Holy Shield %d times with Holy Shield up, want 1", got)
	}
	if got := aura.GetStacks(); got != 4 {
		t.Errorf("Holy Shield has %d charges after the shield strike, want 4", got)
	}

	bare := shortRun{Talents: talents, Equipment: withoutShield(t)}.start(t)
	bareStrike := bare.spellOf(t, spellHolyStrike)
	bareStrike.ApplyEffects(bare.sim, bare.boss, bareStrike)
	if bare.tank.PseudoStats.CanBlock {
		t.Fatal("the shieldless tank can block")
	}
}

// Shieldsworn's shield strike rolls Shield Specialization's mana (100% at
// rank 2) without its 3 second internal cooldown.
func TestShieldswornStrikeRestoresShieldSpecializationMana(t *testing.T) {
	f := shortRun{Talents: map[string]int{"redoubt": 5, "shield_specialization": 2, "shieldsworn": 1}}.start(t)
	f.tank.SpendMana(f.sim, f.tank.MaxMana()*0.5, f.tank.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionPotion}))
	holyStrike := f.spellOf(t, spellHolyStrike)
	restores := 0.04 * f.tank.MaxMana()
	for strike := 1; strike <= 2; strike++ {
		before := f.tank.CurrentMana()
		holyStrike.ApplyEffects(f.sim, f.boss, holyStrike)
		if got := f.tank.CurrentMana() - before; !near(got, restores, 1) {
			t.Errorf("shield strike %d restored %v mana, want %v", strike, got, restores)
		}
	}
}

// Hammer of the Righteous strikes up to three targets.
// Hammer of the Righteous strikes four targets: 407632's damage effect 0
// has EffectChainTargets 4 (effect 2 too), and Wowhead Forever's text is
// "the current target and up to 3 additional nearby targets".
func TestHammerOfTheRighteousStrikesFourTargets(t *testing.T) {
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
	if struck != 4 {
		t.Errorf("Hammer of the Righteous damaged %d targets of 5, want 4", struck)
	}
}

// Holy Shield's mana costs moved down a bracket with its damage: the 70707
// hotfix push of 31 October sets the top rank's SpellPower 184913 to 195
// (shipped 240), and Wowhead Forever's live 20928 reads 195 Mana.
func TestHolyShieldCostsTheHotfixedMana(t *testing.T) {
	f := shortRun{Talents: map[string]int{"holy_shield": 1}}.start(t)
	if got := f.spellOf(t, spellHolyShield).Cost.BaseCost; got != 195 {
		t.Errorf("the top Holy Shield's base cost is %v, want 195", got)
	}
}

// otherActionThreat sums the threat an "other" action (mana gain, rage
// gain) put on every target.
func otherActionThreat(metrics *proto.UnitMetrics, other proto.OtherAction) float64 {
	var threat float64
	for _, action := range metrics.Actions {
		if action.Id.GetOtherId() != other {
			continue
		}
		for _, target := range action.Targets {
			threat += target.Threat
		}
	}
	return threat
}

// Shield Specialization's mana comes from 1310925 (SpellEffect 1340826,
// effect 137 energize 4%), whose SpellMisc 857517 carries Attributes_1
// 0x400, SPELL_ATTR1_NO_THREAT (1.60.1.70707; the hotfix push leaves the
// row alone). Both server reimplementations drop an energize's threat for
// that attribute before any amount is computed: TrinityCore's
// ThreatManager::ForwardThreatForAssistingMe and cmangos classic's
// HostileRefManager::threatAssist. So the blocks restore mana and add no
// threat.
func TestShieldSpecializationManaAddsNoThreat(t *testing.T) {
	metrics := shortRun{
		Talents:    map[string]int{"shield_specialization": 2},
		Rotation:   rotationOf(nil, []string{cast(spellConsecration)}), // spends the mana the blocks restore
		Boss:       rapidBoss(1.0),
		Seconds:    60,
		Iterations: 5,
	}.metrics(t)

	if got := resourceActualGain(metrics, 20150); got <= 0 {
		t.Fatalf("Shield Specialization restored %v mana over the run, want some", got)
	}
	if got := otherActionThreat(metrics, proto.OtherAction_OtherActionManaGain); got != 0 {
		t.Errorf("Shield Specialization's mana made %v threat, want 0 (SPELL_ATTR1_NO_THREAT)", got)
	}
}

func resourceActualGain(metrics *proto.UnitMetrics, spellID int32) float64 {
	for _, resource := range metrics.Resources {
		if resource.Type == proto.ResourceType_ResourceTypeMana && resource.Id.GetSpellId() == spellID {
			return resource.ActualGain
		}
	}
	return 0
}

// A mana gain's threat is half a point a mana, with no threat multiplier:
// TrinityCore's Unit::EnergizeBySpell forwards it with ignoreModifiers,
// and the Classic threat research says so in as many words (Magey's
// classic-warrior wiki, Threat Mechanics: "Power gains are not affected
// ... by threat multipliers"; Warcraft Tavern's Classic threat table:
// "These values are not ever multiplied by modifiers"). The engine's rage
// gain already works that way. So Righteous Fury's +33% does not raise
// it.
func TestManaGainThreatIgnoresRighteousFury(t *testing.T) {
	metrics := shortRun{
		Rotation:   rotationOf([]string{prepullCast(20357, "1")}, []string{cast(spellConsecration)}),
		Seconds:    60,
		Iterations: 5,
	}.metrics(t)

	var gained float64
	for _, resource := range metrics.Resources {
		if resource.Type != proto.ResourceType_ResourceTypeMana || resource.Id.GetOtherId() == proto.OtherAction_OtherActionManaRegen || resource.ActualGain <= 0 {
			continue // spirit and mp5 regen add no threat; spends are negative
		}
		gained += resource.ActualGain
	}
	if gained <= 0 {
		t.Fatal("Seal of Wisdom restored no mana over the run")
	}
	if got, want := otherActionThreat(metrics, proto.OtherAction_OtherActionManaGain), 0.5*gained; !near(got, want, 0.01*want) {
		t.Errorf("mana gains of %.0f made %.1f threat under Righteous Fury, want %.1f (0.5 a mana, unmodified)", gained, got, want)
	}
}

// Judgement of Fury (20414 and its ranks) carries the same rows round 6
// moved Judgement of Righteousness and Command to the melee table with:
// SpellCategories DefenseType 2 (row 76345) and SpellMisc Attributes_0
// 0x250000, which holds 0x200000, no active defense. It rolled the spell
// table instead: about 13% misses and 1.5x crits.
func TestJudgementOfFuryRollsTheMeleeTableWithoutActiveDefenses(t *testing.T) {
	f := shortRun{}.start(t)
	judgement := f.spellOf(t, spellJudgementOfFury)
	if judgement.DefenseType != core.DefenseTypeMelee {
		t.Errorf("Judgement of Fury defense type = %v, want melee", judgement.DefenseType)
	}
	misses := 0
	const casts = 4000
	for i := 0; i < casts; i++ {
		judgement.ApplyEffects(f.sim, f.boss, judgement)
	}
	for _, target := range judgement.SpellMetrics {
		misses += int(target.Misses)
		if target.Dodges+target.Parries+target.Blocks > 0 {
			t.Errorf("Judgement of Fury was dodged, parried or blocked: %+v", target)
		}
	}
	// The boss is three levels up: 9% melee misses less Precision, against
	// 17% on the spell table.
	if rate := float64(misses) / casts; rate > 0.11 {
		t.Errorf("Judgement of Fury misses %.1f%%, want the melee table's (at most 9%%)", 100*rate)
	}
}

// Iron Creed (1311034) opens its damage reduction from aura 231 on a done
// melee-class spell (ProcTypeMask 16) with no hit mask, which both server
// cores read as a landed hit or crit (TrinityCore's default done hit mask;
// cmangos procs on hit or crit only). The engine opened it before checking
// whether Holy Strike landed, so a missed, dodged or parried strike still
// cut the damage taken.
func TestIronCreedOpensOnlyOnALandedHolyStrike(t *testing.T) {
	f := shortRun{Talents: map[string]int{"iron_creed": 3}}.start(t)
	holyStrike := f.spellOf(t, spellHolyStrike)
	creed := f.tank.GetAuraByID(core.ActionID{SpellID: auraIronCreed})
	opened, landed := 0, 0
	for i := 0; i < 2000; i++ {
		creed.Deactivate(f.sim)
		avoided := func() int32 {
			m := holyStrike.SpellMetrics[0]
			return m.Misses + m.Dodges + m.Parries
		}
		before := avoided()
		holyStrike.ApplyEffects(f.sim, f.boss, holyStrike)
		if avoided() == before {
			landed++
		}
		if creed.IsActive() {
			opened++
		}
	}
	if landed == 2000 {
		t.Fatal("every Holy Strike landed; the test needs misses, dodges or parries")
	}
	if opened != landed {
		t.Errorf("Iron Creed opened on %d strikes of which %d landed, want only the landed ones", opened, landed)
	}
}

// Divine Protection and Divine Shield (498, 5573, 642, 1020) are one
// SpellCategories Category, 37, with a 300 s CategoryRecoveryTime: casting
// one puts both on cooldown. Templar's Bulwark (1311015) has Category 0 and
// its own cooldown. The engine gave each its own timer, so only Forbearance
// kept them apart.
func TestDivineProtectionAndDivineShieldShareOneCooldown(t *testing.T) {
	f := shortRun{Talents: map[string]int{"templars_bulwark": 1}}.start(t)
	shield := f.spellOf(t, spellDivineShield)
	protection := f.spellOf(t, spellDivineProtection)
	bulwark := f.spellOf(t, spellTemplarsBulwark)
	if shield.CD.Timer != protection.CD.Timer {
		t.Error("Divine Shield and Divine Protection have separate cooldowns, want one (category 37)")
	}
	if bulwark.CD.Timer == shield.CD.Timer {
		t.Error("Templar's Bulwark shares the immunities' cooldown, want its own")
	}
}
