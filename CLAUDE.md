# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Current state

**Pre-code / planning.** The repo is scaffolded (LICENSE, README, STATUS, CHANGELOG,
`.gitignore`) but contains **no Go code yet** — no `go.mod`. The complete, cold-start
build plan is **`exifscalpel-HANDOFF.md`** — treat it as the source of truth for
provenance, package layout, API signatures, the 7-phase migration, and required
tests. `STATUS.md` tracks current phase. When the plan and this file disagree, the
handoff wins; update `STATUS.md` as phases land.

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

## Test conventions

- **Byte-fixtures only — no real photos in the repo.** Real JPEGs stay gitignored in
  tidy-exif's `testdata/`. Build minimal Exif/XMP segments programmatically.
- Mandatory fixtures (handoff §5): XMP attribute-form history regression; EXIF
  Software round-trip in both byte orders (II/MM); EXIF in-place vs rebuild for
  whichever modes are exposed; ported lapis EXIF behaviors (GPS IFD removal).

## Commands (once Go code exists)

```bash
go mod init codeberg.org/elkarrde/exifscalpel   # Phase 0, go 1.22
go build ./...
go vet ./... && go test ./...
go test ./xmp/ -run TestCleanAttributeHistoryEmptiesAgents   # single test
go test -cover ./...
```

## Open decision before Phase 2

The `exif` engine: **lift lapis's lean zero-dependency engine (recommended**, matches
both tools' no-deps ethos) vs depend on `dsoprea/go-exif/v3`. Record the choice in
the handoff/STATUS before writing `exif/`. See handoff §7 and §9.
