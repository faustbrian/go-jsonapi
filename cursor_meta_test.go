package jsonapi

import (
	"errors"
	"testing"
)

func TestCursorPageMetaBuildsCanonicalProfileMetadata(t *testing.T) {
	t.Parallel()

	total := int64(200)
	bestGuess := int64(210)
	rangeTruncated := true
	meta, err := (CursorPageMeta{
		RangeTruncated: &rangeTruncated,
		Total:          &total,
		EstimatedTotal: &CursorEstimatedTotal{BestGuess: &bestGuess},
	}).Meta()
	if err != nil {
		t.Fatalf("build cursor page meta: %v", err)
	}

	payload, err := Marshal(Document{Data: ResourceCollection(), Meta: meta})
	if err != nil {
		t.Fatalf("marshal document: %v", err)
	}
	want := `{"data":[],"meta":{"page":{"rangeTruncated":true,"total":200,"estimatedTotal":{"bestGuess":210}}}}`
	if string(payload) != want {
		t.Fatalf("unexpected metadata:\n got: %s\nwant: %s", payload, want)
	}
}

func TestCursorPageMetaPreservesEmptyEstimateObject(t *testing.T) {
	t.Parallel()

	meta, err := (CursorPageMeta{EstimatedTotal: &CursorEstimatedTotal{}}).Meta()
	if err != nil {
		t.Fatalf("build empty estimate: %v", err)
	}
	payload, err := Marshal(Document{Data: ResourceCollection(), Meta: meta})
	if err != nil {
		t.Fatalf("marshal document: %v", err)
	}
	want := `{"data":[],"meta":{"page":{"estimatedTotal":{}}}}`
	if string(payload) != want {
		t.Fatalf("unexpected metadata: got %s, want %s", payload, want)
	}
}

func TestParseCursorPageMetaFromDecodedDocument(t *testing.T) {
	t.Parallel()

	document, err := Unmarshal([]byte(`{
		"data":[],
		"meta":{"page":{"rangeTruncated":false,"total":200,"estimatedTotal":{"bestGuess":210}}}
	}`))
	if err != nil {
		t.Fatalf("decode document: %v", err)
	}
	metadata, present, err := ParseCursorPageMeta(document.Meta)
	if err != nil {
		t.Fatalf("parse cursor metadata: %v", err)
	}
	if !present || metadata.RangeTruncated == nil || *metadata.RangeTruncated ||
		metadata.Total == nil || *metadata.Total != 200 ||
		metadata.EstimatedTotal == nil || metadata.EstimatedTotal.BestGuess == nil ||
		*metadata.EstimatedTotal.BestGuess != 210 {
		t.Fatalf("unexpected parsed metadata: %#v", metadata)
	}
}

func TestCursorPageMetaRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	negative := int64(-1)
	if _, err := (CursorPageMeta{Total: &negative}).Meta(); err == nil {
		t.Fatal("expected negative total error")
	}

	tests := []struct {
		meta Meta
		path string
	}{
		{meta: Meta{"page": "invalid"}, path: "/meta/page"},
		{meta: Meta{"page": map[string]any{"total": 1.5}}, path: "/meta/page/total"},
		{meta: Meta{"page": map[string]any{"rangeTruncated": "yes"}}, path: "/meta/page/rangeTruncated"},
		{meta: Meta{"page": map[string]any{"estimatedTotal": 10}}, path: "/meta/page/estimatedTotal"},
		{meta: Meta{"page": map[string]any{"estimatedTotal": map[string]any{"bestGuess": -1.0}}}, path: "/meta/page/estimatedTotal/bestGuess"},
	}
	for _, test := range tests {
		_, _, err := ParseCursorPageMeta(test.meta)
		var validationError *ValidationError
		if !errors.As(err, &validationError) {
			t.Fatalf("expected ValidationError, got %T: %v", err, err)
		}
		if validationError.Violations[0].Path != test.path {
			t.Fatalf("unexpected violation: %#v", validationError.Violations[0])
		}
	}
}

func TestCursorItemMetaRoundTrip(t *testing.T) {
	t.Parallel()

	resource := ResourceObject{
		Type: "articles",
		ID:   "1",
		Meta: CursorItemMeta("opaque"),
	}
	cursor, present, err := ParseCursorItemMeta(resource.Meta)
	if err != nil {
		t.Fatalf("parse item cursor: %v", err)
	}
	if !present || cursor != "opaque" {
		t.Fatalf("unexpected cursor: %q present=%v", cursor, present)
	}
}

func TestCursorPageMetaPreservesLargeDecodedIntegers(t *testing.T) {
	t.Parallel()

	document, err := Unmarshal([]byte(`{
		"data":[],
		"meta":{"page":{"total":9007199254740993}}
	}`))
	if err != nil {
		t.Fatalf("decode document: %v", err)
	}
	metadata, _, err := ParseCursorPageMeta(document.Meta)
	if err != nil {
		t.Fatalf("parse cursor metadata: %v", err)
	}
	if metadata.Total == nil || *metadata.Total != 9007199254740993 {
		t.Fatalf("large integer lost precision: %#v", metadata.Total)
	}
}
