// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package exifscalpel is a small library of JPEG metadata write/edit
// primitives: byte-level parsing and surgical, minimal-diff editing of a
// JPEG's segments, its EXIF/TIFF block, and its XMP packet.
//
// It exists to de-duplicate code shared between two sibling CLIs (tidy-exif,
// which empties Adobe software signatures, and lapis, a privacy stripper) and
// to give them a single, well-tested core. The dependency flows one way: both
// CLIs import exifscalpel; exifscalpel imports neither. It ships primitives
// only — no orchestration, flags, printing, or file walking. Policy (the
// Adobe-only gate, paranoia levels, output) stays in the consumers.
//
// # Packages
//
// The library is three independent packages:
//
//   - jpeg — JPEG segment parse/write plus segment identification
//     (IsEXIF/IsXMP). Depends only on the standard library.
//   - exif — TIFF/IFD parse and rebuild, with tag-level edit helpers.
//   - xmp — XMP field-level surgery, length-preserving.
//
// exif and xmp take a segment payload ([]byte) and return bytes; they do not
// import jpeg, so each package stays independently usable and testable.
// Segment identification (the "Exif\x00\x00" and Adobe xap namespace
// signatures) lives in jpeg.
//
// # Invariants
//
// These behaviors cost real debugging time in the predecessor tools and are
// baked into the code and tests here:
//
//  1. XMP xmpMM:History stEvt:softwareAgent appears in ATTRIBUTE form
//     (<rdf:li stEvt:softwareAgent="Adobe Photoshop CS6 (Windows)"/>), not
//     only element form. The xmp package handles both on parse and clean;
//     the attribute-form regression fixture is mandatory.
//
//  2. Length-preserving edits avoid rewriting JPEG offsets. XMP pads the
//     cleaned XML with whitespace inside <?xpacket?> so the APP1 segment keeps
//     its byte length. The exif package can instead rebuild the payload
//     (length may change) — correct when removing tags or IFDs. Both modes
//     are available; neither is forced.
//
//  3. EXIF value storage: a TIFF entry value of four bytes or fewer is stored
//     inline; larger values live at an offset relative to the TIFF header
//     (after "Exif\x00\x00"). The exif parser resolves every entry's value to
//     actual bytes.
//
//  4. JPEG parsing stops at SOS (0xFF 0xDA) and returns the compressed tail
//     verbatim. Standalone markers (SOI, EOI, RST) carry no length; legal
//     0xFF padding before a marker is skipped.
package exifscalpel
