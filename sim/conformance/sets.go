package conformance

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// SetsGoldenPath is where the set bonus report lives, beside the class
// goldens.
const SetsGoldenPath = GoldenDir + "/sets.golden.md"

// Verdicts of the set bonus report. A bonus is checked against the client's
// ItemSetSpell row; `matches` is the only verdict that means the engine
// applies exactly what the client states.
const (
	VerdictMatches          = "declared, matches"
	VerdictModelled         = "modelled"
	VerdictNoSim            = "no sim effect"
	VerdictMismatch         = "mismatch"
	VerdictMissingThreshold = "threshold missing"
	VerdictUnverified       = "registered, not verified"
	// VerdictUnreachable is a bonus the engine gets wrong (or lacks) on a
	// set that cannot reach its threshold in Phase 1. It is not a failure of
	// this report; the note says what is wrong.
	VerdictUnreachable = "unreachable in Phase 1"
)

// statTolerance is how far an engine stat may sit from the client's
// amount: float noise only.
const statTolerance = 1e-9

// SetBonusRow is one client bonus spell of a registered set.
type SetBonusRow struct {
	SetID     int32
	SetName   string
	Threshold int32
	SpellID   int32
	SpellName string
	Engine    string
	Verdict   string
	Note      string
}

// SetsReport is the whole set bonus conformance result.
type SetsReport struct {
	Build string
	Rows  []SetBonusRow
	// ExtraThresholds are engine thresholds the client's row does not have,
	// as "Name (id): N".
	ExtraThresholds []string
	// Unregistered are client sets with no engine registration.
	Unregistered []string
}

// BuildSetsReport checks every Phase 1 client set against the engine's
// registrations. The engine is whatever Presets registered.
func BuildSetsReport(clientBuild string) SetsReport {
	report := SetsReport{Build: clientBuild}
	registered := core.RegisteredItemSets()
	probe := newSetProbe()

	ids := make([]int32, 0, len(core.ClientSetRows()))
	for id := range core.ClientSetRows() {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		row := core.ClientSetRows()[id]
		engineSet := findRegistered(registered, id, row.Name)
		if engineSet == nil {
			report.Unregistered = append(report.Unregistered, fmt.Sprintf("%s (%d)", row.Name, id))
			continue
		}
		report.Rows = append(report.Rows, probe.rowsFor(id, row, engineSet)...)
		report.ExtraThresholds = append(report.ExtraThresholds, extraThresholds(id, row, engineSet)...)
	}
	return report
}

// findRegistered is the engine's own lookup (character item set counting):
// by id first, then by name.
func findRegistered(registered []*core.ItemSet, id int32, name string) *core.ItemSet {
	for _, set := range registered {
		if set.ID == id {
			return set
		}
	}
	for _, set := range registered {
		if set.Name == name || set.AlternativeName == name {
			return set
		}
	}
	return nil
}

func extraThresholds(id int32, row core.ClientSet, engineSet *core.ItemSet) []string {
	client := map[int32]bool{}
	for _, bonus := range row.Bonuses {
		client[bonus.Threshold] = true
	}
	var extra []string
	for threshold := range engineSet.Bonuses {
		if !client[threshold] {
			extra = append(extra, fmt.Sprintf("%s (%d): %dP", row.Name, id, threshold))
		}
	}
	sort.Strings(extra)
	return extra
}

// setProbe measures what a set bonus does to a character's stats.
type setProbe struct {
	// deltas caches the stat change of one more piece, per (set, pieces).
	deltas map[[2]int32]probeResult
}

type probeResult struct {
	delta  stats.Stats
	preset Preset
	ok     bool
}

func newSetProbe() *setProbe {
	return &setProbe{deltas: map[[2]int32]probeResult{}}
}

// thresholdDelta is the stat change of wearing `threshold` pieces instead of
// threshold-1 on the first preset that can wear the set.
func (p *setProbe) thresholdDelta(setID, threshold int32) probeResult {
	key := [2]int32{setID, threshold}
	if cached, ok := p.deltas[key]; ok {
		return cached
	}
	result := probeResult{}
	for _, preset := range Presets {
		with, err := wearOn(preset, setID, int(threshold))
		if err != nil {
			continue
		}
		without, err := wearOn(preset, setID, int(threshold)-1)
		if err != nil {
			continue
		}
		result.delta = subtract(with, without)
		result.preset = preset
		result.ok = true
		break
	}
	p.deltas[key] = result
	return result
}

func wearOn(preset Preset, setID int32, pieces int) (stats.Stats, error) {
	equipment, database := core.ClientSetTestGear(setID, pieces)
	built, err := buildCharacterWearing(preset, 60, "", equipment, database)
	if err != nil {
		return stats.Stats{}, err
	}
	return built.GetStats(), nil
}

// referenceItemBase is where the ids of the non-set items wearReference
// wears start. The engine's item table keeps the first definition of an id
// for the life of the process, so every reference item needs an id of its
// own.
const referenceItemBase = 9_900_000

