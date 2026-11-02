package druid

import "testing"

// Nature's Swiftness (17116) names its spells in SpellEffect 695239: class
// masks 0x1000261 / 0x2000000 / 0x2000000 / 0. Of the spells this package
// registers, those bits hold Wrath (mask_0 0x1), Healing Touch (0x20) and
// Regrowth (0x40). Wild Growth is mask_1 0x4000000 (SpellClassOptions 408120
// and 1238215), which Nature's Swiftness does not name, so it can neither
// make Wild Growth instant nor be spent by one.
func TestNaturesSwiftnessCoversOnlyTheSpellsItsClientMaskNames(t *testing.T) {
	want := DruidSpellMaskWrath | DruidSpellMaskHealingTouch | DruidSpellMaskRegrowth
	if DruidSpellMaskNatureCasts != want {
		t.Errorf("Nature's Swiftness covers mask %b, client names %b (Wrath, Healing Touch, Regrowth)", DruidSpellMaskNatureCasts, want)
	}
	if DruidSpellMaskNatureCasts&DruidSpellMaskWildGrowth != 0 {
		t.Error("Nature's Swiftness covers Wild Growth, which its client mask does not name")
	}
}
