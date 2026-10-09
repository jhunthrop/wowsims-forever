package priest

import (
	"math"
	"testing"
)

// Forever's 1.60.1.70291 talent text (talents/priest.json) at max rank:
// Silent Resolve 30% Holy threat, Shadow Affinity 30% Shadow threat, Shadow
// Focus 5% Shadow hit, Searing Light 2/5% Holy damage, Shadowform +10%
// Shadow damage, -50% Shadow mana cost, +100% Shadow crit damage bonus and
// 15% less Physical damage taken.
func TestPriestTalentRatesAreForevers(t *testing.T) {
	near := func(name string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	near("Silent Resolve 3/3", silentResolveThreatPerRank*3, 0.30)
	near("Shadow Affinity 3/3", shadowAffinityThreatPerRank*3, 0.30)
	near("Shadow Focus 5/5", shadowFocusHitPerRank*5, 5)
	near("Searing Light 1/2", searingLightDamage[1], 0.02)
	near("Searing Light 2/2", searingLightDamage[2], 0.05)
	near("Shadowform damage", shadowformDamageBonus, 0.10)
	near("Shadowform mana cost", shadowformManaCostPct, -50)
	near("Shadowform crit damage bonus", shadowformCritDamageBonus, 1.0)
	near("Shadowform physical damage taken", shadowformPhysicalDamageTaken, 0.85)
}
