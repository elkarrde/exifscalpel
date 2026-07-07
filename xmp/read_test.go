// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xmp

import "testing"

const (
	nsAnalogExif = "http://analogexif.sourceforge.net/ns/"
	nsAux        = "http://ns.adobe.com/exif/1.0/aux/"
)

// analogExifPayload mirrors a real ExifNotes/AnalogExif packet: scalars in
// attribute form on rdf:Description (Film, FilmMaker, aux:Lens) plus one field
// in element form (Developer), with an XML entity to decode (B&amp;W).
func analogExifPayload() []byte {
	xml := `<?xpacket begin='' id='W5M0MpCehiHzreSzNTczkc9d'?>` +
		`<x:xmpmeta xmlns:x='adobe:ns:meta/'>` +
		`<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>` +
		`<rdf:Description rdf:about=''` +
		` xmlns:AnalogExif='http://analogexif.sourceforge.net/ns/'` +
		` xmlns:aux='http://ns.adobe.com/exif/1.0/aux/'` +
		` AnalogExif:Film='Mono' AnalogExif:FilmMaker='KosmoFoto'` +
		` AnalogExif:DevelopProcess='B&amp;W'` +
		` aux:Lens='Tokina SZ-X 80-200mm F4.5-5.6'>` +
		`<AnalogExif:Developer>Adonal</AnalogExif:Developer>` +
		`</rdf:Description></rdf:RDF></x:xmpmeta><?xpacket end='w'?>`
	return append([]byte("http://ns.adobe.com/xap/1.0/\x00"), []byte(xml)...)
}

func TestReadProperties(t *testing.T) {
	want := []Property{
		{nsAnalogExif, "Film"},           // attribute form
		{nsAnalogExif, "FilmMaker"},      // attribute form
		{nsAnalogExif, "DevelopProcess"}, // attribute form with entity
		{nsAnalogExif, "Developer"},      // element form
		{nsAux, "Lens"},                  // attribute form, different namespace
		{nsAnalogExif, "Missing"},        // absent -> omitted
	}
	got, err := ReadProperties(analogExifPayload(), want)
	if err != nil {
		t.Fatalf("ReadProperties: %v", err)
	}

	expect := map[Property]string{
		{nsAnalogExif, "Film"}:           "Mono",
		{nsAnalogExif, "FilmMaker"}:      "KosmoFoto",
		{nsAnalogExif, "DevelopProcess"}: "B&W",
		{nsAnalogExif, "Developer"}:      "Adonal",
		{nsAux, "Lens"}:                  "Tokina SZ-X 80-200mm F4.5-5.6",
	}
	for p, v := range expect {
		if got[p] != v {
			t.Errorf("%s:%s = %q, want %q", p.Namespace, p.Name, got[p], v)
		}
	}
	if _, ok := got[Property{nsAnalogExif, "Missing"}]; ok {
		t.Error("absent property should be omitted from the result")
	}
	if len(got) != len(expect) {
		t.Errorf("got %d properties, want %d: %v", len(got), len(expect), got)
	}
}

// TestReadPropertiesNamespaceIsolation confirms matching is by namespace URI,
// not just local name: a different namespace's same-named field must not match.
func TestReadPropertiesNamespaceIsolation(t *testing.T) {
	got, err := ReadProperties(analogExifPayload(), []Property{{nsAux, "Film"}})
	if err != nil {
		t.Fatalf("ReadProperties: %v", err)
	}
	if v, ok := got[Property{nsAux, "Film"}]; ok {
		t.Errorf("aux:Film should not match AnalogExif:Film, got %q", v)
	}
}

// TestReadPropertiesNotXMP guards the signature check.
func TestReadPropertiesNotXMP(t *testing.T) {
	if _, err := ReadProperties([]byte("not xmp"), []Property{{nsAux, "Lens"}}); err == nil {
		t.Error("expected an error for a non-XMP payload")
	}
}

const nsT = "urn:test"

