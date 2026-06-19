# exifscalpel

**exifscalpel** is a small Go library of **JPEG metadata write/edit primitives** —
byte-level parsing and *surgical, minimal-diff* editing of a JPEG's segments, its
EXIF/TIFF block, and its XMP packet.

It exists to de-duplicate code currently copy-pasted between two CLIs and give them
a shared, better-tested core:

- **[tidy-exif](../tidy-exif/)** — surgically empties *Adobe* software signatures
  (XMP CreatorTool, history softwareAgent, EXIF Software), preserving everything else.
- **[lapis](../lapis/)** — privacy tool; strips identifying metadata (GPS, device
  IDs, thumbnails, IPTC) at three paranoia levels.

Both CLIs import exifscalpel as equal peers; the dependency flows one way (the
library depends on neither tool). **Policy stays in the CLIs** — exifscalpel ships
primitives only: no orchestration, flags, printing, or file walking.

## Why this exists (and why not a library off the shelf)

The Go ecosystem's metadata libraries are **read-only** (`bep/imagemeta` states
writing "is not supported, and never will"). The write-capable options either
rebuild rather than minimally edit (`dsoprea/go-exif`) or reserialize the whole XMP
packet via a model round-trip (`trimmer-io/go-xmp`) — the opposite of a
length-preserving, leave-every-other-byte-identical edit. That gap is exifscalpel's
reason to exist. Full survey in `exifscalpel-HANDOFF.md` §9.

## Planned packages

| Package | Role | Source of the engine |
|---|---|---|
| `jpeg` | segment parse/write + identification | lapis (canonical; identical in both repos) |
| `exif` | TIFF/IFD parse + rebuild, tag edit | lapis `parseEXIF`/`buildEXIF` |
| `xmp`  | XMP field-level surgery, length-preserving | tidy-exif `parseXMP`/`cleanXMP`/`marshalXMP` |

Primitives only; the high-level orchestration (`InspectJPEG`/`CleanJPEG`, lapis's
`Strip`) stays in the consumers.

## Status

**Pre-code / planning.** Repo is scaffolded; no Go yet. The build is fully
specified in **[`exifscalpel-HANDOFF.md`](exifscalpel-HANDOFF.md)** — a cold-start
guide (provenance, hard-won conclusions, package layout, API sketch, 7-phase
migration, tests, decisions, prior-art). See [`STATUS.md`](STATUS.md).

One decision to make before Phase 2: the `exif` engine — lift lapis's lean,
zero-dependency engine (recommended) vs depend on `dsoprea/go-exif` (handoff §7/§9).

## License

Mozilla Public License 2.0 (MPL-2.0) — same as [tidy-exif](../tidy-exif/).
[lapis](../lapis/) is MIT; MPL-2.0 is file-level copyleft and imports cleanly into
MIT projects (only modifications to exifscalpel's own files stay MPL).
