# conformance

Differential tests for exifscalpel's hand-rolled **EXIF** engine, cross-checked
against a mature, independently-maintained reference reader
(`github.com/dsoprea/go-exif/v3`).

**This is a separate Go module on purpose.** The reference dependency is
dev/test-only and must never reach the exifscalpel library or its consumers.
Keeping it in its own `go.mod` (with `replace … => ../` pointing at the working
copy) means the main module's manifest stays literally dependency-free — it has
no `go.sum`. See [`../CONTRIBUTING.md`](../CONTRIBUTING.md) for the full policy.

## Run

```bash
go -C conformance test ./...    # from the repo root
# or, from this directory:
go test ./...
```

## What it checks

- **`TestReference_ReadsOurOutput`** — a third-party reader parses exifscalpel's
  `Build` output correctly (Make, Adobe Software, ISO in the Exif sub-IFD, GPS).
  Proves we emit standards-compliant bytes, not just bytes we can read back.
- **`TestReference_ConfirmsScrubAndGPSRemoval`** — after `RemoveIFD(GPS)` and a
  Software scrub, the reference sees no GPS data and no Adobe signature, while
  Make survives. Proves our *edits* produce genuinely clean output.

## Next (ideas for later sessions)

- **XMP is intentionally not covered here yet** (handoff Phase 3). A model-based
  reference XMP library is a poor oracle for our byte-surgery approach (it
  reserializes and may drop unmodeled namespaces); prefer `exiftool` (an
  external binary, not a Go dep) for "what fields exist" checks if needed.
- Add a **fuzz corpus** (stdlib `testing.F`) and/or property tests asserting
  `Parse∘Build` stability and length-preservation invariants.
- Optionally cross-check against `github.com/bep/imagemeta` as a second oracle.
