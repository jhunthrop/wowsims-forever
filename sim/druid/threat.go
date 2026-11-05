package druid

// HighThreatMultiplier stands for the client's "generating a high amount of
// threat" (Primal Bite's talent text; Lacerate is read as the same tier).
// The client table has no threat column and nothing states a number for
// "high".
//
// unconfirmed: 2.0 is Swipe's +100% from Season of Discovery's druid tuning
// passive (spell 436895), the only threat modifier the client's druid rows
// state, applied to the tier; Bear Form's own +115% (bearFormThreatMultiplier)
// stacks on top as a unit multiplier.
const HighThreatMultiplier = 2.0

// ClassicBearThreatMultiplier is the server threat Classic gives Maul and
// Swipe: 1.75 times their damage. No client row states it (the client has
// no threat column); the sources are Warcraft Tavern's Classic threat table
// ("Maul x1.75", "Swipe x1.75") and wowsims' Era-based Season of Discovery
// engine (Maul 1.75, Swipe 1.75 doubled by that season's tuning passive
// 436895, which no 1.60.1.70707 SkillLineAbility row teaches, so a Forever
// druid never has it). cmangos classic gives Maul's extra as a flat amount
// instead (spell_threat 9881: 207). Bear Form's +115% stacks on top as a unit
// multiplier.
const ClassicBearThreatMultiplier = 1.75
