# exifscalpel — extraction handoff

> **Build plan.** This is the cold-start guide for building this repo; it assumes
> no prior context. The repo is scaffolded (LICENSE, README, STATUS, CHANGELOG,
> `.gitignore`) but has **no Go code yet** — start at §4 Phase 0. Source repos
> referenced below are siblings under `codeberg.org/elkarrde/` (`tidy-exif`,
> `lapis`).

## 0. What this is and why

`exifscalpel` is a small Go library of **JPEG metadata primitives** — byte-level
parsing and editing of a JPEG's segments, its EXIF/TIFF block, and its XMP
packet. It exists to de-duplicate code that is **already copy-pasted** between
two CLIs and to give each a shared, better-tested core:

- **tidy-exif** — surgically empties *Adobe* software signatures (XMP CreatorTool,
  history softwareAgent, EXIF Software), preserving everything else.
- **lapis** — privacy tool; strips identifying metadata (GPS, device IDs,
  thumbnails, IPTC) at three "paranoia" levels, willing to drop whole segments.

**Dependency direction is one-way:** both CLIs import `exifscalpel`; `exifscalpel`
imports neither. Policy (Adobe-only gate, paranoia levels, config, output) stays
in each CLI. `exifscalpel` ships *primitives only* — no orchestration, no flags, no
printing, no file walking.

### Provenance (where each layer's best implementation lives today)

| Layer | Canonical source to lift | Notes |
|---|---|---|
| JPEG segment parse/write | **lapis** `internal/strip/strip.go` | byte-identical to tidy-exif `internal/meta/jpeg.go`; pick either, lapis is canonical |
| EXIF / TIFF parse + rebuild | **lapis** `internal/strip/exif.go` | full IFD engine (ifd0/exif/gps), `parseEXIF`/`buildEXIF`; far ahead of tidy-exif |
| XMP field-level surgery | **tidy-exif** `internal/meta/xmp.go` | `parseXMP`/`cleanXMP`/`marshalXMP`; lapis has none (it deletes the whole XMP segment) |

tidy-exif's `internal/meta/exif.go` (in-place Software-tag overwrite) is **superseded** by
lapis's engine — do not lift it; re-express it on top of lapis's `exif` package
(blank tag `0x0131`, rebuild).

---

## 1. Hard-won conclusions to preserve (do not re-learn these)

These cost real debugging time. Bake them into code + tests.

1. **Adobe writes `xmpMM:History` `stEvt:softwareAgent` in ATTRIBUTE form**, not
   element form: `<rdf:li stEvt:softwareAgent="Adobe Photoshop CS6 (Windows)"/>`.
   The original tidy-exif parser only handled element form
   (`<stEvt:softwareAgent>…</…>`) and **silently missed every real Lightroom/
   Photoshop file**. The `xmp` package MUST handle both forms on parse and on
   clean. A regression fixture is mandatory (see §5).

