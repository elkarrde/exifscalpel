# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-07-07

Read-only XMP property extraction, additive and backward compatible: existing
`jpeg`, `exif`, `iptc`, and `xmp` Parse/Clean behavior is unchanged, so consumers
on 0.2.0 are unaffected. Still stdlib-only at runtime. Motivated by film-scan
metadata (ExifNotes/AnalogExif), which writes a structured, cleaner copy of its
fields to XMP than to the EXIF `UserComment` text block — including the lens and
scanner software, which the EXIF block omits entirely.

### Added

- **`xmp.ReadProperties`** — extract arbitrary simple (scalar) XMP properties from
  an APP1 payload, matched by namespace URI + local name (prefix-independent).
  Recognises both attribute form (`ns:Name="value"`, how AnalogExif/aux write
  scalars on `rdf:Description`) and simple element form (`<ns:Name>value</ns:Name>`).
  Read-only, no vocabulary of its own — the caller names the namespaces/fields it
  wants (e.g. the AnalogExif film schema, `aux:Lens`); policy stays in the consumer.
  New `xmp.Property{Namespace, Name}` type keys the request and the result map.

## [0.2.0] - 2026-07-02

Location-stripping primitives for the `no-gps` use case: a new `iptc` package and
an opt-in `xmp.CleanLocation`. Additive and backward compatible — existing `jpeg`,
`exif`, and `xmp.Clean` behavior is unchanged, so consumers on 0.1.0 are unaffected.
Still stdlib-only at runtime (no `go.sum`).

### Added

- **`iptc`** — surgical edits on the IPTC-IIM metadata in a JPEG APP13 (Photoshop
  Image Resource) segment (`Parse`, `Dataset`, `(*Data).Datasets`, `Remove`,
  `Empty`, `Build`). Walks the `8BIM` resource blocks, decodes the IIM datasets of
  the `0x0404` block, removes datasets by `record:number` (repeatables included),
  and rebuilds — recomputing the block size and even-padding, dropping the `0x0404`
  block when it empties. Every other resource block (thumbnail, ICC, embedded EXIF)
  is preserved verbatim. *Which* datasets count as "location" is policy and stays in
  the consumer. Big-endian throughout; stdlib only.
- **`xmp.CleanLocation`** — opt-in, length-preserving removal of location and GPS
  fields (`exif:GPS*`, `photoshop:City/State/Country`, `Iptc4xmpCore:*`, and the
  structured `Iptc4xmpExt:LocationCreated`/`LocationShown` leaf fields). Blanks each
  field's value at every nesting depth via `patchAll`, so struct-nested location
  data is emptied without container-aware parsing; the empty containers remain and
  the payload keeps its byte length. Kept separate from `Clean` (Adobe-signature
  fields, consumed by tidy-exif) so that default behavior is untouched.

## [0.1.0] - 2026-06-24

First consumable release: three independent, primitives-only packages, stdlib-only
at runtime (no `go.sum`). Differential EXIF conformance suite lives in its own
`conformance/` module.

### Added

- **`jpeg`** — byte-level JPEG segment parse/write (`Segment`, `Parse`, `Write`)
  and segment identification (`IsEXIF`, `IsXMP`). Parsing stops at SOS and returns
  the compressed tail verbatim; skips legal `0xFF` padding before a marker. Stdlib
  only.
- **`exif`** — TIFF/IFD parse and edit over a `*Data` model (`Find`, `Set`,
  `Remove`, `RemoveIFD`). Two edit modes, both exposed: `(*Data).Build` rebuilds the
  payload (length may change; reconciles sub-IFD pointers) and
  `OverwriteValueInPlace`/`ReadValue` edit length-preservingly. Handles both byte
  orders (II/MM) and inline vs. offset value storage. Exports `SoftwareTag`,
  `ExifIFDPointer`, `GPSIFDPointer`.
- **`xmp`** — field-level, length-preserving XMP surgery (`Parse`, `Clean`,
  `Fields`). Pads cleaned XML inside `<?xpacket?>` so the APP1 segment keeps its byte
  length. Handles `xmpMM:History` `stEvt:softwareAgent` in both element and
  attribute form (`patchAll`).
- **`conformance/`** — separate module; differential EXIF tests validating the
  hand-rolled engine against `dsoprea/go-exif/v3` as a read oracle.

[0.2.0]: https://codeberg.org/elkarrde/exifscalpel/releases/tag/v0.2.0
[0.1.0]: https://codeberg.org/elkarrde/exifscalpel/releases/tag/v0.1.0
