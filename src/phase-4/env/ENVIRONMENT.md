Environment contract for PoC V2

- Language: Go only
- Go version: see GO_VERSION.txt
- Output must be reproducible on a clean machine using only this bundle

Non negotiable:
- Use only the Swiss Ephemeris artifact provided in ephemeris/swisseph
- Use only the ephemeris data files provided in ephemeris/data/REQUIRED_EPHEMERIS_FILES
- Do not rely on system installed Swiss Ephemeris or system ephemeris files
- No external APIs or services
- No hidden downloads during execution
