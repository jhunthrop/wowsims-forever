package shaman

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Elemental Fury (16089) names "your Searing and Magma Totems and your
// Fire, Frost, and Nature spells" through a class mask that carries the
// shock, bolt and nova bits and the Searing Totem's Attack and Magma
// Totem's pulse, and not the healing spells (Healing Wave 0x40, Chain
// Heal 0x100).
func TestElementalFuryAppliesToTheDamageItNames(t *testing.T) {
	magic := func(code int32, flags core.SpellFlag, mask core.ProcMask) *core.Spell {
		return &core.Spell{SpellCode: code, Flags: flags, ProcMask: mask, DefenseType: core.DefenseTypeMagic}
	}
	cases := []struct {
		name  string
		spell *core.Spell
		want  bool
	}{
		{"Lightning Bolt", magic(SpellCode_ShamanLightningBolt, SpellFlagShaman, core.ProcMaskSpellDamage), true},
		{"Earth Shock", magic(SpellCode_ShamanEarthShock, SpellFlagShaman, core.ProcMaskSpellDamage), true},
		{"Searing Totem attack", magic(SpellCode_ShamanSearingTotemAttack, 0, core.ProcMaskEmpty), true},
		{"Magma Totem pulse", magic(SpellCode_ShamanMagmaTotem, 0, core.ProcMaskEmpty), true},
		{"Healing Wave", magic(SpellCode_ShamanHealingWave, SpellFlagShaman|core.SpellFlagHelpful, core.ProcMaskSpellHealing), false},
		{"Chain Heal", magic(SpellCode_ShamanChainHeal, SpellFlagShaman|core.SpellFlagHelpful, core.ProcMaskSpellHealing), false},
		{"a physical swing", &core.Spell{Flags: SpellFlagShaman, DefenseType: core.DefenseTypeMelee}, false},
	}
	for _, tc := range cases {
		if got := elementalFuryApplies(tc.spell); got != tc.want {
			t.Errorf("%s: applies = %v, want %v", tc.name, got, tc.want)
		}
	}
}
