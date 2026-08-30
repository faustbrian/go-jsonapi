package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"runtime/debug"

	peer "github.com/DataDog/jsonapi"
	local "github.com/faustbrian/go-jsonapi"
)

const (
	localName  = "faustbrian/go-jsonapi"
	peerName   = "DataDog/jsonapi"
	peerModule = "github.com/DataDog/jsonapi"
)

type article struct {
	ID    string `jsonapi:"primary,articles"`
	Title string `jsonapi:"attribute" json:"title"`
}

type person struct {
	ID string `jsonapi:"primary,people"`
}

type articleWithAuthor struct {
	ID     string  `jsonapi:"primary,articles"`
	Author *person `jsonapi:"relationship"`
}

type observation struct {
	decisionID     string
	caseName       string
	implementation string
	version        string
	outcome        string
	classification string
}

func main() {
	observations, err := observe()
	if err != nil {
		panic(err)
	}
	if err := write(os.Stdout, observations); err != nil {
		panic(err)
	}
}

func observe() ([]observation, error) {
	peerVersion, err := moduleVersion(peerModule)
	if err != nil {
		return nil, err
	}
	observations := make([]observation, 0, 10)
	add := func(decisionID, caseName, localOutcome, peerOutcome string) {
		classification := "deliberate policy difference"
		if localOutcome == peerOutcome {
			classification = "maintained peer agreement"
		}
		observations = append(observations,
			observation{decisionID, caseName, localName, "workspace", localOutcome, classification},
			observation{decisionID, caseName, peerName, peerVersion, peerOutcome, classification},
		)
	}

	const unknownMember = `{"data":{"type":"articles","id":"1","future-member":true}}`
	if _, err := local.Unmarshal([]byte(unknownMember)); err != nil {
		return nil, fmt.Errorf("local unknown-member observation: %w", err)
	}
	var peerArticle article
	if err := peer.Unmarshal([]byte(unknownMember), &peerArticle); err != nil {
		return nil, fmt.Errorf("peer unknown-member observation: %w", err)
	}
	add("JSONAPI-DEC-001", "unknown-resource-member", "accepted-and-ignored", "accepted-and-ignored")

	const relationshipShapes = `{"data":{"type":"articles","id":"1","relationships":{"author":{"data":null},"comments":{"data":[]}}}}`
	if _, err := local.Unmarshal([]byte(relationshipShapes)); err != nil {
		return nil, fmt.Errorf("local relationship-shape observation: %w", err)
	}
	var peerRelationships struct {
		ID       string     `jsonapi:"primary,articles"`
		Author   *article   `jsonapi:"relationship"`
		Comments []*article `jsonapi:"relationship"`
	}
	if err := peer.Unmarshal([]byte(relationshipShapes), &peerRelationships); err != nil {
		return nil, fmt.Errorf("peer relationship-shape observation: %w", err)
	}
	if peerRelationships.Author != nil || peerRelationships.Comments != nil {
		return nil, errors.New("peer relationship-shape observation did not collapse both shapes to nil")
	}
	add("JSONAPI-DEC-004", "null-and-empty-linkage", "accepted-distinct-shapes", "accepted-collapsed-shapes")

	const duplicateResources = `{"data":{"type":"articles","id":"1","relationships":{"author":{"data":{"type":"people","id":"9"}}}},"included":[{"type":"people","id":"9"},{"type":"people","id":"9"}]}`
	if _, err := local.Unmarshal([]byte(duplicateResources)); err == nil {
		return nil, errors.New("local duplicate-included-resource observation unexpectedly succeeded")
	}
	var peerArticleWithAuthor articleWithAuthor
	if err := peer.Unmarshal([]byte(duplicateResources), &peerArticleWithAuthor, peer.UnmarshalCheckUniqueness()); !errors.Is(err, peer.ErrNonuniqueResource) {
		return nil, fmt.Errorf("peer duplicate-included-resource observation: %w", err)
	}
	add("JSONAPI-DEC-005", "duplicate-included-resource", "rejected", "rejected")

	query := url.Values{"fields[articles]": {"title"}}
	parsed, err := local.ParseQuery(query)
	if err != nil || len(parsed.Fields["articles"]) != 1 || parsed.Fields["articles"][0] != "title" {
		return nil, fmt.Errorf("local sparse-field observation: %w", err)
	}
	peerPayload, err := peer.Marshal(article{ID: "1", Title: "A"}, peer.MarshalFields(query))
	if err != nil {
		return nil, fmt.Errorf("peer sparse-field observation: %w", err)
	}
	var projected struct {
		Data struct {
			Attributes map[string]json.RawMessage `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(peerPayload, &projected); err != nil {
		return nil, fmt.Errorf("decode peer sparse-field output: %w", err)
	}
	if _, ok := projected.Data.Attributes["title"]; !ok || len(projected.Data.Attributes) != 1 {
		return nil, fmt.Errorf("peer sparse-field output differs: %s", peerPayload)
	}
	add("JSONAPI-DEC-003", "sparse-field-family", "parsed-for-application", "applied-during-marshal")
	add("JSONAPI-DEC-006", "sparse-field-responsibility", "parsed-for-application", "applied-during-marshal")

	return observations, nil
}

func moduleVersion(path string) (string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", errors.New("read differential build information")
	}
	for _, dependency := range info.Deps {
		if dependency.Path == path {
			return dependency.Version, nil
		}
	}
	return "", fmt.Errorf("missing differential module version for %s", path)
}

func write(target io.Writer, observations []observation) error {
	writer := csv.NewWriter(target)
	writer.Comma = '\t'
	if err := writer.Write([]string{"decision_id", "case", "implementation", "version", "outcome", "classification"}); err != nil {
		return err
	}
	for _, observation := range observations {
		if err := writer.Write([]string{
			observation.decisionID,
			observation.caseName,
			observation.implementation,
			observation.version,
			observation.outcome,
			observation.classification,
		}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
