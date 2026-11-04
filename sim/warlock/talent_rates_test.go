package warlock

import (
	"math"
	"testing"
	"time"
)

// Forever's 1.60.1.70291 talent text (talents/warlock.json) at max rank:
// Shadow Mastery +5% Shadow damage, Demonic Embrace +15% Stamina, Soul Link
// 30% of damage taken moves to the demon, Ruin +100% Destruction crit damage
// bonus (20% a rank), Improved Corruption +10% damage, Unholy Power +10% pet
// damage, Master Demonologist +10% per effect, and Demonic Sacrifice 15% /
// 2% Mana / 3% Health per 4 sec.
func TestWarlockTalentRatesAreForevers(t *testing.T) {
	near := func(name string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	near("Shadow Mastery 5/5", shadowMasteryDamagePerRank*5, 0.10)
	near("Demonic Embrace 5/5", demonicEmbraceStaminaPerRank*5, 0.15)
	near("Soul Link damage taken", soulLinkDamageTakenMultiplier, 0.70)
	near("Ruin 5/5", ruinCritDamageBonusPerRank*5, 1.0)
	near("Improved Corruption 5/5", improvedCorruptionDamagePerRank*5, 0.10)
	near("Unholy Power 5/5", unholyPowerDamagePerRank*5, 0.10)
	near("Master Demonologist 5/5", masterDemonologistEffectPerRank*5, 0.10)
	near("Demonic Sacrifice school damage", demonicSacrificeSchoolDamage, 1.15)
	near("Demonic Sacrifice mana", demonicSacrificeManaPercent, 0.02)
	near("Demonic Sacrifice health", demonicSacrificeHealthPercent, 0.03)
	if demonicSacrificeRegenPeriod != 4*time.Second {
		t.Errorf("Demonic Sacrifice period = %v, want 4s", demonicSacrificeRegenPeriod)
	}
}

// TestShadowAndFlameReadsThe70755Curve pins 426316's effects 2 and 3
// (Shadow and Fire damage) at 4% a rank, 20% at 5/5, raised from 2% a rank
// by the 70755 hotfix push, and effect 1's 20% a rank Immolate chance.
func TestShadowAndFlameReadsThe70755Curve(t *testing.T) {
	if got := shadowAndFlameDamagePerRank * 5; math.Abs(got-0.20) > 1e-9 {
		t.Errorf("Shadow and Flame 5/5 damage = %v, want 0.20", got)
	}
	if got := shadowAndFlameImmolatePreserveChance * 5; math.Abs(got-1.0) > 1e-9 {
		t.Errorf("Shadow and Flame 5/5 Immolate chance = %v, want 1.0", got)
	}
	if shadowAndFlameBuffDuration != 20*time.Second {
		t.Errorf("Shadow and Flame buff lasts %v, want 20s", shadowAndFlameBuffDuration)
	}
}