var referenceItemCount int32

// wearReference is the stats of preset wearing a single item that carries
// exactly `bonus` as item stats: what the stat pipeline (dependent stats
// such as health from stamina, block and parry from defense) makes of that
// amount, which is what a set bonus of the same size has to come to.
func wearReference(preset Preset, bonus stats.Stats) (stats.Stats, error) {
	referenceItemCount++
	itemID := referenceItemBase + referenceItemCount
	equipment := &proto.EquipmentSpec{Items: make([]*proto.ItemSpec, int(proto.ItemSlot_ItemSlotRanged)+1)}
	for i := range equipment.Items {
		equipment.Items[i] = &proto.ItemSpec{}
	}
	equipment.Items[proto.ItemSlot_ItemSlotHead] = &proto.ItemSpec{Id: itemID}
	database := &proto.SimDatabase{Items: []*proto.SimItem{{
		Id: itemID, Name: "conformance reference", Type: proto.ItemType_ItemTypeHead, Stats: bonus.ToFloatArray(),
	}}}
	built, err := buildCharacterWearing(preset, 60, "", equipment, database)
	if err != nil {
		return stats.Stats{}, err
	}
	return built.GetStats(), nil
}

func subtract(a, b stats.Stats) (out stats.Stats) {
	for i := range out {
		out[i] = a[i] - b[i]
	}
	return out
}

func (p *setProbe) rowsFor(id int32, row core.ClientSet, engineSet *core.ItemSet) []SetBonusRow {
	models := map[int32]core.ClientBonusModel{}
	for _, model := range core.ClientSetModels()[id] {
		models[model.SpellID] = model
	}
	byThreshold := map[int32][]core.ClientSetBonus{}
	for _, bonus := range row.Bonuses {
		byThreshold[bonus.Threshold] = append(byThreshold[bonus.Threshold], bonus)
	}

	var rows []SetBonusRow
	for _, bonus := range row.Bonuses {
		out := SetBonusRow{
			SetID: id, SetName: row.Name, Threshold: bonus.Threshold, SpellID: bonus.SpellID,
			SpellName: core.MustClientSpellRow(bonus.SpellID).Name,
		}
		if _, ok := engineSet.Bonuses[bonus.Threshold]; !ok {
			out.Engine = "none"
			out.Verdict = VerdictMissingThreshold
		} else {
			p.classify(&out, bonus, models, byThreshold[bonus.Threshold])
		}
		markUnreachable(&out, row)
		rows = append(rows, out)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].SetID != rows[j].SetID {
			return rows[i].SetID < rows[j].SetID
		}
		if rows[i].Threshold != rows[j].Threshold {
			return rows[i].Threshold < rows[j].Threshold
		}
		return rows[i].SpellID < rows[j].SpellID
	})
	return rows
}

// markUnreachable downgrades a failing verdict on a bonus the set cannot
// reach in Phase 1 (more pieces than any class can obtain) to
// VerdictUnreachable, keeping what was wrong in the note.
func markUnreachable(out *SetBonusRow, row core.ClientSet) {
	if out.Threshold <= row.PhaseOnePieces {
		return
	}
	if out.Verdict != VerdictMismatch && out.Verdict != VerdictMissingThreshold {
		return
	}
	out.Note = strings.TrimSpace(fmt.Sprintf("%s (%d of %d pieces obtainable) %s", out.Verdict, row.PhaseOnePieces, out.Threshold, out.Note))
	out.Verdict = VerdictUnreachable
}

// classify decides one bonus's verdict. A model the engine recorded decides
// itself; otherwise a flat client bonus is held to the stats the engine
// actually adds at that threshold, and anything else is registered by hand
// and unverified here.
func (p *setProbe) classify(out *SetBonusRow, bonus core.ClientSetBonus, models map[int32]core.ClientBonusModel, atThreshold []core.ClientSetBonus) {
	spell := core.MustClientSpellRow(bonus.SpellID)
	_, isFlat := core.DecodeClientFlatBonus(spell)

	if model, ok := models[bonus.SpellID]; ok {
		switch model.Kind {
		case core.ClientBonusNoSim:
			out.Engine, out.Verdict, out.Note = "none", VerdictNoSim, model.Reason
			return
		case core.ClientBonusModelled:
			out.Engine, out.Verdict = "hand-written", VerdictModelled
			return
		}
	}
	if !isFlat {
		out.Engine, out.Verdict = "hand-written (legacy)", VerdictUnverified
		return
	}

	expected, exact := expectedThresholdStats(atThreshold)
	measured := p.thresholdDelta(out.SetID, bonus.Threshold)
	out.Engine = "stats"
	if !measured.ok {
		out.Verdict, out.Note = VerdictUnverified, "no preset could wear the set"
		return
	}
	want, err := referenceDelta(measured.preset, expected)
	if err != nil {
		out.Verdict, out.Note = VerdictUnverified, err.Error()
		return
	}
	switch {
	case statsAgree(measured.delta, want, exact):
		out.Verdict = VerdictMatches
	default:
		out.Verdict, out.Note = VerdictMismatch, "engine adds "+describeStats(measured.delta)+", client "+describeStats(want)
	}
}

