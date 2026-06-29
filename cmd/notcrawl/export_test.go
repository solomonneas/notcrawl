package main

import (
	"encoding/json"
	"testing"

	"github.com/openclaw/notcrawl/internal/adapter"
	"github.com/openclaw/notcrawl/internal/store"
)

func TestBuildNotionPageRecord(t *testing.T) {
	// 1623456789000 ms = 2021-06-12T00:13:09Z
	page := store.Page{
		ID:             "p-1",
		SpaceID:        "s-1",
		CollectionID:   "col-1",
		Title:          "Roadmap",
		URL:            "https://notion.so/p-1",
		CreatedTime:    1623456789000,
		LastEditedTime: 1623470000000,
	}
	body := blockText([]store.Block{
		{Text: "Q3 goals"},
		{Text: ""},
		{Text: "ship evidence stack"},
	})

	rec := buildNotionPageRecord(page, "Engineering", body, "9.9.9")

	if rec.Source.Kind != "notion" || rec.Source.Version != "9.9.9" {
		t.Fatalf("source = %+v", rec.Source)
	}
	if rec.Collection.ExternalID != "notion:space:s-1" || rec.Collection.Kind != "notion_space" || rec.Collection.Name != "Engineering" {
		t.Fatalf("collection = %+v", rec.Collection)
	}
	if rec.Item.ExternalID != "notion:page:p-1" || rec.Item.Kind != "page" {
		t.Fatalf("item = %+v", rec.Item)
	}
	if rec.Item.Text != "Roadmap\n\nQ3 goals\nship evidence stack" {
		t.Fatalf("text = %q", rec.Item.Text)
	}
	if rec.Item.CreatedAt != "2021-06-12T00:13:09Z" {
		t.Fatalf("created_at = %q, want 2021-06-12T00:13:09Z", rec.Item.CreatedAt)
	}
	if len(rec.Links) != 1 || rec.Links[0].URL != "https://notion.so/p-1" {
		t.Fatalf("links = %+v", rec.Links)
	}
	if rec.Actor == nil || rec.Actor.Type != "system" {
		t.Fatalf("actor = %+v", rec.Actor)
	}

	line, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := adapter.Parse(line); err != nil {
		t.Fatalf("adapter.Parse rejected emitted record: %v", err)
	}
}

func TestBuildNotionPageRecordFallbacks(t *testing.T) {
	// No space, no blocks, no time: space falls back to "default", text is the
	// title only, timestamp is empty, and the record must still validate.
	rec := buildNotionPageRecord(store.Page{ID: "p-2", Title: "Loose Page"}, "", "", "")

	if rec.Collection.ExternalID != "notion:space:default" || rec.Collection.Name != "default" {
		t.Fatalf("collection = %+v", rec.Collection)
	}
	if rec.Item.Text != "Loose Page" {
		t.Fatalf("text = %q", rec.Item.Text)
	}
	if rec.Item.CreatedAt != "" {
		t.Fatalf("created_at should be empty, got %q", rec.Item.CreatedAt)
	}
	if len(rec.Links) != 0 {
		t.Fatalf("expected no links, got %+v", rec.Links)
	}
	line, _ := json.Marshal(rec)
	if _, err := adapter.Parse(line); err != nil {
		t.Fatalf("adapter.Parse rejected fallback record: %v", err)
	}
}

func TestEpochMillisToRFC3339(t *testing.T) {
	if got := epochMillisToRFC3339(0); got != "" {
		t.Errorf("epoch 0 = %q, want empty", got)
	}
	if got := epochMillisToRFC3339(1623456789000); got != "2021-06-12T00:13:09Z" {
		t.Errorf("epoch = %q, want 2021-06-12T00:13:09Z", got)
	}
}
