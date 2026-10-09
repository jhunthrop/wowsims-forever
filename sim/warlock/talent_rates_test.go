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
	near("Shadow Mastery 5/5", shadowMasteryDamagePerRank*5, 0.05)
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
