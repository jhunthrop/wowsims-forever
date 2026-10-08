package core

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
)

type OnComboPointsSpent func(sim *Simulation, spell *Spell, comboPoints int32)
type OnComboPointsGained func(sim *Simulation)

// Forever regenerates Energy continuously instead of in Classic's 2.02 s ticks
// (Blizzard class-highlights video, 30 Sep 2026). No per-second rate is
// published, so the Classic mean rate is kept: 20.2 Energy per 2.02 s.
//
// The bar accrues on demand: every read and every change first brings the
// stored value up to sim.CurrentTime at the current rate, so the value is
// exact at any instant. A scheduled task only exists to wake the APL when a
// decision threshold is crossed.
//
// The stored value is an integer count of energyUnitsPerEnergy-ths of an
// Energy. Accrual, spending and the wake time are integer arithmetic, so they
// are identical on every CPU architecture: Go fuses a float "x*y + z" into one
// rounding on arm64 but not on amd64, and a threshold comparison such as
// "energy >= cost" can flip on that last bit. Floats appear only when a value
// is derived by a single division or a single rounded product, neither of
// which can be fused.
const EnergyRegenPerSecond = 10

// energyUnitsPerEnergy is the resolution of the stored value: 1e-8 Energy.
// At 10 Energy per second that is exactly one unit per nanosecond, the
// resolution of sim time, so accrual never rounds.
const energyUnitsPerEnergy int64 = 100_000_000

// energyUnitsPerNanosecond is the regen of one multiplier step in units per
// nanosecond. It must divide exactly (see TestEnergyUnitsPerNanosecondIsExact).
const energyUnitsPerNanosecond = EnergyRegenPerSecond * energyUnitsPerEnergy / int64(time.Second)

// energyToUnits converts an Energy amount to units, rounding to the nearest
// unit. The product is its own rounded statement: it is not fused with
// anything.
func energyToUnits(energy float64) int64 {
	scaled := energy * float64(energyUnitsPerEnergy)
	return int64(math.Round(scaled))
}

// energyFromUnits derives the float Energy value by one division, which the
// compiler cannot fuse and which is exact for whole Energy amounts.
func energyFromUnits(units int64) float64 {
	return float64(units) / float64(energyUnitsPerEnergy)
}

// Wake interval for units whose APL has no precomputed decision thresholds
// (everything but rogues), so energy-conditioned decisions are re-evaluated.
const EnergyPollInterval = time.Millisecond * 100

// EnergyForTime is the base Energy regenerated over a duration.
func EnergyForTime(duration time.Duration) float64 {
	return energyFromUnits(int64(duration) * energyUnitsPerNanosecond)
}

// TimeForEnergy is the base time needed to regenerate an amount of Energy.
func TimeForEnergy(amount float64) time.Duration {
	return time.Duration(energyToUnits(amount) / energyUnitsPerNanosecond)
}

type energyBar struct {
	unit *Unit

	maxEnergy    float64
	maxUnits     int64
	currentUnits int64

	comboPoints      int32
	comboPointTarget *Unit

	// Lifecycle Callbacks
	onComboPointsSpentCallbacks  []OnComboPointsSpent  // Triggered when the energy user successfully spends combo points on a spell
	onComboPointsGainedCallbacks []OnComboPointsGained // Triggered when the energy user successfully gains combo points

	// List of energy levels that might affect APL decisions. E.g:
	// [10, 15, 20, 30, 60, 85]
	energyDecisionThresholds []int

	// Slice with len == maxEnergy+1 with each index corresponding to an amount of energy. Looks like this:
	// [0, 0, 0, 0, 1, 1, 1, 2, 2, 2, 2, 2, 3, 3, ...]
	// Increments by 1 at each value of energyDecisionThresholds.
	cumulativeEnergyDecisionThresholds []int

	sim            *Simulation
	active         bool
	lastAccrualAt  time.Duration
	nextWakeAt     time.Duration
	notifiedBucket int // decision-threshold bucket the APL was last told about

	// Multiplies continuous energy regen (Adrenaline Rush). Change it only
	// through AddEnergyRegenMultiplier so earlier time accrues at the old rate.
	regenMultiplier int64

	regenMetrics        *ResourceMetrics
	EnergyRefundMetrics *ResourceMetrics
}

