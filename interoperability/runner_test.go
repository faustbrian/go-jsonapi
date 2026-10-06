package main

import (
	"bytes"
	"os"
	"runtime/debug"
	"testing"
)

func TestObservationsIdentifySelectedLocalModule(t *testing.T) {
	t.Parallel()

	selected := selectedLocalVersion(t)
	t.Logf("selected local dependency: %s", selected)
	observations, err := observe()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, result := range observations {
		if result.implementation == localName {
			count++
			if result.version != selected {
				t.Errorf("local observation identity = %q, selected module = %q", result.version, selected)
			}
		}
	}
	if count != 5 {
		t.Fatalf("local observations = %d, want 5", count)
	}
}

// This oracle reads Go's actual selected build metadata independently of the
// production identity helper and of the recorded behavior matrix.
func selectedLocalVersion(t *testing.T) string {
	t.Helper()
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Fatal("missing selected build metadata")
	}
	for _, dependency := range info.Deps {
		if dependency.Path != "github.com/faustbrian/go-jsonapi/v2" {
			continue
		}
		if dependency.Replace != nil {
			if dependency.Replace.Version != "" && dependency.Replace.Version != "(devel)" {
				return dependency.Replace.Path + "@" + dependency.Replace.Version
			}
			return "workspace"
		}
		if dependency.Version == "(devel)" {
			return "workspace"
		}
		if dependency.Version != "" {
			return dependency.Version
		}
		t.Fatal("selected public module lacks a version")
	}
	t.Fatal("local dependency missing from selected build metadata")
	return ""
}

func TestObservationsMatchRecordedMatrix(t *testing.T) {
	t.Parallel()

	observations, err := observe()
	if err != nil {
		t.Fatal(err)
	}
	selected := selectedLocalVersion(t)
	for index := range observations {
		if observations[index].implementation != localName {
			continue
		}
		if observations[index].version != selected {
			t.Fatalf("matrix observation identity = %q, selected module = %q", observations[index].version, selected)
		}
		// The recorded matrix is the unchanged workspace behavior reference.
		// Normalize only after independently verifying the actual provenance;
		// observe and write must retain the selected identity in real output.
		observations[index].version = "workspace"
	}
	var actual bytes.Buffer
	if err := write(&actual, observations); err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile("../specification/differential.tsv")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual.Bytes(), expected) {
		t.Fatalf("differential matrix differs\nactual:\n%s\nexpected:\n%s", actual.Bytes(), expected)
	}
}
