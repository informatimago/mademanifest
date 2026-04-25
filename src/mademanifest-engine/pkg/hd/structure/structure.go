// Package structure derives the canonical Trinity Human Design
// structural layer from the combined activation set produced by
// pkg/trinity/hd in Phase 6.
//
// Phase 7 of the implementation plan
// (src/doc/trinity-implementation-plan.org) pins the deliverables:
//
//   * channels        – dedupe + sort by channel_id (lexicographic)
//   * centers         – ordered head..root, each with state ∈ {defined,undefined}
//   * definition      – none | single | split | triple_split | quadruple_split
//   * type            – reflector | generator | manifesting_generator | manifestor | projector
//   * authority       – eight-way priority list per Document 05
//   * profile         – "personality_sun_line/design_sun_line"
//   * incarnation_cross – structural {p_sun, p_earth, d_sun, d_earth}
//
// Determinism: results depend only on the activation arrays passed
// in and the compiled-in canon constants (canon.ChannelTable,
// canon.CenterOrder, canon.MotorCenters).  No environment, no I/O.
package structure

import (
	"fmt"
	"sort"

	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/trinity/output"
)

// Result bundles the seven structural sub-fields the HTTP handler
// drops into the success envelope.  Field order matches the canon
// HumanDesignOut struct so Result can be applied to an envelope
// with assignment statements rather than struct-conversion shims.
type Result struct {
	Channels         []output.HDChannel
	Centers          []output.HDCenter
	Definition       string
	Type             string
	Authority        string
	Profile          string
	IncarnationCross output.HDIncarnationCross
}

// Compute returns the canonical structural derivation for a pair of
// activation snapshots.  The two slices may be in any order; the
// function pulls the four pillar activations out by object_id.
//
// Errors are surfaced for engine-internal misuse (missing required
// pillars from the activation set).  In production these are
// impossible because pkg/trinity/hd.ComputeActivations always emits
// all 13 canon snapshot bodies; the error path exists so tests can
// fail loudly on missing fixtures rather than producing zero-valued
// HDIncarnationCross silently.
func Compute(personality, design []output.HDActivation) (Result, error) {
	personalityByID := byObjectID(personality)
	designByID := byObjectID(design)

	pSun, ok := personalityByID["sun"]
	if !ok {
		return Result{}, fmt.Errorf("structure: missing personality sun activation")
	}
	pEarth, ok := personalityByID["earth"]
	if !ok {
		return Result{}, fmt.Errorf("structure: missing personality earth activation")
	}
	dSun, ok := designByID["sun"]
	if !ok {
		return Result{}, fmt.Errorf("structure: missing design sun activation")
	}
	dEarth, ok := designByID["earth"]
	if !ok {
		return Result{}, fmt.Errorf("structure: missing design earth activation")
	}

	// Combined active gate set: union of personality and design
	// gates.  Lines are irrelevant for channel detection; only gate
	// numbers participate in the canonical channel table.
	activeGates := make(map[int]bool, len(personality)+len(design))
	for _, a := range personality {
		activeGates[a.Gate] = true
	}
	for _, a := range design {
		activeGates[a.Gate] = true
	}

	channels := activeChannels(activeGates)
	centerStates := centerStateMap(channels)
	centers := emitCenters(centerStates)
	components := connectedComponents(channels, centerStates)
	definition := definitionClass(len(components))
	hdType := typeFor(centerStates, components)
	authority := authorityFor(hdType, centerStates)
	profile := fmt.Sprintf("%d/%d", pSun.Line, dSun.Line)
	cross := output.HDIncarnationCross{
		PersonalitySun:   output.HDGateLine{Gate: pSun.Gate, Line: pSun.Line},
		PersonalityEarth: output.HDGateLine{Gate: pEarth.Gate, Line: pEarth.Line},
		DesignSun:        output.HDGateLine{Gate: dSun.Gate, Line: dSun.Line},
		DesignEarth:      output.HDGateLine{Gate: dEarth.Gate, Line: dEarth.Line},
	}

	return Result{
		Channels:         channels,
		Centers:          centers,
		Definition:       definition,
		Type:             hdType,
		Authority:        authority,
		Profile:          profile,
		IncarnationCross: cross,
	}, nil
}

