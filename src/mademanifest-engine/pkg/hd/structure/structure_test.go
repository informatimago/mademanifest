package structure

import (
	"reflect"
	"testing"

	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/trinity/output"
)

// gatesActivations builds an activation slice that activates the
// given gate numbers under the personality / design distinction.
// Lines are zero unless overridden via per-object_id maps in the
// tests that need them; only profile/incarnation_cross consume the
// line values, the structural derivations do not.
func gatesActivations(gates ...int) []output.HDActivation {
	out := make([]output.HDActivation, 0, len(gates))
	for i, g := range gates {
		// Use canonical object_id slots so byObjectID lookups
		// cover at least sun and earth.  Tests that require those
		// pillars override these names below.
		objectID := "filler"
		switch i {
		case 0:
			objectID = "sun"
		case 1:
			objectID = "earth"
		default:
			objectID = "filler" // unused slot
		}
		out = append(out, output.HDActivation{
			ObjectID: objectID,
			Gate:     g,
			Line:     1,
		})
	}
	return out
}

// requiredPillars returns a [4]activation prefix that satisfies
// Compute's pillar-presence checks (sun + earth on both
// snapshots) without contributing extra channels.  Use a gate
// (2) that is not part of any active channel by default; tests
// override this when the pillars themselves participate in active
// channels.
func requiredPillars(pSun, pEarth, dSun, dEarth output.HDActivation) (personality, design []output.HDActivation) {
	return []output.HDActivation{pSun, pEarth},
		[]output.HDActivation{dSun, dEarth}
}

// TestComputeReflectorWhenNoChannelsActive synthesises an activation
// set with every gate fired except the second gate of any canonical
// channel — i.e. every gate is alone, no channel can be complete,
// every center is undefined.  Definition = none, type = reflector,
// authority = lunar.
func TestComputeReflectorWhenNoChannelsActive(t *testing.T) {
	// Activate gates 1, 2, 3 (chosen to be canonical channel halves
	// without their partners): gate 1's partner is 8, gate 2's is
	// 14, gate 3's is 60.  None of those partners are in the set.
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 1, Line: 4},
		{ObjectID: "earth", Gate: 2, Line: 5},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 3, Line: 2},
		{ObjectID: "earth", Gate: 4, Line: 1},
	}
	res, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(res.Channels) != 0 {
		t.Errorf("Channels = %+v, want empty", res.Channels)
	}
	for _, c := range res.Centers {
		if c.State != "undefined" {
			t.Errorf("center %s state = %q, want undefined", c.CenterID, c.State)
		}
	}
	if res.Definition != "none" {
		t.Errorf("Definition = %q, want none", res.Definition)
	}
	if res.Type != "reflector" {
		t.Errorf("Type = %q, want reflector", res.Type)
	}
	if res.Authority != "lunar" {
		t.Errorf("Authority = %q, want lunar", res.Authority)
	}
	if res.Profile != "4/2" {
		t.Errorf("Profile = %q, want 4/2", res.Profile)
	}
	wantCross := output.HDIncarnationCross{
		PersonalitySun:   output.HDGateLine{Gate: 1, Line: 4},
		PersonalityEarth: output.HDGateLine{Gate: 2, Line: 5},
		DesignSun:        output.HDGateLine{Gate: 3, Line: 2},
		DesignEarth:      output.HDGateLine{Gate: 4, Line: 1},
	}
	if res.IncarnationCross != wantCross {
		t.Errorf("IncarnationCross = %+v, want %+v", res.IncarnationCross, wantCross)
	}
}

// TestComputeGenerator activates channel 5-15 only (sacral ↔ g),
// but no motor-to-throat path: sacral is defined, throat is not.
func TestComputeGenerator(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 5, Line: 3},
		{ObjectID: "earth", Gate: 11, Line: 1},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 15, Line: 5},
		{ObjectID: "earth", Gate: 12, Line: 2},
	}
	res, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Type != "generator" {
		t.Errorf("Type = %q, want generator", res.Type)
	}
	if res.Authority != "sacral" {
		t.Errorf("Authority = %q, want sacral", res.Authority)
	}
	if res.Definition != "single" {
		t.Errorf("Definition = %q, want single", res.Definition)
	}
	if res.Profile != "3/5" {
		t.Errorf("Profile = %q, want 3/5", res.Profile)
	}
	checkCenterState(t, res.Centers, map[string]string{
		"sacral": "defined", "g": "defined",
		"throat": "undefined", "head": "undefined",
		"solar_plexus": "undefined", "spleen": "undefined",
	})
}

// TestComputeManifestingGenerator activates 20-34 (throat ↔ sacral)
// — sacral defined AND throat connected to a motor (sacral itself).
func TestComputeManifestingGenerator(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 20, Line: 1},
		{ObjectID: "earth", Gate: 11, Line: 6},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 34, Line: 4},
		{ObjectID: "earth", Gate: 12, Line: 3},
	}
	res, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Type != "manifesting_generator" {
		t.Errorf("Type = %q, want manifesting_generator", res.Type)
	}
	if res.Authority != "sacral" {
		t.Errorf("Authority = %q, want sacral", res.Authority)
	}
}

