# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[0.1.0]: https://codeberg.org/elkarrde/exifscalpel/releases/tag/v0.1.0
