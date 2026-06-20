# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Current state

**Phases 0–2 done (green).** `jpeg/` and `exif/` are lifted and tested; `xmp/` is
**next (Phase 3)**. The complete, cold-start build plan is **`exifscalpel-HANDOFF.md`**
— treat it as the source of truth for provenance, package layout, API signatures, the
7-phase migration, and required tests. `STATUS.md` tracks current phase and carries a
"start here next session" pointer. When the plan and this file disagree, the handoff
wins; update `STATUS.md` as phases land.

## What this is

A small Go library of **JPEG metadata write/edit primitives** — byte-level parsing
and *surgical, minimal-diff* editing of a JPEG's segments, its EXIF/TIFF block, and
its XMP packet. It exists to de-duplicate code currently copy-pasted between two
sibling CLIs (`../tidy-exif/`, `../lapis/`) and give them a shared, tested core.

Module path: `codeberg.org/elkarrde/exifscalpel` · Go **1.22** · **MPL-2.0**.

## Architecture (three independent packages, primitives only)

| Package | Role | Lift the engine from |
|---|---|---|
| `jpeg` | segment parse/write + identification (`IsEXIF`/`IsXMP`) | lapis `internal/strip/strip.go` (canonical) |
| `exif` | TIFF/IFD parse + rebuild, tag edit | lapis `internal/strip/exif.go` |
| `xmp`  | XMP field-level surgery, length-preserving | tidy-exif `meta/xmp.go` |

Hard rules that shape the design:

- **Primitives only.** No orchestration, flags, printing, or file walking.
  `InspectJPEG`/`CleanJPEG` (tidy-exif) and `Strip` (lapis) stay in the consumers —
  they encode *policy*. The dependency flows one way: both CLIs import exifscalpel;
  exifscalpel imports neither.
- **`jpeg/` depends only on stdlib.** `exif/` and `xmp/` take a segment payload
  (`[]byte`) and return bytes; they do **not** import `jpeg/`, so each package stays
  independently usable and testable. Segment *identification* (the `Exif\0\0` and
  Adobe `xap` namespace signatures) lives in `jpeg/`.
- The **"Adobe-only" gate is policy**, not a primitive — deciding whether a Software
  value is an Adobe signature (vs VueScan/camera firmware to preserve) belongs in
  tidy-exif. exifscalpel just reads/writes the tag.

## Non-obvious invariants to preserve (handoff §1 — these cost real debugging)

1. **XMP `xmpMM:History` `stEvt:softwareAgent` appears in ATTRIBUTE form**
   (`<rdf:li stEvt:softwareAgent="Adobe Photoshop CS6 (Windows)"/>`), not only
   element form. tidy-exif's original parser handled only element form and silently
   missed every real Lightroom/Photoshop file. The `xmp` package MUST handle both on
   parse and clean — keep `patchAll`. The attribute-form regression fixture is
   **mandatory**.
2. **Length-preserving edits avoid rewriting JPEG offsets.** XMP pads cleaned XML
   with whitespace inside `<?xpacket?>` so APP1 keeps its byte length
   (`adjustPadding`). lapis's `buildEXIF` instead *rebuilds* the EXIF payload (length
   changes) — correct when removing tags/IFDs. **Expose both modes; don't force one.**
3. **EXIF value storage:** a TIFF entry value ≤ 4 bytes is stored inline; larger
   values live at an offset relative to the TIFF header (after `Exif\0\0`). lapis's
   `parseEXIF` already resolves every entry's value to actual bytes — keep that model.
4. **JPEG parsing stops at SOS (`0xFF 0xDA`)** and returns the compressed tail
   verbatim. Standalone markers (SOI/EOI/RST) carry no length; skip legal `0xFF`
   padding before a marker.

## Dependency policy ("no deps" = zero *runtime* baggage)

The motto is about the **shipped artifacts** (the Lapis/TidyExif binaries), not
development. Two tiers, see `CONTRIBUTING.md`:

- **Runtime** — the library (`jpeg/`, `exif/`, `xmp/`, root): **stdlib only, forever.**
  No non-test file in the main module may import a third-party package. This is why
  the main module has **no `go.sum`** and why both consumers inherit zero transitive
  deps. Wanting a third-party parser in engine code is a design smell — the engines
  are deliberately hand-rolled (handoff §7/§9).
- **Dev/test** — unrestricted (reference impls, fuzzing, property tests). Go keeps
  these out of consumers (test-only imports aren't compiled downstream; go 1.17+
  graph pruning keeps them out of consumers' graphs). Because Go has no
  `devDependencies` field, **heavy dev tooling lives in a separate module** so the
  main `go.mod` stays dependency-free. See `conformance/` — a differential EXIF suite
  vs. `dsoprea/go-exif/v3`, in its own module with `replace … => ../`.

## Test conventions

- **Byte-fixtures only — no real photos in the repo.** Real JPEGs stay gitignored in
  tidy-exif's `testdata/`. Build minimal Exif/XMP segments programmatically.
- Mandatory fixtures (handoff §5): XMP attribute-form history regression; EXIF
  Software round-trip in both byte orders (II/MM); EXIF in-place vs rebuild for
  whichever modes are exposed; ported lapis EXIF behaviors (GPS IFD removal).
- **Differential testing** lives in `conformance/` (separate module): validate the
  hand-rolled engines against a mature reference reader. Runtime stays dep-free.

## Commands

```bash
go build ./... && go vet ./... && go test ./...   # library (stdlib only)
go test -cover ./...
gofmt -l .                                         # must be empty
go test ./xmp/ -run TestCleanAttributeHistoryEmptiesAgents   # single test (Phase 3)
go -C conformance test ./...                       # differential EXIF suite (separate module)
```

Requires Go **1.22+** (system toolchain at `/usr/local/go`). `./...` does **not**
descend into `conformance/` — it's a separate module.

## Resolved decisions (were open; see handoff §7)

- **`exif` engine** → lift lapis's lean zero-dep engine (done, Phase 2). dsoprea is
  used only as a *test oracle* in `conformance/`, never at runtime.
- **EXIF edit API** → mutate `*Data` (`Find/Set/Remove/RemoveIFD`).
- **In-place vs rebuild** → both exposed: `(*Data).Build` (rebuild) and
  `exif.OverwriteValueInPlace` (length-preserving).
