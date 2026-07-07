// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xmp

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// Property identifies an XMP property by its namespace URI and local name,
// independent of the namespace prefix a given file happens to bind (one file may
// write "AnalogExif:Film", another a different prefix for the same URI; the URI
// is the stable identifier).
type Property struct {
	Namespace string // namespace URI, e.g. "http://analogexif.sourceforge.net/ns/"
	Name      string // local name, e.g. "Film"
}

// ReadProperties extracts the requested simple (scalar) XMP properties from a
// raw XMP APP1 payload, matching each by namespace URI and local name. Both
// forms real writers use are recognised: attribute form (ns:Name="value", how
// AnalogExif/aux write their scalars on rdf:Description) and simple element form
// (<ns:Name>value</ns:Name>). XML entities are decoded by the token reader. The
// first occurrence of a property wins; properties absent or empty are omitted.
//
// This is a read-only primitive with no vocabulary of its own — the caller names
// the namespaces and fields it wants (e.g. the AnalogExif film schema, or
// aux:Lens) and policy stays in the caller. Results are keyed by the same
// Property values passed in. The payload must begin with the Adobe xap namespace
// signature (as jpeg.Segment.Data for an XMP segment does).
//
// Parsing is best-effort: the error result is non-nil only when the payload lacks
// the XMP signature. A malformed or truncated packet is not reported as an error —
// whatever was collected before the fault is returned, so an absent property and a
// parse failure are indistinguishable from the result alone.
func ReadProperties(payload []byte, want []Property) (map[Property]string, error) {
	if !bytes.HasPrefix(payload, xmpSig) {
		return nil, fmt.Errorf("xmp: not an XMP segment")
	}

	// Property is already the exact (namespace, name) key shape, so it doubles as
	// the lookup key; matching is by namespace URI + local name, prefix-independent.
	index := make(map[Property]struct{}, len(want))
	for _, p := range want {
		index[p] = struct{}{}
	}

	out := make(map[Property]string)
	dec := xml.NewDecoder(bytes.NewReader(payload[len(xmpSig):]))
	dec.Strict = false

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break // best-effort: return whatever was collected before the error
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		// Attribute form: ns:Name="value" on any element.
		for _, a := range se.Attr {
			p := Property{a.Name.Space, a.Name.Local}
			if _, ok := index[p]; !ok || a.Value == "" {
				continue
			}
			if _, seen := out[p]; !seen {
				out[p] = a.Value
			}
		}

		// Element form: <ns:Name>value</ns:Name>.
		p := Property{se.Name.Space, se.Name.Local}
		if _, ok := index[p]; ok {
			if _, seen := out[p]; !seen {
				if v := nextCharData(dec); v != "" {
					out[p] = v
				}
			}
		}
	}
	return out, nil
}
