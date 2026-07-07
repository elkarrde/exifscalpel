# exifscalpel

**exifscalpel** is a small, dependency-free Go library of **JPEG metadata
write/edit primitives** — byte-level parsing and *surgical, minimal-diff* editing
of a JPEG's segments, its EXIF/TIFF block, its XMP packet, and its IPTC-IIM datasets.

The design is **primitives only**: no orchestration, flags, printing, or file
walking. Each of the metadata packages takes bytes and returns bytes, so you own
the policy — the library just does the byte surgery, and keeps every other byte
identical.

## Why this exists (and why not a library off the shelf)

The Go ecosystem's metadata libraries are **read-only** (`bep/imagemeta` states
writing "is not supported, and never will"). The write-capable options either
rebuild rather than minimally edit (`dsoprea/go-exif`) or reserialize the whole XMP
packet via a model round-trip (`trimmer-io/go-xmp`) — the opposite of a
length-preserving, leave-every-other-byte-identical edit. That gap is exifscalpel's
reason to exist.

## Packages

| Package | Role |
|---|---|
| `jpeg` | byte-level segment parse/write + identification (`IsEXIF`/`IsXMP`) |
| `exif` | TIFF/IFD parse + edit; rebuild (`(*Data).Build`) or length-preserving in-place (`OverwriteValueInPlace`) |
| `xmp`  | field-level XMP surgery, length-preserving (`Clean`, `CleanLocation`, `ReadProperties`) |
| `iptc` | IPTC-IIM dataset edits inside an APP13 / Photoshop 8BIM segment |

Only `jpeg` walks the JPEG container; `exif`, `xmp`, and `iptc` each take a raw
segment payload (`[]byte`) and return bytes, so every package is independently
usable and testable. All four are stdlib-only.

## Install

```bash
go get codeberg.org/elkarrde/exifscalpel@latest
```

Requires Go 1.22+.

## Dependencies

**Zero runtime dependencies, by design** — the library imports only the Go standard
library, so the main module has no `go.sum` and anything that imports it inherits
nothing transitive. Dev/test tooling (a differential EXIF suite cross-checked
against `dsoprea/go-exif/v3`) lives in the separate `conformance/` module and never
reaches importers. See [`CONTRIBUTING.md`](CONTRIBUTING.md).

## Status

Published and in active use. See [`CHANGELOG.md`](CHANGELOG.md) for the current
release and [`STATUS.md`](STATUS.md) for project state.

## License

Mozilla Public License 2.0 (MPL-2.0) — file-level copyleft: it imports cleanly into
projects under other licenses (including MIT), and only modifications to
exifscalpel's own files stay under MPL.
