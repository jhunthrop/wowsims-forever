package core

import "testing"

// eraAttackTableFixture pins today's Era-derived attack-table numbers so a
// Forever divergence -- the per-item weapon skill cut sevenfold, or the
// second dodge-reduction lever -- fails this test rather than drifting
// silently through the sim unnoticed. Filled by running
// TestAttackTableConstantsAreUnchanged once and copying what it prints.
var eraAttackTableFixture = map[string]derivedTable{
	"skill 300 vs level 63": {
		BaseMissChance:   0.08,
		BaseDodgeChance:  0.065,
		BaseParryChance:  0.14,
		BaseGlanceChance: 0.4,

		GlanceMultiplierMin: 0.55,
		GlanceMultiplierMax: 0.75,

		HitSuppression:       0.01,
		MeleeCritSuppression: 0.048,
		SpellCritSuppression: 0.021,
	},
	"skill 305 vs level 63": {
		BaseMissChance:   0.060000000000000005,
		BaseDodgeChance:  0.060000000000000005,
		BaseParryChance:  0.14,
		BaseGlanceChance: 0.4,

		GlanceMultiplierMin: 0.8,
		GlanceMultiplierMax: 0.8999999999999999,

		HitSuppression:       0,
		MeleeCritSuppression: 0.048,
		SpellCritSuppression: 0.021,
	},
	"skill 300 vs level 60": {
		BaseMissChance:   0.05,
		BaseDodgeChance:  0.05,
		BaseParryChance:  0.05,
		BaseGlanceChance: 0.1,

		GlanceMultiplierMin: 0.91,
		GlanceMultiplierMax: 0.99,

		HitSuppression:       0,
		MeleeCritSuppression: 0,
		SpellCritSuppression: 0,
	},
}

// The attack table is the subsystem most at risk in the Forever port:
// weapon skill survives, but the per-item magnitude fell sevenfold and a
// second dodge-reduction lever exists. Extracting the constants into
// config must change no arithmetic, and a Forever divergence must fail
// here rather than drift silently, so this pins today's derived numbers
// across the level gaps that matter.
func TestAttackTableConstantsAreUnchanged(t *testing.T) {
	// A player at 60 against a boss at 63 is the case every melee spec
	// cares about; 60 against 60 is the control.
	cases := []struct {
		name        string
		weaponSkill float64
		targetLevel int32
	}{
		{"skill 300 vs level 63", 300, 63},
		{"skill 305 vs level 63", 305, 63},
		{"skill 300 vs level 60", 300, 60},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveAttackTable(tc.weaponSkill, tc.targetLevel)
			want, ok := eraAttackTableFixture[tc.name]
			if !ok {
				t.Fatalf("no fixture for %q; add it with the value this prints: %+v", tc.name, got)
			}
			if got != want {
				t.Errorf("the derived attack table moved.\n got %+v\nwant %+v\n"+
					"If this is a deliberate Forever fit, update the fixture in the same commit "+
					"and say in the body which measurement produced it.", got, want)
			}
		})
	}
}