func (unit *Unit) EnableEnergyBar(maxEnergy float64) {
	unit.SetCurrentPowerBar(EnergyBar)

	maxEnergy = max(100, maxEnergy)
	unit.energyBar = energyBar{
		unit:                unit,
		maxEnergy:           maxEnergy,
		maxUnits:            energyToUnits(maxEnergy),
		regenMultiplier:     1,
		regenMetrics:        unit.NewEnergyMetrics(ActionID{OtherID: proto.OtherAction_OtherActionEnergyRegen}),
		EnergyRefundMetrics: unit.NewEnergyMetrics(ActionID{OtherID: proto.OtherAction_OtherActionRefund}),
	}
}

// Computes the energy thresholds.
func (eb *energyBar) setupEnergyThresholds() {
	// eb.unit.Rotation is nil for an energy-bar unit built with no
	// rotation configured at all (e.g. a test character that drives
	// ApplyEffects directly rather than through the APL - see
	// sim/rogue/dps_rogue/mutilate_test.go's buildRogueForTest, which
	// has to pass an empty-but-non-nil &proto.APLRotation{} to work
	// around exactly this). allAPLActions() dereferences the receiver's
	// own fields (priorityList) with no nil check of its own, so a
	// genuinely nil Rotation panicked here before this guard existed.
	if eb.unit == nil || eb.unit.Rotation == nil {
		return
	}
	var energyThresholds []int

	// Energy thresholds from spell costs.
	for _, action := range eb.unit.Rotation.allAPLActions() {
		for _, spell := range action.GetAllSpells() {
			if spell.Cost != nil && spell.Cost.CostType() == CostTypeEnergy {
				energyThresholds = append(energyThresholds, int(math.Ceil(spell.DefaultCast.Cost)))
			}
		}
	}

	// Energy thresholds from conditional comparisons.
	for _, action := range eb.unit.Rotation.allAPLActions() {
		for _, value := range action.GetAllAPLValues() {
			if cmpValue, ok := value.(*APLValueCompare); ok {
				_, lhsIsEnergy := cmpValue.lhs.(*APLValueCurrentEnergy)
				_, rhsIsEnergy := cmpValue.rhs.(*APLValueCurrentEnergy)
				if !lhsIsEnergy && !rhsIsEnergy {
					continue
				}

				lhsConstVal := getConstAPLFloatValue(cmpValue.lhs)
				rhsConstVal := getConstAPLFloatValue(cmpValue.rhs)

				if lhsIsEnergy && rhsConstVal != -1 {
					energyThresholds = append(energyThresholds, int(math.Ceil(rhsConstVal)))
				} else if rhsIsEnergy && lhsConstVal != -1 {
					energyThresholds = append(energyThresholds, int(math.Ceil(lhsConstVal)))
				}
			}
		}
	}

	slices.SortStableFunc(energyThresholds, func(t1, t2 int) int {
		return t1 - t2
	})

	// Add each unique value to the final thresholds list.
	curVal := 0
	for _, threshold := range energyThresholds {
		if threshold > curVal {
			eb.energyDecisionThresholds = append(eb.energyDecisionThresholds, threshold)
			curVal = threshold
		}
	}

	curEnergy := 0
	cumulativeVal := 0
	eb.cumulativeEnergyDecisionThresholds = make([]int, int(eb.maxEnergy)+1)
	for _, threshold := range eb.energyDecisionThresholds {
		for curEnergy < threshold {
			eb.cumulativeEnergyDecisionThresholds[curEnergy] = cumulativeVal
			curEnergy++
		}
		cumulativeVal++
	}
	for curEnergy < len(eb.cumulativeEnergyDecisionThresholds) {
		eb.cumulativeEnergyDecisionThresholds[curEnergy] = cumulativeVal
		curEnergy++
	}
}

func (unit *Unit) HasEnergyBar() bool {
	return unit.energyBar.unit != nil
}

func (eb *energyBar) CurrentEnergy() float64 {
	eb.accrue()
	return energyFromUnits(eb.currentUnits)
}

func (eb *energyBar) MaxEnergy() float64 {
	return eb.maxEnergy
}

// regenUnitsPerNanosecond is the current regen including multipliers.
func (eb *energyBar) regenUnitsPerNanosecond() int64 {
	return energyUnitsPerNanosecond * eb.regenMultiplier
}