// byObjectID indexes an activation slice by its object_id.  The
// activation arrays the engine produces always have unique
// object_ids, so the map values are unambiguous.
func byObjectID(acts []output.HDActivation) map[string]output.HDActivation {
	m := make(map[string]output.HDActivation, len(acts))
	for _, a := range acts {
		m[a.ObjectID] = a
	}
	return m
}

// activeChannels walks canon.ChannelTable in declaration order,
// emits each channel whose two gates both appear in activeGates,
// and returns the survivors sorted by channel_id (lexicographic).
//
// canon.ChannelTable is already sorted by canonical channel_id, so
// the output of this helper preserves that order without an
// explicit sort step.  The defensive sort below is a guard against
// future canon edits that re-order the table.
func activeChannels(activeGates map[int]bool) []output.HDChannel {
	out := make([]output.HDChannel, 0, len(canon.ChannelTable))
	for _, c := range canon.ChannelTable {
		if activeGates[c.GateA] && activeGates[c.GateB] {
			out = append(out, output.HDChannel{
				ChannelID: c.ID,
				GateA:     c.GateA,
				GateB:     c.GateB,
				CenterA:   c.CenterA,
				CenterB:   c.CenterB,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ChannelID < out[j].ChannelID
	})
	return out
}

// centerStateMap returns a name → bool map where true means the
// center is "defined" (i.e. participates in at least one active
// channel).  Centers absent from the map are undefined.
func centerStateMap(channels []output.HDChannel) map[string]bool {
	defined := make(map[string]bool, len(canon.CenterOrder))
	for _, c := range channels {
		defined[c.CenterA] = true
		defined[c.CenterB] = true
	}
	return defined
}

// emitCenters returns the canonical 9-entry centers array in
// canon.CenterOrder, each with state "defined" or "undefined".
func emitCenters(defined map[string]bool) []output.HDCenter {
	out := make([]output.HDCenter, 0, len(canon.CenterOrder))
	for _, id := range canon.CenterOrder {
		state := "undefined"
		if defined[id] {
			state = "defined"
		}
		out = append(out, output.HDCenter{
			CenterID: id,
			State:    state,
		})
	}
	return out
}

// connectedComponents returns the connected components of the graph
// whose vertices are the *defined* centers and whose edges are the
// active channels.  Each component is the set of center_ids in that
// component.  Components are returned with each component's centers
// sorted (canon CenterOrder index) so the output ordering is
// deterministic; the slice itself is sorted by lowest-index center.
//
// Implementation: textbook union-find over center indices in
// canon.CenterOrder.  We could equivalently run a BFS, but
// union-find scales linearly in channel count and avoids building
// an adjacency list.
func connectedComponents(channels []output.HDChannel, defined map[string]bool) [][]string {
	idx := make(map[string]int, len(canon.CenterOrder))
	for i, id := range canon.CenterOrder {
		idx[id] = i
	}
	parent := make([]int, len(canon.CenterOrder))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[ra] = rb
		}
	}
	for _, c := range channels {
		union(idx[c.CenterA], idx[c.CenterB])
	}

	// Group defined centers by root.
	groups := make(map[int][]int)
	for id, ok := range defined {
		if !ok {
			continue
		}
		i := idx[id]
		root := find(i)
		groups[root] = append(groups[root], i)
	}

	out := make([][]string, 0, len(groups))
	for _, members := range groups {
		sort.Ints(members)
		names := make([]string, len(members))
		for i, m := range members {
			names[i] = canon.CenterOrder[m]
		}
		out = append(out, names)
	}
	sort.Slice(out, func(i, j int) bool {
		return idx[out[i][0]] < idx[out[j][0]]
	})
	return out
}

