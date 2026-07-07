# Status

*Last updated: 2026-07-08*

| Field | Value |
|:--|:--|
| Phase | **Phases 0–6 done; read-only XMP extraction added.** `xmp.ReadProperties` (film-scan/AnalogExif fields) lands on top of the 0.2.0 location primitives (`iptc` + `xmp.CleanLocation`) |
| Version | `v0.3.1` (tagged + pushed to both remotes; proxy-resolvable). **0.3.0 was skipped** — tagged locally but never pushed; `ReadProperties` ships in 0.3.1 |
| Build | `go build`/`vet`/`gofmt` clean; main module has **no `go.sum`** (zero runtime deps) |
| Tests | `go test ./...` green (`jpeg`, `exif`, `xmp`, `iptc`); `go -C conformance test ./...` green |
| Published | `v0.1.0`, `v0.2.0`, `v0.3.1` pushed to origin (Codeberg) + github. `v0.3.0` deliberately unreleased |
| Next | Optional: adopt `xmp.ReadProperties` in a consumer (e.g. tidy-exif film-scan reporting); Phase 7 (lapis XMP-scrub level); conformance extensions (XMP oracle, fuzzing) |

## ▶ Next session — start here

**Nothing pending to release.** `v0.3.1` is the current published release (Codeberg +
github, proxy-resolvable); `v0.3.0` was intentionally skipped (local-only tag, never
pushed — see CHANGELOG). All library work through read-only XMP extraction is
committed and green.

Remaining work is all optional and consumer-side:

1. **Consume `xmp.ReadProperties`** where it helps — e.g. tidy-exif reporting the
   AnalogExif film/lens/scanner fields that the EXIF `UserComment` blob omits. The
   real-photo film scans for spot-checking live outside the repo at
   `../masterdata/testdata` (`Scan-*` files carry AnalogExif + `aux` XMP).
2. **Phase 7** (lapis XMP-scrub level via `xmp.Clean`) and **conformance extensions**
   (XMP differential via an `exiftool` oracle; fuzzing) — see below.

## ▶ Prior milestone — start here (0.1.0)

**Core migration is complete (Phases 0–6).** The library is published at `v0.1.0`
and both sibling CLIs now consume it; their own engine copies are deleted. Remaining
work is all optional:

- **Phase 7 (new capability):** add a lapis level that *scrubs* identifying XMP
  fields via `xmp.Clean` (keeping the XMP block) instead of excising the whole APP1
  segment — the extensibility payoff the shared core unlocks. See handoff §4 Phase 7.
- **Conformance extensions** (`conformance/` README "Next"): XMP differential via an
  `exiftool` oracle; fuzzing the parsers.

**Phase 5 done (2026-06-24):** tidy-exif migrated. Deleted `internal/meta/{jpeg,xmp}.go`
and the EXIF engine in `exif.go` (kept only `isAdobeSoftware`, the Adobe-only gate).
`inspect.go` now drives `jpeg`/`xmp`/`exif`: `xmp.Clean` for the XMP segment,
`exif.ReadValue`/`exif.OverwriteValueInPlace(SoftwareTag)` for EXIF; the
`ParseXMPFromJPEG`/`CleanXMPInJPEG` convenience wrappers were re-homed here over the
library. `FileReport.XMP` is now `*xmp.Fields`; **the CLI needed no changes**
(`report.XMP.{CreatorTool,SoftwareAgents}` + `report.HasAdobeData()` all still
resolve). go.mod bumped 1.16→1.22. Engine-only tests dropped (covered in the lib);
the mandatory attribute-form regression kept at the CleanJPEG integration level.
build/vet/test green; only one new dep (exifscalpel, zero transitive). **Uncommitted
in `../tidy-exif/`**; `MIGRATION.md` there is now executed (delete or keep as record).

**Phase 6 done (2026-06-24):** lapis migrated. `internal/strip/{strip,exif}.go` import
`exifscalpel/{jpeg,exif}`; engine deleted, all policy kept. Green; one new dep,
zero transitive. **Committed in `../lapis/`** by the user.

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