// xmpDescription wraps a body of rdf:Description content in a valid XMP packet
// with the rdf and t (urn:test) namespaces bound.
func xmpDescription(body string) []byte {
	xml := `<x:xmpmeta xmlns:x='adobe:ns:meta/'>` +
		`<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>` +
		`<rdf:Description rdf:about='' xmlns:t='` + nsT + `'>` +
		body +
		`</rdf:Description></rdf:RDF></x:xmpmeta>`
	return append([]byte("http://ns.adobe.com/xap/1.0/\x00"), []byte(xml)...)
}

// TestReadPropertiesElementForms exercises the element-form value reader across
// the cases real writers produce: entity decoding, whitespace trimming, and an
// empty element (which must be omitted, not stored as "").
func TestReadPropertiesElementForms(t *testing.T) {
	payload := xmpDescription(
		`<t:Entity>B&amp;W &lt;push&gt;</t:Entity>` +
			"<t:Spaced>\n    Ilford HP5\n  </t:Spaced>" +
			`<t:Empty></t:Empty>`)
	want := []Property{{nsT, "Entity"}, {nsT, "Spaced"}, {nsT, "Empty"}}
	got, err := ReadProperties(payload, want)
	if err != nil {
		t.Fatalf("ReadProperties: %v", err)
	}
	if v := got[Property{nsT, "Entity"}]; v != "B&W <push>" {
		t.Errorf("entity: got %q, want %q", v, "B&W <push>")
	}
	if v := got[Property{nsT, "Spaced"}]; v != "Ilford HP5" {
		t.Errorf("whitespace: got %q, want %q", v, "Ilford HP5")
	}
	if v, ok := got[Property{nsT, "Empty"}]; ok {
		t.Errorf("empty element should be omitted, got %q", v)
	}
}

// TestReadPropertiesFirstOccurrenceWins pins the precedence contract: attribute
// form (encountered first, on rdf:Description) wins over a later element form of
// the same property, and among repeated elements the first wins.
func TestReadPropertiesFirstOccurrenceWins(t *testing.T) {
	// Same property in both forms: the attribute on rdf:Description is tokenised
	// before the nested element, so it must win.
	both := append([]byte("http://ns.adobe.com/xap/1.0/\x00"),
		[]byte(`<x:xmpmeta xmlns:x='adobe:ns:meta/'>`+
			`<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>`+
			`<rdf:Description rdf:about='' xmlns:t='`+nsT+`' t:Field='attr'>`+
			`<t:Field>elem</t:Field>`+
			`</rdf:Description></rdf:RDF></x:xmpmeta>`)...)
	got, err := ReadProperties(both, []Property{{nsT, "Field"}})
	if err != nil {
		t.Fatalf("ReadProperties: %v", err)
	}
	if v := got[Property{nsT, "Field"}]; v != "attr" {
		t.Errorf("attribute form should win over element form: got %q, want %q", v, "attr")
	}

	// Repeated element form: first occurrence wins.
	dup := xmpDescription(`<t:Dup>first</t:Dup><t:Dup>second</t:Dup>`)
	got, err = ReadProperties(dup, []Property{{nsT, "Dup"}})
	if err != nil {
		t.Fatalf("ReadProperties: %v", err)
	}
	if v := got[Property{nsT, "Dup"}]; v != "first" {
		t.Errorf("duplicate element: got %q, want %q (first wins)", v, "first")
	}
}

// TestReadPropertiesMalformedIsBestEffort pins the documented error contract:
// a truncated packet is not reported as an error, and values collected before
// the fault are still returned.
func TestReadPropertiesMalformedIsBestEffort(t *testing.T) {
	// Well-formed prefix carrying t:A as an attribute, then an unterminated tag.
	payload := append([]byte("http://ns.adobe.com/xap/1.0/\x00"),
		[]byte(`<rdf:Description xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'`+
			` xmlns:t='`+nsT+`' t:A='kept'><t:B>never closed`)...)
	got, err := ReadProperties(payload, []Property{{nsT, "A"}, {nsT, "B"}})
	if err != nil {
		t.Fatalf("malformed XML must not error, got: %v", err)
	}
	if v := got[Property{nsT, "A"}]; v != "kept" {
		t.Errorf("value before the fault should survive: got %q, want %q", v, "kept")
	}
}
