package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanFindsDevAndLargeFiles(t *testing.T) {
	tmp := t.TempDir()

	nm := filepath.Join(tmp, "proj", "node_modules")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, "pkg.js"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	gitDir := filepath.Join(tmp, "proj", ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "pack"), make([]byte, 8192), 0o644); err != nil {
		t.Fatal(err)
	}

	bigFile := filepath.Join(tmp, "proj", "dump.iso")
	if err := os.WriteFile(bigFile, make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	wslDisk := filepath.Join(tmp, "proj", "ext4.vhdx")
	if err := os.WriteFile(wslDisk, make([]byte, 8192), 0o644); err != nil {
		t.Fatal(err)
	}

	strayDist := filepath.Join(tmp, "other-app", "dist")
	if err := os.MkdirAll(strayDist, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(strayDist, "app.bin"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	winSys := filepath.Join(tmp, "Windows", "System32")
	if err := os.MkdirAll(winSys, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(winSys, "sys.dll"), make([]byte, 8192), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := scan(tmp, false, 4000)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d: %+v", len(items), items)
	}
	if items[0].Path != nm || items[0].Category != "dev" || items[0].Size != 2048 {
		t.Errorf("unexpected first item: %+v", items[0])
	}
	if items[1].Path != bigFile || items[1].Category != "large" || items[1].Size != 4096 {
		t.Errorf("unexpected second item: %+v", items[1])
	}

	cacheDir := filepath.Join(tmp, "Temp")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "tmp.bin"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "desktop.ini"), []byte("icon"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := dirSize(cacheDir); got != 1024 {
		t.Fatalf("expected dirSize=1024 ignoring desktop.ini, got %d", got)
	}
	freed, rem := cleanItem(Item{Path: cacheDir, Size: 1024, Category: "cache"})
	if freed != 1024 || rem != 0 {
		t.Fatalf("expected freed=1024 rem=0, got freed=%d rem=%d", freed, rem)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "desktop.ini")); err != nil {
		t.Fatalf("expected desktop.ini to remain intact: %v", err)
	}
}
