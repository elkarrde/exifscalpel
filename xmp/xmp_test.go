// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xmp

import (
	"bytes"
	"testing"
)

// makeSeg prepends the Adobe xap signature to XML content, producing a payload
// shaped like jpeg.Segment.Data for an XMP segment.
func makeSeg(content string) []byte {
	return append(append([]byte(nil), xmpSig...), []byte(content)...)
}

// sampleXMP is a representative Lightroom-style XMP block with all six target
// fields, history entries in element form.
const sampleXMP = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:xmp="http://ns.adobe.com/xap/1.0/"
        xmlns:xmpMM="http://ns.adobe.com/xap/1.0/mm/"
        xmlns:stEvt="http://ns.adobe.com/xap/1.0/sType/ResourceEvent#"
        xmp:CreatorTool="Adobe Lightroom Classic 13.0"
        xmp:MetadataDate="2024-01-15T12:00:00+01:00"
        xmpMM:DocumentID="xmp.did:abc123"
        xmpMM:InstanceID="xmp.iid:def456"
        xmpMM:OriginalDocumentID="xmp.did:xyz789">
      <xmpMM:History>
        <rdf:Seq>
          <rdf:li rdf:parseType="Resource">
            <stEvt:action>saved</stEvt:action>
            <stEvt:softwareAgent>Adobe Lightroom Classic 13.0</stEvt:softwareAgent>
          </rdf:li>
          <rdf:li rdf:parseType="Resource">
            <stEvt:action>saved</stEvt:action>
            <stEvt:softwareAgent>Adobe Photoshop 2024</stEvt:softwareAgent>
          </rdf:li>
        </rdf:Seq>
      </xmpMM:History>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`

// attrHistoryXMP uses the attribute form of history entries that Lightroom and
// Photoshop actually write (<rdf:li stEvt:softwareAgent="..."/>). Regression
// fixture: the original parser only handled the element form and missed these.
const attrHistoryXMP = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:xmp="http://ns.adobe.com/xap/1.0/"
        xmlns:xmpMM="http://ns.adobe.com/xap/1.0/mm/"
        xmlns:stEvt="http://ns.adobe.com/xap/1.0/sType/ResourceEvent#"
        xmp:CreatorTool="">
      <xmpMM:History>
        <rdf:Seq>
          <rdf:li stEvt:action="saved" stEvt:softwareAgent="Adobe Photoshop Lightroom 5.0 (Windows)" stEvt:when="2017-10-25T00:50:38+02:00"/>
          <rdf:li stEvt:action="saved" stEvt:softwareAgent="Adobe Photoshop CS6 (Windows)" stEvt:when="2017-10-25T01:06:07+02:00"/>
        </rdf:Seq>
      </xmpMM:History>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`

func TestParse(t *testing.T) {
	f, err := Parse(makeSeg(sampleXMP))
	if err != nil {
		t.Fatal(err)
	}
	if f.CreatorTool != "Adobe Lightroom Classic 13.0" {
		t.Errorf("CreatorTool = %q", f.CreatorTool)
	}
	if f.MetadataDate != "2024-01-15T12:00:00+01:00" {
		t.Errorf("MetadataDate = %q", f.MetadataDate)
	}
	if f.DocumentID != "xmp.did:abc123" {
		t.Errorf("DocumentID = %q", f.DocumentID)
	}
	if f.InstanceID != "xmp.iid:def456" {
		t.Errorf("InstanceID = %q", f.InstanceID)
	}
	if f.OriginalDocumentID != "xmp.did:xyz789" {
		t.Errorf("OriginalDocumentID = %q", f.OriginalDocumentID)
	}
	if len(f.SoftwareAgents) != 2 {
		t.Fatalf("SoftwareAgents len = %d, want 2", len(f.SoftwareAgents))
	}
	if f.SoftwareAgents[0] != "Adobe Lightroom Classic 13.0" {
		t.Errorf("SoftwareAgents[0] = %q", f.SoftwareAgents[0])
	}
	if f.SoftwareAgents[1] != "Adobe Photoshop 2024" {
		t.Errorf("SoftwareAgents[1] = %q", f.SoftwareAgents[1])
	}
}

func TestParseAttributeHistory(t *testing.T) {
	f, err := Parse(makeSeg(attrHistoryXMP))
	if err != nil {
		t.Fatal(err)
	}
	if !f.Any() {
		t.Fatal("Any = false for attribute-form history (regression)")
	}
	if len(f.SoftwareAgents) != 2 {
		t.Fatalf("SoftwareAgents len = %d, want 2", len(f.SoftwareAgents))
	}
	if f.SoftwareAgents[0] != "Adobe Photoshop Lightroom 5.0 (Windows)" ||
		f.SoftwareAgents[1] != "Adobe Photoshop CS6 (Windows)" {
		t.Errorf("agents = %q", f.SoftwareAgents)
	}
}

