package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
)

const clientWarlockSpellconst = "../core/testdata/conformance/client/warlock.json"

// Each table is checked against the client rows of the ids the engine
// registers (or, for Rain of Fire and Death Coil, whose cast ids carry no
// damage effect, the ids of the damage spells they trigger).

func TestWarlockDirectDamageTablesMatchClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientWarlockSpellconst)
	for _, table := range []struct {
		name string
		ids  []int32
		got  []clientdamage.Effect
	}{
		{"Shadow Bolt", []int32{0, 686, 695, 705, 1088, 1106, 7641, 11659, 11660, 11661, 25307}, ShadowBoltDamage[:]},
		{"Shadowburn", []int32{0, 17877, 18867, 18868, 18869, 18870, 18871}, ShadowburnDamage[:]},
		{"Conflagrate", []int32{0, 1293817, 1293818, 17962, 18930, 18931, 18932}, ConflagrateDamage[:]},
		{"Incinerate", []int32{0, 412758, 1293812, 1293813}, IncinerateDamage[:]},
		{"Searing Pain", []int32{0, 5676, 17919, 17920, 17921, 17922, 17923}, SearingPainDamage[:]},
		{"Soul Fire", []int32{0, 6353, 17924}, SoulFireDamage[:]},
		{"Immolate", []int32{0, 348, 707, 1094, 2941, 11665, 11667, 11668, 25309}, ImmolateDamage[:]},
		{"Rain of Fire", []int32{0, 1282380, 1282383, 1282384, 1282385}, RainOfFireTickDamage[:]},
	} {
		clientdamagetest.AssertTable(t, class, clientdamagetest.Direct, table.name, table.ids, table.got, nil)
	}
}

func TestWarlockPeriodicDamageTablesMatchClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientWarlockSpellconst)
	for _, table := range []struct {
		name string
		ids  []int32
		got  []clientdamage.Effect
	}{
		{"Immolate tick", []int32{0, 348, 707, 1094, 2941, 11665, 11667, 11668, 25309}, ImmolateTickDamage[:]},
		{"Corruption tick", []int32{0, 172, 6222, 6223, 7648, 11671, 11672, 25311}, CorruptionTickDamage[:]},
		{"Drain Soul tick", []int32{0, 1120, 8288, 8289, 11675}, DrainSoulTickDamage[:]},
		{"Curse of Agony tick", []int32{0, 980, 1014, 6217, 11711, 11712, 11713}, CurseOfAgonyTickDamage[:]},
		{"Unstable Affliction tick", []int32{0, 427717, 1242971, 1242972}, UnstableAfflictionTickDamage[:]},
	} {
		clientdamagetest.AssertTable(t, class, clientdamagetest.Periodic, table.name, table.ids, table.got, nil)
	}
}

func TestWarlockDrainAndLeechTablesMatchClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientWarlockSpellconst)
	clientdamagetest.AssertTable(t, class, clientdamagetest.PeriodicLeech, "Drain Life tick",
		[]int32{0, 689, 699, 709, 7651, 11699, 11700}, DrainLifeTickDamage[:], nil)
	clientdamagetest.AssertTable(t, class, clientdamagetest.PeriodicLeech, "Siphon Life tick",
		[]int32{0, 18265, 18879, 18880, 18881}, SiphonLifeTickDamage[:], nil)
	clientdamagetest.AssertTable(t, class, clientdamagetest.HealthLeech, "Death Coil",
		[]int32{0, 6789, 17925, 17926}, DeathCoilDamage[:], nil)
}
