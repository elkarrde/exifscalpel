# Status

*Last updated: 2026-06-24*

| Field | Value |
|:--|:--|
| Phase | Phase 6 done (lapis migrated, green). Phase 4 `v0.1.0` published; Phase 5 (tidy-exif) handoff written, not yet executed |
| Version | `v0.1.0` (tagged + pushed; resolvable via Go proxy) |
| Build | `go build`/`vet`/`gofmt` clean; main module has **no `go.sum`** (zero runtime deps) |
| Tests | `go test ./...` green (`jpeg/` 80.9%, `exif/` 88.6%, `xmp/` 86.4%); `go -C conformance test ./...` green |
| Published | `v0.1.0` pushed; lapis now requires it (zero transitive deps) |
| Next | **START HERE → Phase 5: execute the tidy-exif migration** per `../tidy-exif/MIGRATION.md` |

## ▶ Next session — start here

**Phase 5: migrate tidy-exif onto `exifscalpel`.** The full step-by-step (go.mod
bump 1.16→1.22, symbol-by-symbol mapping, what stays as policy, tests, acceptance
gate) is written in **`../tidy-exif/MIGRATION.md`**. lapis (Phase 6) is the worked
reference for the same pattern — see `../lapis/internal/strip/{strip,exif}.go`.

Key facts for tidy-exif: the CLI (`cmd/tidy-exif/`) needs **no changes** (its
`report.XMP.{CreatorTool,SoftwareAgents}` accesses line up with `xmp.Fields`); only
`internal/meta/` changes. Keep `isAdobeSoftware` (the Adobe-only gate) and
`InspectJPEG`/`CleanJPEG` (orchestration) as policy.

**Phase 6 done (2026-06-24):** lapis migrated. `internal/strip/{strip,exif}.go` now
import `exifscalpel/{jpeg,exif}`; the engine code is deleted and all policy
(`Level`/`ParseLevel`/`Strip`/`processSegments`, the keep/remove tag maps) stays.
`strip_test.go` rebuilt on library types. `go build`/`vet`/`gofmt`/`test ./...`
green; `go mod graph` shows one new edge (exifscalpel, zero transitive deps).
**Changes are uncommitted in `../lapis/`** — commit when ready.

Optional polish: extend `conformance/` per its README "Next" (XMP via `exiftool`
oracle, fuzzing). Phase 7 (optional): new lapis level that scrubs XMP fields via
`xmp.Clean` instead of excising the segment.

Decisions are all locked (handoff §7); no open questions blocking Phase 4.

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

2026-06-21: **Phase 2** — `exif/` package. Resolved the §7 open decisions (recorded
in handoff §7): **lapis zero-dep engine**, **mutate `*Data`** via
`Find/Set/Remove/RemoveIFD`, **both edit modes exposed**. Rebuild = `(*Data).Build`
(lifts lapis `parseEXIF`/`buildEXIF`; length may change; `Build` now self-reconciles
sub-IFD pointers). In-place length-preserving = `OverwriteValueInPlace`/`ReadValue`
(re-expressed from tidy-exif `cleanExifSoftware`, generalized to any IFD0 tag).
Exported tags: `SoftwareTag`, `ExifIFDPointer`, `GPSIFDPointer`. Tests cover both
byte orders (II/MM), in-place inline vs offset values, rebuild scrub, ported lapis
GPS IFD removal, and parse round-trip. `exif/` 88.6% coverage. Left lapis's
journalist/scout filter maps behind (policy).

2026-06-21: Recorded the **dependency policy** (`CONTRIBUTING.md`, CLAUDE.md, handoff
§10): "no deps" = zero *runtime* baggage; dev/test tooling is unrestricted but heavy
deps go in a side module. Added **`conformance/`** — a separate Go module
(`replace … => ../`) with differential EXIF tests vs. `dsoprea/go-exif/v3`. Two tests
green: a reference reader parses our `Build` output, and confirms GPS removal +
Software scrub. Isolation verified: main module still has no `go.sum`; `./...` does
not descend into `conformance/`.

Consumers to migrate once published (decoupled, either first):
[tidy-exif](../tidy-exif/) (Phase 5) and [lapis](../lapis/) (Phase 6).

2026-06-23: **Phase 3** — `xmp/` package, lifted from tidy-exif
`internal/meta/xmp.go`. Exported `Parse(payload) (*Fields, error)`,
`Clean(payload, replacements) (out, changed, err)`, `Fields`/`Fields.Any()`.
Lifted `cleanFields`/`marshal`/`patchField`/`patchAll`/`adjustPadding`/
`matchAttr`/`nextCharData` + `ns*` consts unexported. **Kept `patchAll`** (the
attribute-form history bug fix). Dropped the JPEG-level orchestration
(`ParseXMPFromJPEG`/`CleanXMPInJPEG`) — that is consumer policy and would import
`jpeg`; `xmp` takes a payload starting with the xap signature (= `jpeg.Segment.Data`
for an XMP segment), like `exif.Parse`. Ported all tests incl. the mandatory
`TestCleanAttributeHistoryEmptiesAgents` regression (length-preserved, no
`Adobe Photoshop` substring, reparses to `Any()==false`). `xmp/` at 86.4%
coverage; `build`/`vet`/`gofmt`/`test ./...` green; main module still no `go.sum`.
All three packages now green → Phase 4 (tag v0.1.0) gate is met.
