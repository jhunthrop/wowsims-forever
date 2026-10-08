package druid

// HighThreatMultiplier stands for the client's "generating a high amount of
// threat" (Primal Bite's talent text; Lacerate is read as the same tier).
// The client table has no threat column and nothing states a number for
// "high".
//
// unconfirmed: 2.0 is Swipe's +100% from Season of Discovery's druid tuning
// passive (spell 436895), the only threat modifier the client's druid rows
// state, applied to the tier; Bear Form's own +30% (bearFormThreatMultiplier)
// stacks on top as a unit multiplier.
const HighThreatMultiplier = 2.0
