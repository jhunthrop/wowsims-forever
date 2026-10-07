package core

// Global configuration settings

// IncludeAQ selects the Ahn'Qiraj book spell ranks at level 60.
//
// It is false for Forever: Ahn'Qiraj is not on the roadmap (the 9 December 2026
// raids are Barrow Deeps, Hyjal Summit and Onyxia's Lair) and community trainer
// lists at level 60 top out below the book ranks, so a launch character cannot
// learn them. If a book source appears in a future Forever raid, replace this
// constant with a phase switch rather than flipping it.
const IncludeAQ = false
