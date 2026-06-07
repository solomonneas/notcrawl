package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultUsesCurrentNotionAPIVersion(t *testing.T) {
	cfg := Default()
	if cfg.Notion.API.Version != "2026-03-11" {
		t.Fatalf("unexpected API version: %s", cfg.Notion.API.Version)
	}
}

func TestDefaultConfiguresExperimentalNotionMCPSource(t *testing.T) {
	cfg := Default()
	if cfg.Notion.MCP.Enabled {
		t.Fatal("Notion MCP should remain opt-in for sync --source all")
	}
	if cfg.Notion.MCP.ConnectorID != defaultMCPConnectorID || cfg.Notion.MCP.MaxPages != 100 {
		t.Fatalf("unexpected MCP defaults: %+v", cfg.Notion.MCP)
	}
	if err := cfg.Resolve(); err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(cfg.Notion.MCP.AuthPath) {
		t.Fatalf("auth path was not expanded: %q", cfg.Notion.MCP.AuthPath)
	}
}
