package toonkotsuscraper

import (
	"strings"
	"testing"
)

func TestGetWorkMetadata(t *testing.T) {
	server := newSeriesServer(t, "bm0ssx", `{
		"series": {
			"authors": ["AKUTAMI Gege"],
			"artists": null,
			"description": " Although Yuji Itadori looks like your average teenager... ",
			"cover": "/cdn/c/bibd7/muxe4yt6/460.webp",
			"tags": ["Action", " ", "Supernatural"],
			"originalLanguage": "ja",
			"year": 2018,
			"status": "Complete"
		},
		"chapters": []
	}`)

	metadata, err := GetWorkMetadata(
		server.URL,
		"bm0ssx",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("GetWorkMetadata() error: %v", err)
	}

	if metadata.Author != "AKUTAMI Gege" {
		t.Errorf("author = %q", metadata.Author)
	}

	if metadata.Artist != "" {
		t.Errorf("artist = %q, want empty", metadata.Artist)
	}

	if metadata.Description != "Although Yuji Itadori looks like your average teenager..." {
		t.Errorf("description = %q", metadata.Description)
	}

	if metadata.CoverURL != server.URL+"/cdn/c/bibd7/muxe4yt6/460.webp" {
		t.Errorf("cover = %q", metadata.CoverURL)
	}

	if len(metadata.Tags) != 2 || metadata.Tags[0] != "Action" {
		t.Errorf("tags = %+v", metadata.Tags)
	}

	if metadata.OriginalLanguage != "ja" {
		t.Errorf("original language = %q", metadata.OriginalLanguage)
	}

	if metadata.Year != 2018 {
		t.Errorf("year = %d", metadata.Year)
	}

	if !metadata.Known() {
		t.Error("expected the metadata to be recognized as known")
	}
}

func TestGetWorkMetadataJoinsSeveralPeople(t *testing.T) {
	server := newSeriesServer(t, "pair", `{
		"series": {
			"authors": [" Writer ", "", "Second Author"],
			"artists": ["Penciller"]
		}
	}`)

	metadata, err := GetWorkMetadata(
		server.URL,
		"pair",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("GetWorkMetadata() error: %v", err)
	}

	if metadata.Author != "Writer, Second Author" {
		t.Errorf("author = %q", metadata.Author)
	}

	if metadata.Artist != "Penciller" {
		t.Errorf("artist = %q", metadata.Artist)
	}
}

func TestGetWorkMetadataToleratesEmptyPayload(t *testing.T) {
	server := newSeriesServer(t, "blank", `{"series":{},"chapters":[]}`)

	metadata, err := GetWorkMetadata(
		server.URL,
		"blank",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("GetWorkMetadata() error: %v", err)
	}

	if metadata.Known() {
		t.Errorf("expected empty metadata, got %+v", metadata)
	}
}

func TestGetWorkMetadataMalformedPayload(t *testing.T) {
	server := newSeriesServer(t, "bad", `{"series":{"authors":"not-an-array"}}`)

	_, err := GetWorkMetadata(server.URL, "bad", testDiscardLogger())
	if err == nil {
		t.Fatal("expected a decode error")
	}

	if !strings.Contains(err.Error(), "failed to decode toonkotsu series metadata bad") {
		t.Errorf("expected the decode error, got %v", err)
	}
}
