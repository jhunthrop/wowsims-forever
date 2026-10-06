package enhancement

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/shaman"
)

// Node indices into the Enhancement tree's own 18-character segment
// and the Restoration tree's 16-character segment, from a direct
// protoreflect dump of proto.ShamanTalents's field order - the same
// ground truth sim/shaman/elemental/talents_rewrite_test.go's own node
// constants use, and for the same reason: talents/shaman.json's build
// 1.60.1.70009 tier/column layout does not always match
// talents_auto_gen.go's source build 1.60.1.69893 field order.
const (
	enhNodeMentalDexterity     = 4
	enhNodeShamanisticFocus    = 8
	enhNodeStormstrike         = 12
	enhNodeSpiritWeapons       = 13
	enhNodeMentalQuickness     = 14
	enhNodeImprovedStormstrike = 15
	enhNodeMaelstromWeapon     = 16
	enhNodeRageOfTheFarseer    = 17

	restoNodeWaterShield = 8
)

// shamanTalentString builds a full three-tree talents string from one
// rank map per tree, so a test can isolate the Enhancement or
// Restoration talent it is guarding while still turning on whatever
// else that talent needs to do anything at all (Improved Stormstrike
// needs Stormstrike itself talented, the same way
// elemental_test.go's own talents need Lava Burst).
func shamanTalentString(elemental, enhancement, restoration map[int]int) string {
	build := func(size int, ranks map[int]int) string {
		nodes := make([]byte, size)
		for i := range nodes {
			nodes[i] = '0'
		}
		for idx, rank := range ranks {
			nodes[idx] = byte('0' + rank)
		}
		return string(nodes)
	}
	return build(16, elemental) + "-" + build(18, enhancement) + "-" + build(16, restoration)
}

func newIsolatedTalentShaman(t *testing.T, talentsString string) (*core.Simulation, *shaman.Shaman) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassShaman,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			Buffs:         core.FullBuffs.Player,
			TalentsString: talentsString,
		},
		&proto.Player_EnhancementShaman{
			EnhancementShaman: &proto.EnhancementShaman{
				Options: &proto.EnhancementShaman_Options{SyncType: proto.ShamanSyncType_Auto},
			},
		},
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	agent, ok := sim.Raid.Parties[0].Players[0].(shaman.ShamanAgent)
	if !ok {
		t.Fatal("the raid's first player is not a shaman agent")
	}
	return sim, agent.GetShaman()
}

// TestMentalDexterityAttackPower guards shaman.applyMentalDexterity:
// node 104755, three ranks, Attack Power += 33/67/100% of Intellect.
func TestMentalDexterityAttackPower(t *testing.T) {
	_, baseline := newIsolatedTalentShaman(t, shamanTalentString(nil, nil, nil))
	baseAP := baseline.GetStat(stats.AttackPower)
	intellect := baseline.GetStat(stats.Intellect)
	if intellect <= 0 {
		t.Fatal("baseline shaman has 0 Intellect; can't tell Mental Dexterity's AP bonus apart from 0")
	}

	cases := []struct {
		rank int
		pct  float64
	}{{1, 0.33}, {2, 0.67}, {3, 1.0}}
	for _, c := range cases {
		_, built := newIsolatedTalentShaman(t, shamanTalentString(nil, map[int]int{enhNodeMentalDexterity: c.rank}, nil))
		if got, want := built.Talents.MentalDexterity, int32(c.rank); got != want {
			t.Fatalf("Mental Dexterity rank %d: got %d, want %d", c.rank, got, want)
		}
		gotAP := built.GetStat(stats.AttackPower)
		wantAP := baseAP + intellect*c.pct
		if diff := gotAP - wantAP; diff > 0.01 || diff < -0.01 {
			t.Errorf("Mental Dexterity %d/3: AttackPower = %v, want %v (baseline %v + %v%% of %v Intellect)", c.rank, gotAP, wantAP, baseAP, c.pct*100, intellect)
		}
	}
}

