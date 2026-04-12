Swiss Ephemeris build and linkage contract for PoC V2

Canonical source:
- Version: 2.10.03
- Archive: swisseph-2.10.03.zip (GitHub release)

Allowed integration approaches:
A) CGO build from the provided Swiss Ephemeris source archive
B) A Go Swiss Ephemeris binding that is proven to use Swiss Ephemeris 2.10.03

Non negotiable constraints:
- The effective Swiss Ephemeris version in use must be 2.10.03
- The implementation must use only the provided ephemeris data files from:
  ephemeris/data/REQUIRED_EPHEMERIS_FILES
- All runtime behavior must follow ephemeris/swisseph/FLAGS.md

Notes:
- This PoC does not require any packaging for production.
- The only requirement is that the Golden Test output matches exactly with a deterministic pipeline.
- If a binding is used, the candidate must state precisely how version 2.10.03 is ensured.
