# Status

*Last updated: 2026-06-20*

| Field | Value |
|:--|:--|
| Phase | Phase 1 complete (`jpeg/` lifted + green) |
| Version | none (unreleased; module initialized) |
| Build | `go build ./...` OK; `go vet ./...` clean |
| Tests | `go test ./...` green; `jpeg/` 80.9% coverage |
| Published | not yet |
| Next | handoff §4 Phase 2: `exif/` — resolve engine decision (§7) first, then lift lapis `parseEXIF`/`buildEXIF` + add `Find/Set/Remove/RemoveIFD` |

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

2026-06-20: Updated Go toolchain to 1.22.12. **Phase 0** — `go mod init
codeberg.org/elkarrde/exifscalpel` (`go 1.22`), `doc.go` (package overview + §1
invariants). **Phase 1** — lifted the segment layer from lapis `internal/strip/
strip.go` into `jpeg/` (`Segment`, `Parse`, `Write`, `IsEXIF`, `IsXMP`; sigs kept
unexported). Tests ported from lapis `strip_test.go` builders, covering round-trip,
SOS-tail handling, FF-padding skip, error paths, and the identification predicates.
`go build`/`vet`/`test` all green; `jpeg/` at 80.9% coverage (uncovered lines are
I/O-error returns).

**Decision pending before Phase 2:** the `exif` engine — lift lapis's lean
zero-dependency engine (recommended, matches both tools' no-deps ethos) vs depend on
`dsoprea/go-exif/v3` (handoff §7/§9).

Consumers to migrate once published (decoupled, either first):
[tidy-exif](../tidy-exif/) (Phase 5) and [lapis](../lapis/) (Phase 6).
