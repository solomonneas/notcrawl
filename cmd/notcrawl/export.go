package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/openclaw/notcrawl/internal/adapter"
	"github.com/openclaw/notcrawl/internal/config"
	"github.com/openclaw/notcrawl/internal/store"
)

// runExport routes `notcrawl export <subcommand>`. Today the only target is the
// miseledger.adapter.v1 JSONL contract consumed by MiseLedger.
func runExport(ctx context.Context, stdout, stderr io.Writer, cfg config.Config, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: notcrawl export adapter [--limit N] [--out <file|->]")
	}
	switch args[0] {
	case "adapter":
		return runExportAdapter(ctx, stdout, stderr, cfg, args[1:])
	default:
		return fmt.Errorf("usage: notcrawl export adapter [--limit N] [--out <file|->]")
	}
}

// runExportAdapter walks the local archive and emits one miseledger.adapter.v1
// JSON record per Notion page (its blocks concatenated into the item text), so
// the common pipe is:
//
//	notcrawl export adapter | miseledger crawl adapter -
//
// The progress summary goes to stderr to keep stdout a clean JSONL stream.
func runExportAdapter(ctx context.Context, stdout, stderr io.Writer, cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("export adapter", flag.ContinueOnError)
	fs.SetOutput(stderr)
	limit := fs.Int("limit", 0, "maximum pages to emit (0 = all)")
	outPath := fs.String("out", "-", "output file or - for stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer st.Close()

	spaceNames, err := st.SpaceNames(ctx)
	if err != nil {
		return err
	}
	pages, err := st.Pages(ctx)
	if err != nil {
		return err
	}

	out := stdout
	if *outPath != "-" {
		f, err := os.Create(*outPath)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		out = f
	}

	enc := json.NewEncoder(out)
	count := 0
	for _, page := range pages {
		if *limit > 0 && count >= *limit {
			break
		}
		blocks, err := st.PageBlocks(ctx, page.ID)
		if err != nil {
			return err
		}
		rec := buildNotionPageRecord(page, spaceNames[page.SpaceID], blockText(blocks), version)
		if err := enc.Encode(rec); err != nil {
			return err
		}
		count++
	}
	fmt.Fprintf(stderr, "exported %d notion page(s) to miseledger.adapter.v1\n", count)
	return nil
}

// blockText concatenates the non-empty text of a page's blocks in store order,
// giving the page item a single searchable body.
func blockText(blocks []store.Block) string {
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if t := strings.TrimSpace(b.Text); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "\n")
}

// buildNotionPageRecord maps one archived Notion page onto the adapter
// contract: the page's space is the collection, the page is the item (its title
// plus block text), and the workspace is a system actor. It is a pure function
// so the mapping is unit testable.
func buildNotionPageRecord(page store.Page, spaceName, body, sourceVersion string) adapter.Record {
	spaceID := page.SpaceID
	if strings.TrimSpace(spaceID) == "" {
		spaceID = "default"
	}
	if strings.TrimSpace(spaceName) == "" {
		spaceName = spaceID
	}

	title := strings.TrimSpace(page.Title)
	text := title
	if body != "" {
		if text != "" {
			text += "\n\n" + body
		} else {
			text = body
		}
	}

	itemMeta := map[string]any{"space_id": page.SpaceID}
	if page.CollectionID != "" {
		itemMeta["collection_id"] = page.CollectionID
	}
	if page.URL != "" {
		itemMeta["url"] = page.URL
	}
	if title != "" {
		itemMeta["title"] = title
	}

	links := []adapter.Link{}
	if page.URL != "" {
		links = append(links, adapter.Link{URL: page.URL, Text: title})
	}

	rawHash := hashHex([]byte(page.ID + "\x1f" + text))

	return adapter.Record{
		Schema: adapter.SchemaV1,
		Source: adapter.Source{Kind: "notion", Name: "notion", Version: sourceVersion},
		Collection: adapter.Collection{
			ExternalID: "notion:space:" + spaceID,
			Kind:       "notion_space",
			Name:       spaceName,
			Metadata:   metadataJSON(map[string]any{"space_id": spaceID}),
		},
		Item: adapter.Item{
			ExternalID: "notion:page:" + page.ID,
			Kind:       "page",
			CreatedAt:  epochMillisToRFC3339(page.CreatedTime),
			UpdatedAt:  epochMillisToRFC3339(page.LastEditedTime),
			Text:       text,
			Tags:       []string{"notion", "page"},
			Metadata:   metadataJSON(itemMeta),
		},
		Actor: &adapter.Actor{
			ExternalID: "notion:system:pages",
			Type:       "system",
			Name:       "Notion",
		},
		Artifacts: []adapter.Artifact{},
		Links:     links,
		Relations: []adapter.Relation{},
		Raw: adapter.RawRef{
			Format: "notion/page",
			Hash:   "sha256:" + rawHash,
			Path:   page.ID,
		},
	}
}

// epochMillisToRFC3339 converts a Notion millisecond epoch timestamp into an
// RFC3339Nano UTC string. A non-positive value (unknown time) yields "".
func epochMillisToRFC3339(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339Nano)
}

func metadataJSON(v map[string]any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func hashHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