2. **Length-preserving edits avoid rewriting JPEG offsets.** Two techniques:
   - XMP: pad the cleaned XML with whitespace inside the `<?xpacket?>` region so
     the APP1 segment keeps its original byte length (tidy-exif `adjustPadding`).
   - EXIF Software (tidy-exif's old approach): overwrite the value bytes in place,
     NUL-padded.
   lapis's `buildEXIF` instead **rebuilds** the EXIF payload (length changes) —
   which is correct when you're removing tags/IFDs anyway. Expose both modes;
   don't force one philosophy.

3. **EXIF value storage rule:** a TIFF entry value ≤ 4 bytes is stored *inline*
   in the entry's value/offset field; larger values live at an *offset relative
   to the TIFF header* (the 6th byte, after `Exif\0\0`). lapis's `parseEXIF`
   already resolves every `ifdEntry.value` to actual bytes — keep that model.

4. **The "Adobe-only" gate is POLICY, not primitive.** Deciding *whether* a
   Software value is an Adobe signature (so VueScan/camera firmware is preserved)
   belongs in tidy-exif, not `exifscalpel`. `exifscalpel` just reads/writes the tag.

5. **JPEG structure:** parsing stops at SOS (`0xFF 0xDA`) and returns the
   compressed tail (image data + EOI) verbatim. Standalone markers (SOI `D8`,
   EOI `D9`, RST `D0–D7`) carry no length. Legal `0xFF` padding before a marker
   must be skipped. (Both repos' `parseJPEG` already do this correctly.)

---

## 2. Proposed module + package layout

Module path: `codeberg.org/elkarrde/exifscalpel` · Go **1.22** · **MPL-2.0**
(same as tidy-exif; lapis is MIT — MPL-2.0 is file-level copyleft and imports
cleanly into MIT projects).

```
exifscalpel/
├── go.mod                 # module codeberg.org/elkarrde/exifscalpel; go 1.22
├── LICENSE                # MPL-2.0
├── README.md
├── doc.go                 # package overview, the §1 invariants in prose
├── jpeg/                  # LAYER 1 — segment parse/write + identification
│   ├── jpeg.go
│   └── jpeg_test.go
├── exif/                  # LAYER 2 — TIFF/IFD parse + rebuild (from lapis)
│   ├── exif.go
│   └── exif_test.go
└── xmp/                   # LAYER 3 — XMP field surgery (from tidy-exif)
    ├── xmp.go
    └── xmp_test.go
```

Design rules:
- `jpeg/` depends on nothing but stdlib.
- `exif/` and `xmp/` take a **segment payload** (`[]byte`) and return bytes; they
  do **not** import `jpeg/` (keeps them independently usable and testable).
- Segment *identification* (the `Exif\0\0` and `http://ns.adobe.com/xap/1.0/\0`
  signatures + predicates) lives in `jpeg/`.
- **No facade/orchestration package.** `InspectJPEG`/`CleanJPEG` (tidy-exif) and
  `Strip` (lapis) stay in the consumers — they encode policy.

---

## 3. Public API sketch (conceptual — names final-ish, signatures illustrative)

### `package jpeg`
```go
type Segment struct { Marker byte; Data []byte }

func Parse(r io.Reader) (segs []Segment, tail []byte, err error)
func Write(w io.Writer, segs []Segment, tail []byte) error

func IsEXIF(s Segment) bool   // marker 0xE1 + "Exif\0\0" prefix
func IsXMP(s Segment) bool    // marker 0xE1 + Adobe xap namespace prefix
```
Lift from lapis `internal/strip/strip.go`: `jpegSeg`→`Segment`, `parseJPEG`→`Parse`,
`writeJPEG`→`Write`, `isExifSeg`/`isXMPSeg`→`IsEXIF`/`IsXMP`, plus the sig consts.

### `package exif` (engine from lapis `internal/strip/exif.go`)
```go
const SoftwareTag = 0x0131            // IFD0 "Software"
// (also export GPS IFD pointer + the tags lapis already filters)

type Entry struct { Tag, Type uint16; Count uint32; Value []byte } // value resolved
type Data struct {
    ByteOrder binary.ByteOrder
    IFD0, ExifSub, GPSSub []Entry
}

func Parse(payload []byte) (*Data, error)   // was parseEXIF
func (d *Data) Build() ([]byte, error)       // was buildEXIF (rebuilds; length may change)

// small editing helpers both consumers need:
func (d *Data) Find(ifd IFDID, tag uint16) (*Entry, bool)
func (d *Data) Set(ifd IFDID, tag uint16, value []byte)
func (d *Data) Remove(ifd IFDID, tag uint16)
func (d *Data) RemoveIFD(ifd IFDID)          // e.g. drop GPS wholesale (lapis Scout)
```
tidy-exif's Software scrub becomes: `Parse → Set(IFD0, SoftwareTag, replBytes) →
Build` (or a length-preserving in-place variant if offsets must not move — see §1.2).

### `package xmp` (engine from tidy-exif `xmp.go`)
```go
type Fields struct {
    CreatorTool, MetadataDate, DocumentID, InstanceID, OriginalDocumentID string
    SoftwareAgents []string   // one per history entry; ATTRIBUTE or element form
}
func (f *Fields) Any() bool   // was HasAdobeData

func Parse(payload []byte) (*Fields, error)   // was parseXMP; handles both history forms

// Clean empties/replaces target fields IN PLACE, length-preserving via xpacket
// padding. replacements maps field name -> value ("" = empty). Returns whether
// anything changed.
func Clean(payload []byte, replacements map[string]string) (out []byte, changed bool, err error)
```
Lift `parseXMP`, `cleanXMP`, `marshalXMP`, `patchField`, `patchAll`,
`adjustPadding`, `matchAttr`, `nextCharData`, and the `ns*` consts. Keep
`patchAll` (handles attribute **and** element form) — that is the bug fix.

> Optional future: an `xmp.RemoveSegment` helper or note that lapis's
> "delete whole XMP segment" is just dropping the segment at the `jpeg` layer.

---

## 4. Migration order (each phase ends buildable + green)

**Phase 0 — repo bootstrap.** Create `exifscalpel` repo; `go mod init
codeberg.org/elkarrde/exifscalpel`; `go 1.22`; add MPL-2.0 LICENSE, README skeleton,
`doc.go`. Optional CI running `go vet ./... && go test ./...`.

**Phase 1 — `jpeg/`.** Lift the segment layer from lapis (canonical). Export the
API in §3. Port segment tests from *both* repos (lapis `strip_test.go` segment
cases + tidy-exif `jpeg`/fixture builders). Green.

**Phase 2 — `exif/`.** Lift lapis `parseEXIF`/`buildEXIF`/`ifdEntry`/`parsedEXIF`
+ `typeSizes`. Export per §3 and add `Find/Set/Remove/RemoveIFD`. Port lapis EXIF
tests (GPS removal, journalist filtering). Add a test for the tidy-exif use case
(blank `SoftwareTag`, rebuild, value gone). Green.

**Phase 3 — `xmp/`.** Lift tidy-exif `xmp.go`. Export `Parse`/`Clean`/`Fields`.
Port tidy-exif XMP tests **including the attribute-form history regression**
(§5). Green.

**Phase 4 — tag `exifscalpel` v0.1.0.** First consumable release. Gate: all three
packages green + `conformance/` green (differential EXIF vs. dsoprea, §10) +
`gofmt`/`vet` clean + main module still has **no `go.sum`** (zero runtime deps).

**Phase 5 — migrate tidy-exif (independent of Phase 6).**
- Bump tidy-exif `go.mod` to `go 1.22`; `require codeberg.org/elkarrde/exifscalpel`.
- Delete tidy-exif `jpeg.go`, `xmp.go`, `exif.go`; replace internals of
  `inspect.go` (`InspectJPEG`/`CleanJPEG`) with `jpeg`/`exif`/`xmp` calls.
- Keep tidy-exif policy in the CLI: the Adobe-only gate (`isAdobeSoftware`),
  config→replacements map, check/clean output, dry-run.
- Verify: existing unit suite + the real-file smoke test (tidy-exif keeps a
  gitignored `testdata/` of real Lightroom/Photoshop exports). All Adobe
  signatures gone; non-Adobe (VueScan) preserved; output decodes; sizes stable.

**Phase 6 — migrate lapis (independent of Phase 5).**
- Replace `internal/strip` segment + EXIF code with `exifscalpel` imports.
- Keep `Level`/`Strip`/`processSegments` policy and `internal/timestamp`,
  `internal/rename` as-is (not in scope for exifscalpel).
- Verify lapis's existing `strip_test.go` (Scout GPS removal, Ghost full excise,
  Journalist thumbnail removal, non-JPEG error, clean pass-through).

**Phase 7 — (optional, future) new lapis capability.** Add a less-destructive
option that keeps the XMP block but scrubs identifying fields, powered by
`xmp.Clean` — something lapis cannot do today. This is the "extensibility"
payoff, not required for parity.

> Phases 5 and 6 are decoupled: either tool can migrate first; the other keeps
> its vendored copy until it's ready.

---

## 5. Mandatory regression fixtures / tests

- **XMP attribute-form history** (the bug). Minimal XMP with:
  `<rdf:li stEvt:action="saved" stEvt:softwareAgent="Adobe Photoshop CS6 (Windows)" .../>`
  Assert `Parse` populates `SoftwareAgents` and `Clean` empties it, output length
  unchanged, no `Adobe Photoshop` substring remains. (Copy from tidy-exif
  `xmp_test.go` `attrHistoryXMP` / `TestParseXMPAttributeHistory` /
  `TestCleanAttributeHistoryEmptiesAgents`.)
- **EXIF Software round-trip** in both byte orders (II/MM): build a minimal Exif
  APP1 with a Software tag, blank it, confirm gone. (Copy tidy-exif
  `exif_test.go` `buildExifSeg`.)
- **EXIF length-preserving in-place** vs **rebuild** — test whichever modes you
  expose.
- **lapis EXIF behaviors** — port as-is (GPS IFD removal, etc.).
- **Byte-fixtures only**, no real photos in the repo (real JPEGs stay gitignored
  in tidy-exif's `testdata/`).

---

## 6. Decisions already made

- **Name: `exifscalpel`** (module `codeberg.org/elkarrde/exifscalpel`). Chosen
  over the generic `jpegmeta`, which also collides with an existing read-only
  package (`kovidgoyal/imaging/.../jpegmeta`). "scalpel" signals the surgical,
  minimal-diff intent. Sub-packages stay `jpeg` / `exif` / `xmp`.
- Module: `codeberg.org/elkarrde/exifscalpel`, Go 1.22, **MPL-2.0** (same as tidy-exif; lapis is MIT).
- Three packages `jpeg` / `exif` / `xmp`; primitives only; no orchestration.
- EXIF engine sourced from **lapis**; XMP from **tidy-exif**; segment layer from
  **lapis** (canonical copy).
- Consumers retain all policy; dependency flows one way into `exifscalpel`.
- tidy-exif's go.mod bumps 1.16 → 1.22 on migration (also unblocks
  BurntSushi/toml v1.x there, though not required).

## 7. Open decisions (resolve during build)

- **EXIF engine: hand-roll vs dependency.** ✅ **DECIDED 2026-06-21 — lift lapis's
  lean, zero-dep engine.** Matches both tools' no-deps ethos and byte-level
  control. `dsoprea/go-exif/v3` remains the escape hatch if coverage outgrows it.
- **Single package vs sub-packages.** ✅ Sub-packages (`jpeg`/`exif`/`xmp`), per
  the §2 layout — confirmed as Phase 1 landed.
- **EXIF edit API shape:** ✅ **DECIDED — mutate `*Data`** via
  `Find/Set/Remove/RemoveIFD` (as sketched).
- **EXIF in-place-vs-rebuild:** ✅ **DECIDED — expose BOTH.** Rebuild =
  `(*Data).Build` (length may change; lapis's path). In-place length-preserving =
  `OverwriteValueInPlace` (re-expressed from tidy-exif's `cleanExifSoftware`,
  generalized to any IFD0 tag; NUL-padded, no offsets move).
- **IPTC (APP13):** lapis has a TODO to strip IPTC location fields. Out of scope
  for v1; reserve a future `iptc/` package.
- **Error/`io` conventions:** `[]byte` in/out (current) vs `io.Reader`/`Writer`
  at the `jpeg` layer only (lapis uses readers there).

## 8. Quick reference — exact symbols to lift

**From lapis** (`internal/strip/`):
`strip.go`: `jpegSeg`, `parseJPEG`, `writeJPEG`, `isExifSeg`, `isXMPSeg`,
`exifSig`, `xmpSig`. `exif.go`: `typeSizes`, `typeSize`, `ifdEntry`,
`parsedEXIF`, `parseEXIF`, `buildEXIF`. *(Leave behind: `Level`, `Strip`,
`processSegments`, `process*EXIF` — those are lapis policy.)*

**From tidy-exif** (engine now lives in `internal/meta/`): `meta/xmp.go`:
`XMPData`, `HasAdobeData`, `parseXMP`, `matchAttr`, `nextCharData`, `cleanXMP`,
`marshalXMP`, `patchField`, `patchAll`, `adjustPadding`,
`nsXMP/nsXMPMM/nsStEvt/nsRDF`. *(Leave behind: `meta/inspect.go`
`InspectJPEG`/`CleanJPEG`, `meta/exif.go` entirely, `isAdobeSoftware` — tidy-exif
policy. Do NOT lift tidy-exif's `meta/exif.go`; use lapis's engine.)*

---

## 9. Prior art — evaluated before building (2026-06)

Surveyed existing Go libraries to confirm exifscalpel is worth building rather
than reusing. **Conclusion: the write/edit primitives have no off-the-shelf
substitute that fits a surgical, minimal-diff philosophy.**

### Read-only (cannot serve the clean/strip path at all)
- **`github.com/bep/imagemeta`** (MIT) — broad, mature reader: EXIF/IPTC/XMP/ICC
  across JPEG/TIFF/PNG/WebP/HEIF/AVIF/raw, validated against exiftool. States
  outright: *"Writing is not supported, and never will."* Returns decoded values,
  not byte ranges, so it can't drive edits. Possible *future* use: richer
  `check`/report output only.
- **`github.com/kovidgoyal/imaging/.../jpegmeta`** (MIT) — read-only; image
  dimensions, colour space, ICC profiles. Not our domain. Its package name is the
  reason we renamed to **exifscalpel**.

### Write-capable (the real alternatives)
- **`github.com/dsoprea/go-exif/v3`** (MIT) + **`github.com/dsoprea/go-jpeg-
  image-structure/v2`** (MIT) — genuinely read/modify/**write** EXIF and re-emit
  JPEGs via an `IfdBuilder` API. A real alternative to hand-rolling the `exif`
  package. **Tradeoffs:** rebuilds EXIF (not byte-exact in place), heavier
  transitive deps, more complex API than lapis's ~360-line engine. **XMP:
  go-jpeg-image-structure reads but does not write XMP** — so it can't cover our
  XMP need regardless.
- **`github.com/trimmer-io/go-xmp`** (Apache-2.0) — read/write XMP via a
  *model-based* Unmarshal→modify→Marshal round-trip with 20+ namespace models.
  **Poor fit:** reserializing risks dropping unmodeled namespaces, changing byte
  length, and reformatting — the opposite of exifscalpel's `xmp` package, which
  empties one field and leaves every other byte identical. Also Apache-2.0 vs our
  MPL-2.0.

### Decisions this drives
- **`xmp` stays hand-rolled.** A model round-trip is the wrong tool for "empty
  one field, preserve everything else byte-for-byte." Our regex-patch-in-place
  approach (attribute + element history forms, xpacket padding) is the better fit
  and is dependency-free. This is exifscalpel's most defensible unique value.
- **`exif` is a real fork (see §7).** Default: lift lapis's lean zero-dep engine
  (matches both tools' deliberate no-deps ethos, byte-level control). Escape
  hatch: depend on dsoprea if EXIF coverage/edge-cases outgrow it.
- **`jpeg` segment layer stays hand-rolled** (~120 lines, zero deps) rather than
  pull in go-jpeg-image-structure's heavier abstraction.

---

## 10. Dependency policy & conformance testing (added 2026-06-21)

**"No deps" means zero baggage in the *shipped artifacts* (Lapis/TidyExif), not
in development.** Two tiers — full text in `CONTRIBUTING.md`:

- **Runtime (library: `jpeg`/`exif`/`xmp`/root): stdlib only, forever.** No
  non-test file in the main module imports a third-party package. Invariant to
  watch: the **main module has no `go.sum`**. If one appears at the repo root,
  something leaked into runtime code.
- **Dev/test: unrestricted.** Test-only imports never reach consumers (not
  compiled downstream; go 1.17+ graph pruning keeps them out of consumers'
  graphs). Go has no `devDependencies` field, so **heavy dev tooling goes in a
  separate module** to keep the main `go.mod` clean.

This *re-confirms* the §7 engine decision rather than reopening it: we keep the
lean hand-rolled engine at **runtime** and use a mature reference (dsoprea) only
as a **test oracle**. Best of both — no "depend on dsoprea" escape hatch needed.

**`conformance/`** — a separate Go module (`replace … => ../`) holding
differential EXIF tests vs. `github.com/dsoprea/go-exif/v3`:
- `TestReference_ReadsOurOutput` — a third-party reader parses our `Build`
  output (Make/Software/ISO/GPS) → we emit standards-compliant bytes.
- `TestReference_ConfirmsScrubAndGPSRemoval` — after `RemoveIFD(GPS)` + Software
  scrub, the reference sees no GPS and no Adobe → our edits are genuinely clean.

Run: `go -C conformance test ./...`. Future: XMP is intentionally **not** here
(model-based XMP libs are poor oracles for byte-surgery; prefer `exiftool` as an
external binary); add stdlib fuzzing + property tests for `Parse∘Build`.