// TestMentalQuicknessSpellDamage guards shaman.applyMentalQuickness:
// node 104744, two ranks, spell damage and healing += 15/30% of
// Intellect onto both stats.SpellDamage and stats.HealingPower.
func TestMentalQuicknessSpellDamage(t *testing.T) {
	_, baseline := newIsolatedTalentShaman(t, shamanTalentString(nil, nil, nil))
	baseSpellDamage := baseline.GetStat(stats.SpellDamage)
	intellect := baseline.GetStat(stats.Intellect)
	if intellect <= 0 {
		t.Fatal("baseline shaman has 0 Intellect; can't tell Mental Quickness's bonus apart from 0")
	}

	cases := []struct {
		rank int
		pct  float64
	}{{1, 0.15}, {2, 0.30}}
	for _, c := range cases {
		_, built := newIsolatedTalentShaman(t, shamanTalentString(nil, map[int]int{enhNodeMentalQuickness: c.rank}, nil))
		if got, want := built.Talents.MentalQuickness, int32(c.rank); got != want {
			t.Fatalf("Mental Quickness rank %d: got %d, want %d", c.rank, got, want)
		}
		// Both stats.SpellDamage and stats.HealingPower get the
		// dependency (see applyMentalQuickness's own comment), and
		// HealingPower already cascades a further third of itself into
		// SpellDamage (sim/core/character.go's HealingToSpellDamageRatio,
		// applied to every character), so the net SpellDamage gain is
		// 4/3 of the raw percentage, not 1x.
		gotSpellDamage := built.GetStat(stats.SpellDamage)
		wantSpellDamage := baseSpellDamage + intellect*c.pct*(1+core.HealingToSpellDamageRatio)
		if diff := gotSpellDamage - wantSpellDamage; diff > 0.01 || diff < -0.01 {
			t.Errorf("Mental Quickness %d/2: SpellDamage = %v, want %v", c.rank, gotSpellDamage, wantSpellDamage)
		}
	}
}

// TestShamanisticFocusManaCost guards shaman.applyShamanisticFocus:
// node 104749, one rank, -45% mana cost on Shock and Lightning Shield
// spells.
func TestShamanisticFocusManaCost(t *testing.T) {
	_, without := newIsolatedTalentShaman(t, shamanTalentString(nil, nil, nil))
	earthShockWithout := without.EarthShock[len(without.EarthShock)-1]
	baseMultiplier := earthShockWithout.Cost.Multiplier

	_, with := newIsolatedTalentShaman(t, shamanTalentString(nil, map[int]int{enhNodeShamanisticFocus: 1}, nil))
	if !with.Talents.ShamanisticFocus {
		t.Fatal("Shamanistic Focus talent string did not set the talent true")
	}
	earthShockWith := with.EarthShock[len(with.EarthShock)-1]
	flameShockWith := with.FlameShock[len(with.FlameShock)-1]
	frostShockWith := with.FrostShock[len(with.FrostShock)-1]
	lightningShieldWith := with.LightningShield[len(with.LightningShield)-1]

	if got, want := earthShockWith.Cost.Multiplier, baseMultiplier-45; got != want {
		t.Errorf("Shamanistic Focus: Earth Shock Cost.Multiplier = %d, want %d", got, want)
	}
	if got, want := flameShockWith.Cost.Multiplier, baseMultiplier-45; got != want {
		t.Errorf("Shamanistic Focus: Flame Shock Cost.Multiplier = %d, want %d", got, want)
	}
	if got, want := frostShockWith.Cost.Multiplier, baseMultiplier-45; got != want {
		t.Errorf("Shamanistic Focus: Frost Shock Cost.Multiplier = %d, want %d", got, want)
	}
	if got, want := lightningShieldWith.Cost.Multiplier, baseMultiplier-45; got != want {
		t.Errorf("Shamanistic Focus: Lightning Shield Cost.Multiplier = %d, want %d", got, want)
	}
}

// TestSpiritWeaponsGrantsParry guards shaman.applySpiritWeapons: node
// 104745, one rank, grants the parry chance sim/core/character.go's
// baseline 5% stats.Parry only matters once PseudoStats.CanParry is
// true.
func TestSpiritWeaponsGrantsParry(t *testing.T) {
	_, without := newIsolatedTalentShaman(t, shamanTalentString(nil, nil, nil))
	if without.PseudoStats.CanParry {
		t.Error("without Spirit Weapons: PseudoStats.CanParry = true, want false")
	}

	_, with := newIsolatedTalentShaman(t, shamanTalentString(nil, map[int]int{enhNodeSpiritWeapons: 1}, nil))
	if !with.Talents.SpiritWeapons {
		t.Fatal("Spirit Weapons talent string did not set the talent true")
	}
	if !with.PseudoStats.CanParry {
		t.Error("with Spirit Weapons: PseudoStats.CanParry = false, want true")
	}
}