// definitionClass maps the number of connected components in the
// defined-centers graph to the canonical definition string.  Per
// trinity.org lines 312-317:
//
//   * 0 components → "none"
//   * 1 component  → "single"
//   * 2 components → "split"
//   * 3 components → "triple_split"
//   * 4 components → "quadruple_split"
//
// The canon does not enumerate a "five-or-more" class; a fifth
// component is theoretically possible only with very fragmented
// activation sets.  We surface that as "quadruple_split" rather
// than inventing a sixth class, mirroring the canon list cap.
func definitionClass(componentCount int) string {
	switch componentCount {
	case 0:
		return "none"
	case 1:
		return "single"
	case 2:
		return "split"
	case 3:
		return "triple_split"
	default:
		return "quadruple_split"
	}
}

// typeFor implements trinity.org §"Type derivation" (lines 318-325).
// The decision tree:
//
//   1. definition == none           → reflector
//   2. sacral defined:
//        2a. motor-to-throat path   → manifesting_generator
//        2b. otherwise              → generator
//   3. non-sacral motor connects to throat → manifestor
//   4. otherwise                            → projector
//
// "Motor-to-throat path" means throat sits in the same connected
// component as one of the motor centers (root, sacral, solar_plexus,
// ego), via a chain of active channels.  We compute that on the
// component graph rather than re-walking the channel list.
func typeFor(defined map[string]bool, components [][]string) string {
	if len(components) == 0 {
		return "reflector"
	}
	throatComponent := componentContaining("throat", components)
	if defined["sacral"] {
		if throatComponent >= 0 && componentContainsAnyMotor(components[throatComponent]) {
			return "manifesting_generator"
		}
		return "generator"
	}
	if throatComponent >= 0 && componentContainsNonSacralMotor(components[throatComponent]) {
		return "manifestor"
	}
	return "projector"
}

// authorityFor implements trinity.org §"Authority derivation
// priority" (lines 326-334).  The first matching rule wins.
func authorityFor(hdType string, defined map[string]bool) string {
	switch {
	case defined["solar_plexus"]:
		return "emotional"
	case hdType == "generator" || hdType == "manifesting_generator":
		return "sacral"
	case defined["spleen"]:
		return "splenic"
	case hdType == "manifestor" && defined["ego"]:
		return "ego_manifested"
	case hdType == "projector" && defined["ego"]:
		return "ego_projected"
	case hdType == "projector" && defined["g"]:
		return "self_projected"
	case hdType == "projector":
		return "mental"
	case hdType == "reflector":
		return "lunar"
	default:
		// Per the canon priority list, this branch is unreachable:
		// every type maps to at least one of the eight values
		// above.  We fall back to "lunar" defensively to keep the
		// output strictly inside the canon-allowed set if a future
		// canon revision introduces a new type without updating
		// this priority list.
		return "lunar"
	}
}

// componentContaining returns the index of the component containing
// the named center, or -1 if the center is not in any (i.e. is
// undefined).
func componentContaining(name string, components [][]string) int {
	for i, comp := range components {
		for _, c := range comp {
			if c == name {
				return i
			}
		}
	}
	return -1
}

// componentContainsAnyMotor returns true if the component contains
// any of canon.MotorCenters (root, sacral, solar_plexus, ego).
func componentContainsAnyMotor(component []string) bool {
	for _, c := range component {
		for _, m := range canon.MotorCenters {
			if c == m {
				return true
			}
		}
	}
	return false
}

// componentContainsNonSacralMotor returns true if the component
// contains a motor center other than sacral.  Used by the
// manifestor branch of the type decision tree, which fires only
// when sacral is undefined.
func componentContainsNonSacralMotor(component []string) bool {
	for _, c := range component {
		if c == "sacral" {
			continue
		}
		for _, m := range canon.MotorCenters {
			if c == m {
				return true
			}
		}
	}
	return false
}