// AddEnergyRegenMultiplier changes the regen multiplier by a whole number of
// steps. Time up to now is accrued at the old rate first.
func (eb *energyBar) AddEnergyRegenMultiplier(delta int64) {
	if eb.regenMultiplier+delta < 0 {
		panic("Energy regen multiplier cannot go negative!")
	}
	eb.accrue()
	eb.regenMultiplier += delta
	eb.scheduleWake()
}

// accrue brings the stored energy up to the current sim time. It is silent:
// it never calls into the APL, which is the wake task's job.
func (eb *energyBar) accrue() {
	if !eb.active {
		return
	}
	now := eb.sim.CurrentTime
	if now <= eb.lastAccrualAt {
		return
	}
	accrued := int64(now-eb.lastAccrualAt) * eb.regenUnitsPerNanosecond()
	eb.lastAccrualAt = now

	newUnits := min(eb.currentUnits+accrued, eb.maxUnits)
	eb.regenMetrics.AddEvent(energyFromUnits(accrued), energyFromUnits(newUnits-eb.currentUnits))
	eb.currentUnits = newUnits
}

// thresholdBucket maps an energy value in units to its decision-threshold bucket.
func (eb *energyBar) thresholdBucket(units int64) int {
	if eb.cumulativeEnergyDecisionThresholds == nil {
		return 0
	}
	idx := min(max(int(units/energyUnitsPerEnergy), 0), len(eb.cumulativeEnergyDecisionThresholds)-1)
	return eb.cumulativeEnergyDecisionThresholds[idx]
}

func (eb *energyBar) notifyAPL(sim *Simulation) {
	if sim.CurrentTime < 0 || sim.Options.Interactive || eb.unit.Rotation == nil {
		return
	}
	eb.unit.Rotation.DoNextAction(sim)
}

// computeWakeAt is when the APL next needs to hear about energy: the moment
// the next decision threshold is reached, or a poll step when no thresholds
// were precomputed.
func (eb *energyBar) computeWakeAt() time.Duration {
	if !eb.active || eb.unit.Rotation == nil || eb.currentUnits >= eb.maxUnits {
		return NeverExpires
	}
	if eb.cumulativeEnergyDecisionThresholds == nil {
		return eb.sim.CurrentTime + EnergyPollInterval
	}
	floor := int(eb.currentUnits / energyUnitsPerEnergy)
	for _, threshold := range eb.energyDecisionThresholds {
		if threshold <= floor {
			continue
		}
		if float64(threshold) > eb.maxEnergy {
			break
		}
		missing := int64(threshold)*energyUnitsPerEnergy - eb.currentUnits
		rate := eb.regenUnitsPerNanosecond()
		// Round up so the wake never lands short of the threshold.
		return eb.sim.CurrentTime + time.Duration((missing+rate-1)/rate)
	}
	return NeverExpires
}

func (eb *energyBar) scheduleWake() {
	eb.nextWakeAt = eb.computeWakeAt()
	if eb.nextWakeAt != NeverExpires {
		eb.sim.RescheduleTask(eb.nextWakeAt)
	}
}

func (eb *energyBar) addEnergyInternal(sim *Simulation, amount float64, metrics *ResourceMetrics) bool {
	if amount < 0 {
		panic("Trying to add negative energy!")
	}
	eb.accrue()

	newUnits := min(eb.currentUnits+energyToUnits(amount), eb.maxUnits)
	metrics.AddEvent(amount, energyFromUnits(newUnits-eb.currentUnits))

	if sim.Log != nil {
		eb.unit.Log(sim, "Gained %0.3f energy from %s (%0.3f --> %0.3f).", amount, metrics.ActionID, energyFromUnits(eb.currentUnits), energyFromUnits(newUnits))
	}

	crossedThreshold := eb.cumulativeEnergyDecisionThresholds == nil || eb.thresholdBucket(eb.currentUnits) != eb.thresholdBucket(newUnits)
	eb.currentUnits = newUnits
	eb.notifiedBucket = eb.thresholdBucket(newUnits)

	return crossedThreshold
}

func (eb *energyBar) AddEnergy(sim *Simulation, amount float64, metrics *ResourceMetrics) {
	if eb.addEnergyInternal(sim, amount, metrics) {
		eb.notifyAPL(sim)
	}
	eb.scheduleWake()
}

