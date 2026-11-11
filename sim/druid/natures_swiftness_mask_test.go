package druid

import "testing"

// Nature's Swiftness (17116) names its spells in SpellEffect 695239. Client
// 1.60.1.70888 (Blizzard's 10 November hotfix, "Nature's Swiftness now also
// makes Wild Growth instant") widens mask_1 from 0x2000000 to 0x6000000, so
// besides Wrath (mask_0 0x1), Healing Touch (0x20) and Regrowth (0x40) it
// now names Wild Growth (mask_1 0x4000000, SpellClassOptions 408120 and
// 1238215): Wild Growth is made instant by it and spends it.
func TestNaturesSwiftnessCoversTheSpellsItsClientMaskNames(t *testing.T) {
	want := DruidSpellMaskWrath | DruidSpellMaskHealingTouch | DruidSpellMaskRegrowth | DruidSpellMaskWildGrowth
	if DruidSpellMaskNatureCasts != want {
		t.Errorf("Nature's Swiftness covers mask %b, client names %b (Wrath, Healing Touch, Regrowth, Wild Growth)", DruidSpellMaskNatureCasts, want)
	}
}