// TestImprovedStormstrikeResetsOnParry guards
// shaman.applyImprovedStormstrike's cooldown-reset half: node 104742,
// at 2/2 (100% chance), the shaman's own Parry outcome against an
// incoming attack resets Stormstrike's cooldown.
func TestImprovedStormstrikeResetsOnParry(t *testing.T) {
	sim, built := newIsolatedTalentShaman(t, shamanTalentString(nil, map[int]int{
		enhNodeStormstrike:         1,
		enhNodeSpiritWeapons:       1, // grants the parry chance this test triggers.
		enhNodeImprovedStormstrike: 2,
	}, nil))
	if got, want := built.Talents.ImprovedStormstrike, int32(2); got != want {
		t.Fatalf("Improved Stormstrike: got rank %d, want %d", got, want)
	}
	if built.Stormstrike == nil {
		t.Fatal("Stormstrike talent string did not register shaman.Stormstrike")
	}

	built.Stormstrike.CD.Timer.Set(sim.CurrentTime + time.Second*20)
	if built.Stormstrike.CD.IsReady(sim) {
		t.Fatal("test setup: Stormstrike CD should start not ready")
	}

	trigger := built.GetAura("Improved Stormstrike Trigger")
	if trigger == nil {
		t.Fatal("Improved Stormstrike did not register its \"Improved Stormstrike Trigger\" aura")
	}

	// A landed attack against the shaman that resolves as a Parry.
	dummySpell := &core.Spell{}
	parryResult := &core.SpellResult{Outcome: core.OutcomeParry}
	trigger.OnSpellHitTaken(trigger, sim, dummySpell, parryResult)

	if !built.Stormstrike.CD.IsReady(sim) {
		t.Error("Improved Stormstrike 2/2 (100% chance): a Parry outcome did not reset Stormstrike's cooldown")
	}
}

// TestMaelstromWeaponReducesLightningBolt guards
// shaman.applyMaelstromWeapon: node 104741, melee damage has a chance
// to add a stack (up to 5), each stack cutting Lightning Bolt's cast
// time and mana cost by 4% * rank, consumed on the next Lightning Bolt
// cast.
func TestMaelstromWeaponReducesLightningBolt(t *testing.T) {
	const rank = 5 // 20%/stack, so 5 stacks is a 100% reduction - easy to check exactly.
	sim, built := newIsolatedTalentShaman(t, shamanTalentString(nil, map[int]int{enhNodeMaelstromWeapon: rank}, nil))
	if got, want := built.Talents.MaelstromWeapon, int32(rank); got != want {
		t.Fatalf("Maelstrom Weapon: got rank %d, want %d", got, want)
	}
	if built.MaelstromWeaponAura == nil {
		t.Fatal("Maelstrom Weapon talent did not register shaman.MaelstromWeaponAura")
	}

	lb := built.LightningBolt[len(built.LightningBolt)-1]
	baseCastMultiplier := lb.CastTimeMultiplier
	baseCostMultiplier := lb.Cost.Multiplier

	built.MaelstromWeaponAura.Activate(sim)
	built.MaelstromWeaponAura.SetStacks(sim, 5)

	const epsilon = 1e-9
	if got, want := lb.CastTimeMultiplier, baseCastMultiplier-1.0; got < want-epsilon || got > want+epsilon {
		t.Errorf("Maelstrom Weapon 5/5 at 5 stacks: Lightning Bolt CastTimeMultiplier = %v, want %v (a 100%% reduction)", got, want)
	}
	if got, want := lb.Cost.Multiplier, baseCostMultiplier-100; got != want {
		t.Errorf("Maelstrom Weapon 5/5 at 5 stacks: Lightning Bolt Cost.Multiplier = %d, want %d (a 100%% reduction)", got, want)
	}

	// Consuming it on the next Lightning Bolt cast restores both.
	// aura.RemainingDuration(sim) == aura.Duration is the "just
	// activated, don't consume" guard OnCastComplete itself checks
	// (talents.go), so time has to move before this is a later event.
	sim.CurrentTime += time.Millisecond
	built.MaelstromWeaponAura.OnCastComplete(built.MaelstromWeaponAura, sim, lb)
	if got, want := lb.CastTimeMultiplier, baseCastMultiplier; got < want-epsilon || got > want+epsilon {
		t.Errorf("after consuming Maelstrom Weapon: Lightning Bolt CastTimeMultiplier = %v, want %v (back to baseline)", got, want)
	}
	if got, want := lb.Cost.Multiplier, baseCostMultiplier; got != want {
		t.Errorf("after consuming Maelstrom Weapon: Lightning Bolt Cost.Multiplier = %d, want %d (back to baseline)", got, want)
	}
}