func (eb *energyBar) SpendEnergy(sim *Simulation, amount float64, metrics *ResourceMetrics) {
	if amount < 0 {
		panic("Trying to spend negative energy!")
	}
	eb.accrue()

	newUnits := eb.currentUnits - energyToUnits(amount)
	metrics.AddEvent(-amount, -amount)

	if sim.Log != nil {
		eb.unit.Log(sim, "Spent %0.3f energy from %s (%0.3f --> %0.3f).", amount, metrics.ActionID, energyFromUnits(eb.currentUnits), energyFromUnits(newUnits))
	}

	eb.currentUnits = newUnits
	eb.notifiedBucket = eb.thresholdBucket(newUnits)
	eb.scheduleWake()
}

func (eb *energyBar) ComboPoints() int32 {
	return eb.comboPoints
}

func (eb *energyBar) AddComboPointsIgnoreTarget(sim *Simulation, pointsToAdd int32, metrics *ResourceMetrics) {
	newComboPoints := min(eb.comboPoints+pointsToAdd, 5)
	metrics.AddEvent(float64(pointsToAdd), float64(newComboPoints-eb.comboPoints))

	if sim.Log != nil {
		eb.unit.Log(sim, "Gained %d combo points on %s from %s (%d --> %d)", pointsToAdd, eb.comboPointTarget.LogLabel(), metrics.ActionID, eb.comboPoints, newComboPoints)
	}

	eb.comboPoints = newComboPoints

	for _, callback := range eb.onComboPointsGainedCallbacks {
		callback(sim)
	}
}

func (eb *energyBar) AddComboPoints(sim *Simulation, pointsToAdd int32, target *Unit, metrics *ResourceMetrics) {
	newComboPoints := int32(0)
	if eb.comboPointTarget != nil && eb.comboPointTarget != target {
		// we've detected that our target from our last combo point gain and the current target are different! do not pass go, lose all your combo points

		// metric for combo point loss after a "target swap"
		// note that this isn't actually in target swap logic, so we're only triggering combo point loss on an attempted combo point gain
		// on a new target
		metrics.AddEvent(-float64(eb.comboPoints), -float64(eb.comboPoints))

		// handle someone trying to add more than 5??
		newComboPoints = min(pointsToAdd, 5)

		// add new combo point gain
		metrics.AddEvent(float64(pointsToAdd), float64(newComboPoints))

		if sim.Log != nil {
			// TODO there should probably be some separate combo point metric to capture loss
			eb.unit.Log(sim, "Spent %d combo points on %s from %s (%d --> %d) (target swap)", pointsToAdd, eb.comboPointTarget.LogLabel(), metrics.ActionID, eb.comboPoints, 0)
			eb.unit.Log(sim, "Gained %d combo points on %s from %s (%d --> %d)", pointsToAdd, target.LogLabel(), metrics.ActionID, 0, newComboPoints)
		}
	} else {
		newComboPoints = min(eb.comboPoints+pointsToAdd, 5)
		metrics.AddEvent(float64(pointsToAdd), float64(newComboPoints-eb.comboPoints))

		if sim.Log != nil {
			eb.unit.Log(sim, "Gained %d combo points on %s from %s (%d --> %d)", pointsToAdd, target.LogLabel(), metrics.ActionID, eb.comboPoints, newComboPoints)
		}
	}

	eb.comboPoints = newComboPoints

	// overwrite old comboPointTarget to accurately track
	eb.comboPointTarget = target

	for _, callback := range eb.onComboPointsGainedCallbacks {
		callback(sim)
	}
}

func (eb *energyBar) OnComboPointsSpent(callback OnComboPointsSpent) {
	eb.onComboPointsSpentCallbacks = append(eb.onComboPointsSpentCallbacks, callback)
}

func (eb *energyBar) OnComboPointsGained(callback OnComboPointsGained) {
	eb.onComboPointsGainedCallbacks = append(eb.onComboPointsGainedCallbacks, callback)
}

func (eb *energyBar) SpendComboPoints(sim *Simulation, spell *Spell) {
	comboPoints := eb.comboPoints

	if sim.Log != nil {
		eb.unit.Log(sim, "Spent %d combo points from %s (%d --> %d).", comboPoints, spell.ActionID, comboPoints, 0)
	}
	spell.ComboPointMetrics().AddEvent(float64(-comboPoints), float64(-comboPoints))
	eb.comboPoints = 0

	for _, callback := range eb.onComboPointsSpentCallbacks {
		callback(sim, spell, comboPoints)
	}
}