// TestComputeManifestor activates 21-45 (ego ↔ throat).  Ego is a
// non-sacral motor; throat is connected to it.  Sacral undefined.
// Type = manifestor, authority = ego_manifested (since ego defined).
func TestComputeManifestor(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 21, Line: 2},
		{ObjectID: "earth", Gate: 11, Line: 1},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 45, Line: 4},
		{ObjectID: "earth", Gate: 12, Line: 5},
	}
	res, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Type != "manifestor" {
		t.Errorf("Type = %q, want manifestor", res.Type)
	}
	if res.Authority != "ego_manifested" {
		t.Errorf("Authority = %q, want ego_manifested", res.Authority)
	}
}

// TestComputeProjector activates 8-1 (g ↔ throat).  No sacral, no
// motor-to-throat path (g is not a motor).  Type = projector.
// Authority = self_projected since g is defined.
func TestComputeProjector(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 1, Line: 6},
		{ObjectID: "earth", Gate: 11, Line: 1},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 8, Line: 2},
		{ObjectID: "earth", Gate: 12, Line: 3},
	}
	res, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Type != "projector" {
		t.Errorf("Type = %q, want projector", res.Type)
	}
	if res.Authority != "self_projected" {
		t.Errorf("Authority = %q, want self_projected", res.Authority)
	}
}

// TestComputeDefinitionSplit activates two disjoint channels:
// 5-15 (sacral-g) and 21-45 (ego-throat).  Two components.
func TestComputeDefinitionSplit(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 5, Line: 1},
		{ObjectID: "earth", Gate: 11, Line: 2},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 15, Line: 3},
		{ObjectID: "earth", Gate: 12, Line: 4},
	}
	// add 21 + 45 as fillers
	personality = append(personality, output.HDActivation{ObjectID: "p21", Gate: 21, Line: 1})
	design = append(design, output.HDActivation{ObjectID: "d45", Gate: 45, Line: 1})
	res, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Definition != "split" {
		t.Errorf("Definition = %q, want split (2 components)", res.Definition)
	}
}

// TestComputeDefinitionTripleSplit activates three disjoint
// channels: 5-15 (sacral-g), 21-45 (ego-throat), 4-63 (ajna-head).
func TestComputeDefinitionTripleSplit(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 5, Line: 1},
		{ObjectID: "earth", Gate: 11, Line: 1},
		{ObjectID: "p21", Gate: 21, Line: 1},
		{ObjectID: "p4", Gate: 4, Line: 1},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 15, Line: 1},
		{ObjectID: "earth", Gate: 12, Line: 1},
		{ObjectID: "d45", Gate: 45, Line: 1},
		{ObjectID: "d63", Gate: 63, Line: 1},
	}
	res, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Definition != "triple_split" {
		t.Errorf("Definition = %q, want triple_split", res.Definition)
	}
}

// TestComputeDefinitionQuadrupleSplit activates four disjoint
// channels:
//   5-15 (sacral-g), 21-45 (ego-throat), 4-63 (ajna-head),
//   18-58 (spleen-root).
func TestComputeDefinitionQuadrupleSplit(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 5, Line: 1},
		{ObjectID: "earth", Gate: 11, Line: 1},
		{ObjectID: "p21", Gate: 21, Line: 1},
		{ObjectID: "p4", Gate: 4, Line: 1},
		{ObjectID: "p18", Gate: 18, Line: 1},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 15, Line: 1},
		{ObjectID: "earth", Gate: 12, Line: 1},
		{ObjectID: "d45", Gate: 45, Line: 1},
		{ObjectID: "d63", Gate: 63, Line: 1},
		{ObjectID: "d58", Gate: 58, Line: 1},
	}
	res, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Definition != "quadruple_split" {
		t.Errorf("Definition = %q, want quadruple_split (4+ components)", res.Definition)
	}
}

// TestComputeDefinitionSingleViaTwoConnectedChannels activates
// 5-15 and 2-14.  Both share the sacral and g centers, so the
// component union is {sacral, g}.  Definition = single.
func TestComputeDefinitionSingleViaTwoConnectedChannels(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 2, Line: 1},
		{ObjectID: "earth", Gate: 11, Line: 1},
		{ObjectID: "p5", Gate: 5, Line: 1},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 14, Line: 1},
		{ObjectID: "earth", Gate: 12, Line: 1},
		{ObjectID: "d15", Gate: 15, Line: 1},
	}
	res, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Definition != "single" {
		t.Errorf("Definition = %q, want single (one merged component)", res.Definition)
	}
}

// TestComputeAuthorityEmotionalWinsOverSacral activates 6-59
// (solar_plexus ↔ sacral): sacral defined and solar_plexus defined.
// Per the canon priority list, solar_plexus wins → emotional.
func TestComputeAuthorityEmotionalWinsOverSacral(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 6, Line: 1},
		{ObjectID: "earth", Gate: 11, Line: 1},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 59, Line: 1},
		{ObjectID: "earth", Gate: 12, Line: 1},
	}
	res, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Type != "manifesting_generator" {
		// 6-59 is a motor-to-throat? solar_plexus is a motor, but
		// throat is not in the component.  So no MG; it's a
		// generator.
		if res.Type != "generator" {
			t.Fatalf("Type = %q, want generator", res.Type)
		}
	}
	if res.Authority != "emotional" {
		t.Errorf("Authority = %q, want emotional (solar_plexus pre-empts sacral)", res.Authority)
	}
}

