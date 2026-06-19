# Status

*Last updated: 2026-06-20*

| Field | Value |
|:--|:--|
| Phase | pre-code / planning (repo scaffolded) |
| Version | none (no `go.mod` yet) |
| Build | n/a (no Go code) |
| Tests | n/a |
| Published | not yet |
| Next | handoff §4 Phase 0–1: `go mod init` (Go 1.22), build `jpeg/` segment layer |

## Notes

Repo initialized 2026-06-19 with scaffolding only (LICENSE, README, STATUS,
CHANGELOG, `.gitignore`). No Go code yet. The full build plan — provenance, package
layout (`jpeg`/`exif`/`xmp`), API sketch, 7-phase migration, tests, prior-art — is
in `exifscalpel-HANDOFF.md`.

2026-06-20: Added `CLAUDE.md` (orients future instances to the handoff). Verified
the handoff's lift list against current tidy-exif: paths are accurate — tidy-exif's
metadata engine now lives in `internal/meta/`. Only the XMP layer lifts from
tidy-exif (`internal/meta/xmp.go`, ~250 of 323 lines + `xmp_test.go`); the `jpeg`
and `exif` engines come from lapis. Fixed three stale top-level paths in handoff §0
provenance table (`jpeg.go`/`xmp.go`/`exif.go` → `internal/meta/…`). Still pre-code.

**Decision pending before Phase 2:** the `exif` engine — lift lapis's lean
zero-dependency engine (recommended, matches both tools' no-deps ethos) vs depend on
`dsoprea/go-exif/v3` (handoff §7/§9).

Consumers to migrate once published (decoupled, either first):
[tidy-exif](../tidy-exif/) (Phase 5) and [lapis](../lapis/) (Phase 6).
