package jsonapi

import "testing"

func FuzzUnmarshal(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{"data":null}`),
		[]byte(`{"data":[]}`),
		[]byte(`{"data":{"type":"articles","id":"1"}}`),
		[]byte(`{"errors":[{"status":"400","title":"Bad request"}]}`),
		[]byte(`{"data":null,"data":[]}`),
		[]byte(`{"data":`),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, payload []byte) {
		document, err := Unmarshal(payload)
		if err != nil {
			return
		}
		canonical, err := Marshal(document)
		if err != nil {
			t.Fatalf("accepted document cannot be marshaled: %v", err)
		}
		if _, err := Unmarshal(canonical); err != nil {
			t.Fatalf("canonical document cannot be decoded: %v", err)
		}
	})
}

func FuzzUnmarshalAtomic(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{"atomic:operations":[{"op":"remove","href":"/articles/1"}]}`),
		[]byte(`{"atomic:results":[{}]}`),
		[]byte(`{"errors":[{"status":"409"}]}`),
		[]byte(`{"data":null}`),
		[]byte(`{"atomic:operations":[{"op":"add"}]}`),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, payload []byte) {
		document, err := UnmarshalAtomic(payload)
		if err != nil {
			return
		}
		canonical, err := MarshalAtomic(document)
		if err != nil {
			t.Fatalf("accepted Atomic document cannot be marshaled: %v", err)
		}
		if _, err := UnmarshalAtomic(canonical); err != nil {
			t.Fatalf("canonical Atomic document cannot be decoded: %v", err)
		}
	})
}