func (eb *energyBar) RunTask(sim *Simulation) time.Duration {
	if sim.CurrentTime < eb.nextWakeAt {
		return eb.nextWakeAt
	}

	eb.accrue()
	bucket := eb.thresholdBucket(eb.currentUnits)
	if eb.cumulativeEnergyDecisionThresholds == nil || bucket != eb.notifiedBucket {
		eb.notifiedBucket = bucket
		eb.notifyAPL(sim)
	}

	eb.nextWakeAt = eb.computeWakeAt()
	return eb.nextWakeAt
}

func (eb *energyBar) reset(sim *Simulation) {
	if eb.unit == nil {
		return
	}

	eb.sim = sim
	eb.currentUnits = eb.maxUnits
	eb.comboPoints = 0
	eb.comboPointTarget = sim.GetTargetUnit(0)
	eb.notifiedBucket = eb.thresholdBucket(eb.currentUnits)

	if eb.unit.Type != PetUnit {
		eb.enable(sim, sim.Environment.PrepullStartTime())
	}
}

func (eb *energyBar) enable(sim *Simulation, startAt time.Duration) {
	eb.active = true
	eb.lastAccrualAt = startAt
	eb.nextWakeAt = NeverExpires
	sim.AddTask(eb)

	if eb.cumulativeEnergyDecisionThresholds != nil && sim.Log != nil {
		eb.unit.Log(sim, "[DEBUG] APL Energy decision thresholds: %v", eb.energyDecisionThresholds)
	}
}

func (eb *energyBar) disable(sim *Simulation) {
	eb.active = false
	eb.nextWakeAt = NeverExpires
	sim.RemoveTask(eb)
}

type EnergyCostOptions struct {
	Cost float64

	Refund        float64
	RefundMetrics *ResourceMetrics // Optional, will default to unit.EnergyRefundMetrics if not supplied.
}
type EnergyCost struct {
	Refund            float64
	RefundMetrics     *ResourceMetrics
	ResourceMetrics   *ResourceMetrics
	ComboPointMetrics *ResourceMetrics
}

func newEnergyCost(spell *Spell, options EnergyCostOptions) *SpellCost {
	if options.Refund > 0 && options.RefundMetrics == nil {
		options.RefundMetrics = spell.Unit.EnergyRefundMetrics
	}
	return &SpellCost{
		spell:      spell,
		BaseCost:   options.Cost,
		Multiplier: 100,
		SpellCostFunctions: &EnergyCost{
			Refund:            options.Refund,
			RefundMetrics:     options.RefundMetrics,
			ResourceMetrics:   spell.Unit.NewEnergyMetrics(spell.ActionID),
			ComboPointMetrics: spell.Unit.NewComboPointMetrics(spell.ActionID),
		},
	}
}

func (ec *EnergyCost) CostType() CostType {
	return CostTypeEnergy
}

func (ec *EnergyCost) MeetsRequirement(_ *Simulation, spell *Spell) bool {
	spell.CurCast.Cost = spell.Cost.GetCurrentCost()
	return spell.Unit.CurrentEnergy() >= spell.CurCast.Cost
}
func (ec *EnergyCost) CostFailureReason(_ *Simulation, spell *Spell) string {
	return fmt.Sprintf("not enough energy (Current Energy = %0.03f, Energy Cost = %0.03f)", spell.Unit.CurrentEnergy(), spell.CurCast.Cost)
}
func (ec *EnergyCost) SpendCost(sim *Simulation, spell *Spell) {
	spell.Unit.SpendEnergy(sim, spell.CurCast.Cost, ec.ResourceMetrics)
}
func (ec *EnergyCost) IssueRefund(sim *Simulation, spell *Spell) {
	if ec.Refund > 0 {
		spell.Unit.AddEnergy(sim, ec.Refund*spell.CurCast.Cost, ec.RefundMetrics)
	}
}

func (spell *Spell) EnergyMetrics() *ResourceMetrics {
	return spell.Cost.SpellCostFunctions.(*EnergyCost).ResourceMetrics
}

func (spell *Spell) ComboPointMetrics() *ResourceMetrics {
	return spell.Cost.SpellCostFunctions.(*EnergyCost).ComboPointMetrics
}