// referenceDelta is what the client's flat stats come to on preset.
func referenceDelta(preset Preset, expected stats.Stats) (stats.Stats, error) {
	with, err := wearReference(preset, expected)
	if err != nil {
		return stats.Stats{}, err
	}
	without, err := wearReference(preset, stats.Stats{})
	if err != nil {
		return stats.Stats{}, err
	}
	return subtract(with, without), nil
}

// expectedThresholdStats is the sum of every flat client bonus at a
// threshold, and whether every bonus at it is flat (so the engine adds
// nothing else).
func expectedThresholdStats(atThreshold []core.ClientSetBonus) (total stats.Stats, exact bool) {
	exact = true
	for _, bonus := range atThreshold {
		flat, ok := core.DecodeClientFlatBonus(core.MustClientSpellRow(bonus.SpellID))
		if !ok {
			exact = false
			continue
		}
		for i := range total {
			total[i] += flat.Stats[i]
		}
	}
	return total, exact
}

// statsAgree holds the engine's stat change to the client's. When every
// bonus at the threshold is flat the change must equal the client's
// exactly; when a hand-written bonus shares the threshold it may add stats
// of its own (spell penetration, say), so the engine need only reach the
// client's amounts.
func statsAgree(got, want stats.Stats, exact bool) bool {
	for i := range want {
		diff := got[i] - want[i]
		if math.Abs(diff) > statTolerance && (exact || diff < 0) {
			return false
		}
	}
	return true
}

func describeStats(s stats.Stats) string {
	var parts []string
	for i, value := range s {
		if math.Abs(value) > statTolerance {
			parts = append(parts, fmt.Sprintf("%s %g", stats.Stat(i).StatName(), value))
		}
	}
	if len(parts) == 0 {
		return "nothing"
	}
	return strings.Join(parts, ", ")
}

// RenderSetsGolden renders the report as markdown.
func RenderSetsGolden(report SetsReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Item set bonuses: engine vs client\n\n")
	fmt.Fprintf(&b, "Client build: %s. Generated by sim/conformance (FOREVER_UPDATE_GOLDEN=1 go test ./sim/conformance/...).\n\n", report.Build)
	fmt.Fprintf(&b, "Every Phase 1 client set (sim/core/client_sets_gen.go: any piece with a loot source not gated later) that the engine registers is listed with each ItemSetSpell bonus. "+
		"`%s`: the bonus is a flat stat (attack power, spell power, healing, stats, resistances, hit, crit, haste, defense, mp5, expertise) and the engine adds exactly the client's amount at that threshold, measured by wearing the set's zero-stat synthetic pieces on a level 60 preset one piece short and at the threshold. "+
		"`%s`: a hand-written effect (proc, modifier, creature-type bonus); its numbers come from the client rows and it is tested in its class package. "+
		"`%s`: a bonus the sim does not model, with the reason (a utility ability, a trigger the sim never produces, or a set that cannot reach the threshold in Phase 1). "+
		"`%s`: a registration that predates this report whose hand-written bonus is not checked here. "+
		"`%s`: the engine is wrong or lacks the bonus on a set that cannot reach the threshold in Phase 1 (the Tier 2 raid helms from Onyxia are one piece each), so it is not a failure; the note says what is wrong. "+
		"`%s` and `%s` are failures.\n\n",
		VerdictMatches, VerdictModelled, VerdictNoSim, VerdictUnverified, VerdictUnreachable, VerdictMismatch, VerdictMissingThreshold)

	fmt.Fprintf(&b, "## Bonuses\n\n| Set | ItemSet | Pieces | Spell | Name | Engine | Verdict | Note |\n|---|---|---|---|---|---|---|---|\n")
	for _, r := range report.Rows {
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %s | %s | %s | %s |\n", r.SetName, r.SetID, r.Threshold, r.SpellID, r.SpellName, r.Engine, r.Verdict, r.Note)
	}

	fmt.Fprintf(&b, "\n## Engine thresholds the client does not have\n\n")
	if len(report.ExtraThresholds) == 0 {
		fmt.Fprintf(&b, "None.\n")
	}
	for _, extra := range report.ExtraThresholds {
		fmt.Fprintf(&b, "- %s\n", extra)
	}

	fmt.Fprintf(&b, "\n## Client sets the engine does not register\n\n")
	if len(report.Unregistered) == 0 {
		fmt.Fprintf(&b, "None.\n")
	}
	for _, name := range report.Unregistered {
		fmt.Fprintf(&b, "- %s\n", name)
	}
	return b.String()
}
