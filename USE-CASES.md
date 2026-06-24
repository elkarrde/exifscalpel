# Use cases

Forward-looking notes on where `exifscalpel` is a good fit beyond the two CLIs it
was carved out of (`lapis`, `tidy-exif`). Not a roadmap — `STATUS.md` and
`exifscalpel-HANDOFF.md` own the plan. This is a catalogue of *why* the library's
shape suits certain problems, and an honest split between what the current
`jpeg`/`exif`/`xmp` primitives reach today and what would need new code.

## What makes the library a fit

The properties that recur below, and that most of the Go ecosystem does **not**
offer (the popular metadata libs are read-only or full-rewrite):

- **Stdlib-only, zero runtime deps.** Compiles anywhere Go does — including
  `GOOS=js GOARCH=wasm` — with no transitive baggage and a small artifact.
- **Byte-level, minimal-diff edits.** Two modes, neither forced:
  `exif.OverwriteValueInPlace` / `xmp.Clean` preserve segment length (every other
  byte, and every JPEG/TIFF offset, stays put); `exif.Data.Build` rebuilds when a
  field count actually has to change.
- **Primitives, not policy.** `jpeg.Parse`/`Write` deal in segment structure;
  `exif`/`xmp` take a payload and return bytes. Orchestration (what to strip, what
  to keep, what an "Adobe" signature is) lives in the caller.

> **Caveat that recurs:** `exif.Data.Build` (rebuild) intentionally drops IFD1
> (the embedded thumbnail) — its entries point at raw bytes that can't be safely
> relocated. The in-place path (`OverwriteValueInPlace`) preserves everything,
> including the thumbnail. Pick the mode per use case; it matters for sanitizers
> (dropping the thumbnail is usually *desirable*) and for diffing (a rebuild won't
> round-trip IFD1).

---

## 1. Client-side stripping in the browser (WASM)

**What:** strip EXIF/GPS from a photo **in the browser, before it is ever
uploaded** — the strongest form of the privacy guarantee, since identifying
metadata never leaves the device.

**Why exifscalpel:** stdlib-only Go compiles cleanly to `GOOS=js GOARCH=wasm` with
no dependency friction and a comparatively small `.wasm` payload. The API already
speaks `[]byte` / `io.Reader`, which bridges trivially to a JS `Uint8Array` /
`File`. Read-only libraries can't do this at all; full-rewrite libraries produce a
larger, lossier result.

**Reachable today.** A thin `syscall/js` shim over the existing primitives:

```go
//go:build js && wasm
func strip(this js.Value, args []js.Value) any {
    in := make([]byte, args[0].Get("length").Int())
    js.CopyBytesToGo(in, args[0])

    segs, tail, err := jpeg.Parse(bytes.NewReader(in))
    // … drop/clean segs per policy (excise EXIF+XMP, or xmp.Clean fields) …
    var out bytes.Buffer
    jpeg.Write(&out, segs, tail)
    // … return a Uint8Array to JS …
}
```

**Needs building:** the `js` shim and a tiny demo page; nothing in the core. Good
first showcase — near-zero new library code, high differentiation.

---

## 2. Server-side upload sanitizer / HTTP middleware

**What:** "scrub on ingest" for photo-sharing backends, forums, CMSes — the same
job as lapis, but as a library call inside a request handler instead of a CLI.

**Why exifscalpel:** zero deps and minimal allocation suit a hot path; the
`io.Reader` → `io.Writer` shape at the `jpeg` layer drops straight into a handler.
A consumer keeps the *policy* (which level, which fields) exactly as lapis does —
the library stays a primitive.

**Reachable today** for JPEG. Sketch:

```go
func sanitize(w http.ResponseWriter, r *http.Request) {
    segs, tail, err := jpeg.Parse(r.Body)
    if err != nil { /* reject non-JPEG */ }
    kept := segs[:0]
    for _, s := range segs {
        if jpeg.IsEXIF(s) || jpeg.IsXMP(s) { continue } // or xmp.Clean / exif edits
        kept = append(kept, s)
    }
    jpeg.Write(dst, kept, tail) // dst = temp file / object store / response
}
```

**Caveats / needs building:** stream-size limits, content-type sniffing, and
**non-JPEG inputs**. The library is JPEG-only today — a real ingest path also sees
PNG/HEIC/WebP, which need new segment layers (see §6 caveat in the roadmap).
Decide policy: reject non-JPEG, or pass through untouched.

---

## 3. IPTC injection / editing (copyright & licensing) — *needs a new package*

**What:** stamp or edit IPTC-IIM fields (creator, copyright notice, credit, usage
terms, keywords) carried in the **APP13 "Photoshop" segment**. The bread-and-butter
of DAM systems, stock pipelines, and newsroom workflows.

