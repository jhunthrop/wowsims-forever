package hunter

// Missile speeds in yards a second, the client's SpellMisc Speed (build
// 1.60.1.70753). A shot lands DistanceFromTarget / speed after it fires.
// Auto Shot's (75, 40) is core's ranged auto attack. The traps carry 0:
// they go off where they lie, with no flight.
const (
	aimedShotMissileSpeed    = 40 // 19434, 20900-20904
	arcaneShotMissileSpeed   = 40 // 3044, 14281-14287
	serpentStingMissileSpeed = 40 // 1978, 13549-13555, 25295
	multiShotMissileSpeed    = 30 // 2643
	sniperShotMissileSpeed   = 60 // 1310687, 1310785, 1310786
	summonHawkMissileSpeed   = 35 // 1293241, 1293525-1293527
	trapMissileSpeed         = 0  // Freezing 1499, Immolation 13795, Explosive 13813 families
)
