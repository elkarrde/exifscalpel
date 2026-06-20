# Contributing

## Dependency policy

exifscalpel ships **zero runtime dependencies**. The "no deps" motto is about
baggage in the *shipped artifacts* — the Lapis and TidyExif binaries that import
this library — not about how we develop it.

Two tiers:

### Runtime — stdlib only, forever

The library itself (`jpeg/`, `exif/`, `xmp/`, and the root package) imports the
Go standard library and nothing else. **No non-test file in the main module may
import a third-party package.** This is a hard rule, and it's load-bearing:

- It's why both consumers can adopt exifscalpel without inheriting a single
  transitive dependency.
- It's why the main module has **no `go.sum`** at all — a nice invariant to
  keep an eye on. If a `go.sum` appears in the repo root, something leaked.

If you find yourself wanting a third-party package in engine code, stop — that's
a design smell. The engines are deliberately hand-rolled for byte-level control
(see `exifscalpel-HANDOFF.md` §7 and §9).

### Dev / test — unrestricted

For development and testing, use whatever serves correctness and ergonomics:
reference implementations for differential testing, fuzzing (stdlib, free),
property-based testing, nicer assertions. Go guarantees these never reach
consumers:

- A dependency imported only from `_test.go` files is never compiled into
  anything that imports the library.
- Module-graph pruning (go 1.17+, and we target 1.22) keeps such deps out of
  consumers' module graphs entirely.

**Caveat:** Go has no `devDependencies` field — a test-only `require` still
lands in `go.mod`, un-separated from runtime requires. So:

- **Heavy dev tooling lives in its own module.** The `conformance/` directory
  is a separate Go module (its own `go.mod`, with `replace … => ../`) that
  imports a mature reference EXIF reader to validate our hand-rolled engine.
  None of that touches the main module's manifest, which stays dependency-free.
- Lightweight, broadly-trusted test helpers (e.g. `go-cmp`) *may* go in the main
  `go.mod` if they genuinely earn it — but never a reference/parser-grade
  dependency, and never anything imported from non-test code.

## Running tests

```bash
go test ./...                  # library — stdlib only, fast
go test -cover ./...
go -C conformance test ./...   # differential vs. dsoprea/go-exif (separate module)
gofmt -l .                     # must print nothing
```

`./...` from the repo root does **not** descend into `conformance/` — it is a
separate module and is run explicitly.