// TestCleanAttributeHistoryEmptiesAgents is the mandatory regression (handoff
// §5): Clean on attribute-form history must empty the agents, preserve length,
// leave no "Adobe Photoshop" substring, and reparse to no Adobe data.
func TestCleanAttributeHistoryEmptiesAgents(t *testing.T) {
	payload := makeSeg(attrHistoryXMP)
	out, changed, err := Clean(payload, nil)
	if err != nil || !changed {
		t.Fatalf("Clean: changed=%v err=%v", changed, err)
	}
	if len(out) != len(payload) {
		t.Errorf("length changed: %d → %d", len(payload), len(out))
	}
	if bytes.Contains(out, []byte("Adobe Photoshop")) {
		t.Error("attribute-form softwareAgent not emptied")
	}
	f, err := Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if f.Any() {
		t.Errorf("still reports Adobe data after clean: %+v", f)
	}
}

func TestParseNotXMP(t *testing.T) {
	_, err := Parse([]byte("not an xmp segment"))
	if err == nil {
		t.Error("expected error for non-XMP input")
	}
}

func TestCleanFields(t *testing.T) {
	f := &Fields{
		CreatorTool:        "Adobe Lightroom Classic 13.0",
		MetadataDate:       "2024-01-15",
		DocumentID:         "xmp.did:abc",
		InstanceID:         "xmp.iid:def",
		OriginalDocumentID: "xmp.did:xyz",
		SoftwareAgents:     []string{"Adobe Lightroom Classic 13.0", "Adobe Photoshop 2024"},
	}

	cleaned := cleanFields(f, nil)

	if cleaned.CreatorTool != "" || cleaned.MetadataDate != "" || cleaned.DocumentID != "" ||
		cleaned.InstanceID != "" || cleaned.OriginalDocumentID != "" {
		t.Errorf("cleanFields did not zero all simple fields: %+v", cleaned)
	}
	for i, a := range cleaned.SoftwareAgents {
		if a != "" {
			t.Errorf("SoftwareAgents[%d] = %q, want empty", i, a)
		}
	}
	// original must be unchanged
	if f.CreatorTool == "" {
		t.Error("cleanFields mutated the input")
	}
}

func TestCleanFieldsWithReplacements(t *testing.T) {
	f := &Fields{
		CreatorTool: "Adobe Lightroom Classic 13.0",
		DocumentID:  "xmp.did:abc",
	}
	cleaned := cleanFields(f, map[string]string{"CreatorTool": "cleaned"})
	if cleaned.CreatorTool != "cleaned" {
		t.Errorf("CreatorTool = %q, want %q", cleaned.CreatorTool, "cleaned")
	}
	if cleaned.DocumentID != "" {
		t.Errorf("DocumentID = %q, want empty", cleaned.DocumentID)
	}
}

func TestCleanPreservesLength(t *testing.T) {
	payload := makeSeg(sampleXMP)
	out, changed, err := Clean(payload, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if len(out) != len(payload) {
		t.Errorf("length: got %d, want %d", len(out), len(payload))
	}
}

func TestCleanZeroesValues(t *testing.T) {
	payload := makeSeg(sampleXMP)
	out, _, err := Clean(payload, nil)
	if err != nil {
		t.Fatal(err)
	}

	reparsed, err := Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if reparsed.CreatorTool != "" {
		t.Errorf("CreatorTool after clean = %q", reparsed.CreatorTool)
	}
	if reparsed.DocumentID != "" {
		t.Errorf("DocumentID after clean = %q", reparsed.DocumentID)
	}
	if reparsed.InstanceID != "" {
		t.Errorf("InstanceID after clean = %q", reparsed.InstanceID)
	}
	if reparsed.OriginalDocumentID != "" {
		t.Errorf("OriginalDocumentID after clean = %q", reparsed.OriginalDocumentID)
	}
	for i, a := range reparsed.SoftwareAgents {
		if a != "" {
			t.Errorf("SoftwareAgents[%d] after clean = %q", i, a)
		}
	}
}

func TestCleanReplacesValues(t *testing.T) {
	payload := makeSeg(sampleXMP)
	out, changed, err := Clean(payload, map[string]string{"CreatorTool": "scrubbed"})
	if err != nil || !changed {
		t.Fatalf("Clean: changed=%v err=%v", changed, err)
	}
	if len(out) != len(payload) {
		t.Errorf("length changed: %d → %d", len(payload), len(out))
	}
	reparsed, err := Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if reparsed.CreatorTool != "scrubbed" {
		t.Errorf("CreatorTool after clean = %q, want %q", reparsed.CreatorTool, "scrubbed")
	}
}

func TestCleanNoAdobeData(t *testing.T) {
	// A valid XMP payload with no target fields populated.
	const empty = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""/>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`
	payload := makeSeg(empty)
	out, changed, err := Clean(payload, nil)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("expected changed=false for payload with no Adobe data")
	}
	if !bytes.Equal(out, payload) {
		t.Error("result differs from input when nothing to clean")
	}
}

func TestCleanNotXMP(t *testing.T) {
	_, _, err := Clean([]byte("not an xmp segment"), nil)
	if err == nil {
		t.Error("expected error for non-XMP input")
	}
}

func TestAny(t *testing.T) {
	empty := &Fields{}
	if empty.Any() {
		t.Error("empty Fields.Any() = true")
	}

	withData := &Fields{CreatorTool: "Lightroom"}
	if !withData.Any() {
		t.Error("Fields with CreatorTool.Any() = false")
	}

	withAgents := &Fields{SoftwareAgents: []string{"", "Photoshop"}}
	if !withAgents.Any() {
		t.Error("Fields with non-empty SoftwareAgent.Any() = false")
	}
}