// TestComputeAuthoritySplenicForProjector activates 16-48
// (throat ↔ spleen): no sacral, no solar_plexus, spleen defined.
// Throat is connected to spleen but spleen is not a motor; so type
// = projector and authority falls to splenic.
func TestComputeAuthoritySplenicForProjector(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 16, Line: 1},
		{ObjectID: "earth", Gate: 11, Line: 1},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 48, Line: 1},
		{ObjectID: "earth", Gate: 12, Line: 1},
	}
	res, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Type != "projector" {
		t.Fatalf("Type = %q, want projector", res.Type)
	}
	if res.Authority != "splenic" {
		t.Errorf("Authority = %q, want splenic", res.Authority)
	}
}

// TestComputeAuthorityEgoProjected activates 25-51 (g ↔ ego): no
// sacral, no solar_plexus, no spleen; type = projector with ego
// defined → ego_projected.
func TestComputeAuthorityEgoProjected(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 25, Line: 1},
		{ObjectID: "earth", Gate: 11, Line: 1},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 51, Line: 1},
		{ObjectID: "earth", Gate: 12, Line: 1},
	}
	res, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Type != "projector" {
		t.Fatalf("Type = %q, want projector", res.Type)
	}
	if res.Authority != "ego_projected" {
		t.Errorf("Authority = %q, want ego_projected", res.Authority)
	}
}

// TestComputeAuthorityMental fires 4-63 (ajna ↔ head) only: no
// sacral, no solar_plexus, no spleen, no ego, no g.  Type = projector
// with only head + ajna defined → mental.
func TestComputeAuthorityMental(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 4, Line: 1},
		{ObjectID: "earth", Gate: 11, Line: 1},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 63, Line: 1},
		{ObjectID: "earth", Gate: 12, Line: 1},
	}
	res, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Type != "projector" {
		t.Fatalf("Type = %q, want projector", res.Type)
	}
	if res.Authority != "mental" {
		t.Errorf("Authority = %q, want mental", res.Authority)
	}
}

// TestComputeChannelsSortedByID activates two non-adjacent channels
// (12-22 and 1-8) and verifies the channels array is sorted
// lexicographically by channel_id (so "1-8" precedes "12-22").
func TestComputeChannelsSortedByID(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 12, Line: 1},
		{ObjectID: "earth", Gate: 11, Line: 1},
		{ObjectID: "p1", Gate: 1, Line: 1},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 22, Line: 1},
		{ObjectID: "earth", Gate: 13, Line: 1},
		{ObjectID: "d8", Gate: 8, Line: 1},
	}
	res, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	got := make([]string, len(res.Channels))
	for i, c := range res.Channels {
		got[i] = c.ChannelID
	}
	want := []string{"1-8", "12-22"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ChannelIDs = %v, want %v", got, want)
	}
}

// TestComputeCentersOrdering pins the canon center ordering even
// when no channels are active.  Every center must appear, in
// canon.CenterOrder.
func TestComputeCentersOrdering(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 1, Line: 1},
		{ObjectID: "earth", Gate: 2, Line: 1},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 3, Line: 1},
		{ObjectID: "earth", Gate: 4, Line: 1},
	}
	res, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(res.Centers) != len(canon.CenterOrder) {
		t.Fatalf("len(Centers) = %d, want %d", len(res.Centers), len(canon.CenterOrder))
	}
	for i, c := range res.Centers {
		if c.CenterID != canon.CenterOrder[i] {
			t.Errorf("Centers[%d].CenterID = %q, want %q", i, c.CenterID, canon.CenterOrder[i])
		}
	}
}

// checkCenterState fails the test if any center's state diverges
// from the expected map.  Centers absent from want are not checked.
func checkCenterState(t *testing.T, centers []output.HDCenter, want map[string]string) {
	t.Helper()
	got := make(map[string]string, len(centers))
	for _, c := range centers {
		got[c.CenterID] = c.State
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("center %s state = %q, want %q", id, got[id], w)
		}
	}
}

// TestComputeMissingPillarErrors guards the engine's
// engine-internal misuse path: if either snapshot lacks one of the
// four pillars (personality sun + earth, design sun + earth),
// Compute must return an error rather than silently emit a
// zero-valued IncarnationCross.
func TestComputeMissingPillarErrors(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 1, Line: 1},
		// earth missing
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 2, Line: 1},
		{ObjectID: "earth", Gate: 3, Line: 1},
	}
	if _, err := Compute(personality, design); err == nil {
		t.Errorf("Compute with missing personality earth: err = nil, want non-nil")
	}
}

// silence unused-helper warning when developing; requiredPillars is
// kept for future tests that need a clean pillar foundation
// distinct from the channel-driving gates.
var _ = requiredPillars