// TestRageOfTheFarseerSpeedBuff guards shaman.registerRageOfTheFarseer:
// node 104740, one rank, a 3-minute-cooldown self-cast granting 30%
// melee and spell casting speed for 25 sec.
func TestRageOfTheFarseerSpeedBuff(t *testing.T) {
	sim, built := newIsolatedTalentShaman(t, shamanTalentString(nil, map[int]int{enhNodeRageOfTheFarseer: 1}, nil))
	if !built.Talents.RageOfTheFarseer {
		t.Fatal("Rage of the Farseer talent string did not set the talent true")
	}
	if built.RageOfTheFarseer == nil {
		t.Fatal("Rage of the Farseer talent did not register shaman.RageOfTheFarseer")
	}

	baseCastSpeed := built.CastSpeed
	baseMeleeSpeed := built.SwingSpeed()

	built.RageOfTheFarseer.Cast(sim, &built.Unit)

	// unit.CastSpeed (sim/core/unit.go) is a scale on cast *time*, not a
	// "bigger is faster" rate - 1/CastSpeedMultiplier - so a 30% speed
	// increase divides it rather than multiplying.
	if got, want := built.CastSpeed, baseCastSpeed/1.30; got < want-0.0001 || got > want+0.0001 {
		t.Errorf("Rage of the Farseer: CastSpeed = %v, want %v (cast time scaled by 1/1.3, i.e. +30%% speed)", got, want)
	}
	if got, want := built.SwingSpeed(), baseMeleeSpeed*1.30; got < want-0.0001 || got > want+0.0001 {
		t.Errorf("Rage of the Farseer: SwingSpeed = %v, want %v (+30%%)", got, want)
	}
}

// TestWaterShieldRestoresManaOnHitTaken guards
// registerWaterShieldSpell: node 104732 (Restoration tree), one rank,
// 3 charges each restoring 2% of max mana when an attack lands on the
// caster.
func TestWaterShieldRestoresManaOnHitTaken(t *testing.T) {
	sim, built := newIsolatedTalentShaman(t, shamanTalentString(nil, nil, map[int]int{restoNodeWaterShield: 1}))
	if !built.Talents.WaterShield {
		t.Fatal("Water Shield talent string did not set the talent true")
	}
	if built.WaterShield == nil {
		t.Fatal("Water Shield talent did not register shaman.WaterShield")
	}

	built.WaterShield.Cast(sim, &built.Unit)
	if built.ActiveShieldAura == nil || !built.ActiveShieldAura.IsActive() {
		t.Fatal("casting Water Shield did not activate shaman.ActiveShieldAura")
	}

	// Spend mana first: a fresh sim starts at full mana, and AddMana
	// cannot push it over MaxMana, which would hide the restore.
	built.SpendMana(sim, built.MaxMana()*0.5, built.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionManaGain}))
	startMana := built.CurrentMana()
	landedHit := &core.SpellResult{Outcome: core.OutcomeHit, Damage: 100}
	meleeSpell := &core.Spell{ProcMask: core.ProcMaskMeleeMHAuto}
	built.ActiveShieldAura.OnSpellHitTaken(built.ActiveShieldAura, sim, meleeSpell, landedHit)

	wantMana := startMana + built.MaxMana()*0.02
	if got := built.CurrentMana(); got < wantMana-0.01 || got > wantMana+0.01 {
		t.Errorf("Water Shield: mana after one landed hit = %v, want %v (+2%% of max)", got, wantMana)
	}
	if got, want := built.ActiveShieldAura.GetStacks(), int32(2); got != want {
		t.Errorf("Water Shield: charges remaining = %d, want %d (3 - 1 consumed)", got, want)
	}
}