**Why exifscalpel:** the architecture already *reserves* this — handoff §7 notes a
future `iptc/` package, and lapis carries a TODO to strip IPTC location fields.
APP13 is already visible to `jpeg.Parse` as a generic `Segment` (marker `0xED`);
what's missing is a parser/editor for the 8BIM → IPTC record/dataset structure
inside it.

**Reachable today:** *locating* and *excising* the APP13 segment (lapis already does
this at its `clean` level). **Reading or editing individual IPTC fields is not** —
that's the new package.

**Needs building — Phase 8 candidate:** an `iptc/` package mirroring the others'
contract:

```go
package iptc
func Parse(payload []byte) (*Data, error)          // payload = jpeg.Segment.Data for APP13
func (d *Data) Set(record, dataset uint8, value []byte)
func (d *Data) Build() ([]byte, error)
// + length-preserving overwrite for fixed-width fields, mirroring exif
```

Same primitives-only discipline: the "which fields are PII vs. rights metadata"
judgement stays in the consumer.

---

## 4. Provenance / authenticity (C2PA-adjacent) — *partial; heaviest lift*

**What:** read, preserve, or attach content-provenance manifests (C2PA / "Content
Credentials"), which for JPEG live in **JUMBF boxes inside APP11** segments.

**Why exifscalpel:** the **minimal-diff property is the real asset here.** Provenance
manifests hash portions of the file; a length-preserving edit elsewhere
(`exif.OverwriteValueInPlace`, `xmp.Clean`) won't shift offsets and can be designed
not to invalidate a hash, and the byte-faithful `jpeg.Write` reproduces untouched
segments exactly. Adding or removing a single APP11 without disturbing neighbours is
precisely what the segment layer is good at.

**Reachable today:** APP11 segments are already parsed as generic `Segment`s, so you
can **locate, count, preserve, or strip** a provenance manifest, and reason about
which edits keep the rest of the file byte-stable.

**Needs building — and a deliberate boundary:** parsing the manifest itself means
**JUMBF box parsing + CBOR** (not stdlib) and **COSE/X.509 signing + verification**
(crypto *is* stdlib, CBOR is not). Per the dependency policy, anything pulling a CBOR
library must **not** land in the zero-dep runtime — it belongs in a consumer or a
separate module (the `conformance/` pattern). exifscalpel's role stays narrow:
*segment-level* manifest placement and offset-stable edits, not manifest semantics.

---

## 5. Metadata diffing — *read-only, reachable today*

**What:** compare two JPEGs' metadata and report exactly what changed — "did this CDN
/ resizer / social platform mangle my EXIF?", regression-checking a processing
pipeline, or auditing before/after a clean.

**Why exifscalpel:** `exif.Parse` resolves every IFD entry to its **actual value
bytes** (not an offset), and `xmp.Parse` extracts named fields — so a structural
diff (tag added / removed / changed) is straightforward without a second library.
It also doubles as a differential oracle, which is exactly how `conformance/` already
uses the engine against `dsoprea/go-exif`.

**Reachable today** as a read-only consumer:

```go
a, _ := exif.Parse(segA.Data)
b, _ := exif.Parse(segB.Data)
// walk a.IFD0 / a.ExifSub / a.GPSSub vs b.* by Tag; compare Entry.Value bytes
xa, _ := xmp.Parse(xmpA) ; xb, _ := xmp.Parse(xmpB) // compare Fields
```

**Caveats:** the parse model deliberately **drops IFD1 (thumbnail)** and unknown
sub-IFDs, so a diff sees IFD0 / Exif / GPS, not the thumbnail — fine for "what
identifying data changed", not for "is this byte-identical" (for that, diff the raw
`Segment.Data`). A friendly textual/JSON report and a stable ordering of entries are
the only things to build.

---

## Summary

| Use case | Reachable with v0.1.0 primitives | New code needed |
|---|---|---|
| 1. Browser/WASM strip | ✅ core; ⬜ `syscall/js` shim | shim + demo only |
| 2. Upload sanitizer | ✅ for JPEG | size/type limits; non-JPEG containers |
| 3. IPTC inject/edit | ⬜ excise only | **`iptc/` package (Phase 8)** |
| 4. C2PA-adjacent | ✅ segment placement / offset-stable edits | JUMBF+CBOR+COSE — in a *consumer*, not the runtime |
| 5. Metadata diff | ✅ read-only | report formatting; raw-bytes path for IFD1 |

The two clear, low-risk expansions are **(1) the WASM showcase** (almost no new
library code, high differentiation) and **(3) the `iptc/` package** (already
designed-for, finishes lapis's TODO, unlocks the licensing/DAM cases). Provenance is
the most strategically interesting but the heaviest, and must respect the zero-dep
runtime line.
