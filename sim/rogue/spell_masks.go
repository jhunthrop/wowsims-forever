package rogue

import "github.com/wowsims/classic/sim/core"

// The engine's rogue ability masks (core.Spell.ClassSpellMask): every
// ability that costs Energy has one, so a client modifier on a rogue spell
// family reaches it through rogueClassMasks.
const (
	RogueSpellMaskEviscerate uint64 = 1 << iota
	RogueSpellMaskSinisterStrike
	RogueSpellMaskBackstab
	RogueSpellMaskGouge
	RogueSpellMaskKick
	RogueSpellMaskGarrote
	RogueSpellMaskAmbush
	RogueSpellMaskCheapShot
	RogueSpellMaskSliceAndDice
	RogueSpellMaskExposeArmor
	RogueSpellMaskRupture
	RogueSpellMaskHemorrhage
	RogueSpellMaskGhostlyStrike
	RogueSpellMaskFeint
	RogueSpellMaskRiposte
	RogueSpellMaskBladeFlurry
	RogueSpellMaskMutilate
	RogueSpellMaskShank
	RogueSpellMaskDeathMark
)

// rogueClassMasks names the client spell family (SpellClassOptions,
// SpellClassSet 8, build 1.60.1.70755) of each engine rogue ability. Sap,
// Distract, Blind, Kidney Shot and Envenom have families too but no engine
// spell, and so does Venom's buff (1310703, mask_2 67108864), which the
// engine models as an aura Shank applies, not a cast.
var rogueClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{2}, Engine: RogueSpellMaskSinisterStrike},
	{Client: core.ClientClassMask{4}, Engine: RogueSpellMaskBackstab},
	{Client: core.ClientClassMask{8}, Engine: RogueSpellMaskGouge},
	{Client: core.ClientClassMask{16}, Engine: RogueSpellMaskKick},
	{Client: core.ClientClassMask{256}, Engine: RogueSpellMaskGarrote},
	{Client: core.ClientClassMask{512}, Engine: RogueSpellMaskAmbush},
	{Client: core.ClientClassMask{1024}, Engine: RogueSpellMaskCheapShot},
	{Client: core.ClientClassMask{131072}, Engine: RogueSpellMaskEviscerate},
	{Client: core.ClientClassMask{262144}, Engine: RogueSpellMaskSliceAndDice},
	{Client: core.ClientClassMask{524288}, Engine: RogueSpellMaskExposeArmor},
	{Client: core.ClientClassMask{1048576}, Engine: RogueSpellMaskRupture},
	{Client: core.ClientClassMask{33554432}, Engine: RogueSpellMaskHemorrhage},
	{Client: core.ClientClassMask{67108864}, Engine: RogueSpellMaskGhostlyStrike},
	{Client: core.ClientClassMask{134217728}, Engine: RogueSpellMaskFeint},
	{Client: core.ClientClassMask{0, 256}, Engine: RogueSpellMaskRiposte},
	{Client: core.ClientClassMask{0, 2048}, Engine: RogueSpellMaskBladeFlurry},
	{Client: core.ClientClassMask{0, 2097152}, Engine: RogueSpellMaskMutilate},
	{Client: core.ClientClassMask{536870912}, Engine: RogueSpellMaskShank},
	{Client: core.ClientClassMask{0, 0, 4}, Engine: RogueSpellMaskDeathMark},
}
