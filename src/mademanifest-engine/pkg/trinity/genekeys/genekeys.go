// Package genekeys derives the canonical Trinity Gene Keys block
// from the personality + design Human Design activations produced
// by Phase 6 (pkg/trinity/hd).  trinity.org §"Gene Keys"
// (lines 139-153) and §"Gene Keys Output" (lines 547-558) pin the
// derivation:
//
//   * Gene Keys is not an independent astronomical system – it
//     reads directly from HD activations.
//   * Four canonical positions:
//       life_work = personality_sun
//       evolution = personality_earth
//       radiance  = design_sun
//       purpose   = design_earth
//   * Each position carries exactly two integer fields:
//       key  = the HD gate number
//       line = the HD line number
//   * No text / shadow / gift / essence / sequence / semantic-state
//     fields appear in v1 output.
//   * system.derivation_basis is the literal string "human_design".
//
// Phase 8 retires the PoC field name "lifes_work" in favour of the
// canonical "life_work"; the rename is enforced by the Phase 3
// output type GKActivations.LifeWork.
package genekeys

import (
	"fmt"

	"mademanifest-engine/pkg/trinity/output"
)

// Compute returns the canonical Gene Keys block.  It is a pure
// function of the four pillar activations (personality sun + earth,
// design sun + earth) extracted from the HD activation snapshots;
// every other body in the snapshots is irrelevant for Gene Keys.
//
// Errors here flag engine-internal misuse: a missing pillar means
// pkg/trinity/hd.ComputeActivations did not emit one of the canon
// HDSnapshotOrder bodies, which is a bug in the HD pipeline rather
// than a user-facing input problem.  The HTTP handler treats this
// as execution_failure (HTTP 500).
func Compute(personality, design []output.HDActivation) (output.GeneKeysOut, error) {
	pSun, err := pillar(personality, "sun", "personality")
	if err != nil {
		return output.GeneKeysOut{}, err
	}
	pEarth, err := pillar(personality, "earth", "personality")
	if err != nil {
		return output.GeneKeysOut{}, err
	}
	dSun, err := pillar(design, "sun", "design")
	if err != nil {
		return output.GeneKeysOut{}, err
	}
	dEarth, err := pillar(design, "earth", "design")
	if err != nil {
		return output.GeneKeysOut{}, err
	}

	return output.GeneKeysOut{
		System: output.GKSystem{
			DerivationBasis: "human_design",
		},
		Activations: output.GKActivations{
			LifeWork:  output.GKActivation{Key: pSun.Gate, Line: pSun.Line},
			Evolution: output.GKActivation{Key: pEarth.Gate, Line: pEarth.Line},
			Radiance:  output.GKActivation{Key: dSun.Gate, Line: dSun.Line},
			Purpose:   output.GKActivation{Key: dEarth.Gate, Line: dEarth.Line},
		},
	}, nil
}

// pillar pulls one named activation out of a snapshot slice and
// returns a descriptive error when the named body is missing.  The
// snapshot slices the engine produces always contain exactly the
// 13 canon HDSnapshotOrder bodies, so in production the error path
// is unreachable; tests rely on it for clean fixture diagnostics.
func pillar(acts []output.HDActivation, objectID, snapshot string) (output.HDActivation, error) {
	for _, a := range acts {
		if a.ObjectID == objectID {
			return a, nil
		}
	}
	return output.HDActivation{}, fmt.Errorf(
		"genekeys: missing %s activation %q (snapshot has %d entries)",
		snapshot, objectID, len(acts))
}
