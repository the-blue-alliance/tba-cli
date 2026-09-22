package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// entries lists a directory, so a test can say "and nothing else".
func entries(t *testing.T, dir string) []string {
	t.Helper()
	des, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	names := make([]string, 0, len(des))
	for _, de := range des {
		names = append(names, de.Name())
	}
	return names
}

func TestWriteFileAtomicWritesTheFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := WriteFileAtomic(path, []byte("year: 2024\n"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading it back: %v", err)
	}
	if string(body) != "year: 2024\n" {
		t.Errorf("contents = %q", body)
	}
	if got := entries(t, dir); len(got) != 1 || got[0] != "config.yaml" {
		t.Errorf("directory = %v, want only the written file", got)
	}
}

func TestWriteFileAtomicReplacesWhatWasThere(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("year: 1999\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := WriteFileAtomic(path, []byte("year: 2024\n"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	body, _ := os.ReadFile(path)
	if string(body) != "year: 2024\n" {
		t.Errorf("contents = %q, want the new ones", body)
	}
	if got := entries(t, dir); len(got) != 1 {
		t.Errorf("directory = %v, want only the written file", got)
	}
}

// The mode is the point of taking perm at all: os.CreateTemp hands out 0600,
// which is right for a config file and wrong for an export.
func TestWriteFileAtomicAppliesThePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not carry Unix file modes")
	}
	dir := t.TempDir()
	for _, perm := range []os.FileMode{0o600, 0o644} {
		path := filepath.Join(dir, "f")
		if err := WriteFileAtomic(path, []byte("x"), perm); err != nil {
			t.Fatalf("WriteFileAtomic: %v", err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != perm {
			t.Errorf("mode = %04o, want %04o", got, perm)
		}
	}
}

// A failed write must not leave a half-finished file lying about under a name
// nobody will ever look at again.
func TestWriteFileAtomicLeavesNothingBehindWhenTheRenameFails(t *testing.T) {
	dir := t.TempDir()
	boom := errors.New("rename refused")
	original := renameFile
	renameFile = func(string, string) error { return boom }
	t.Cleanup(func() { renameFile = original })

	err := WriteFileAtomic(filepath.Join(dir, "config.yaml"), []byte("year: 2024\n"), 0o600)
	if !errors.Is(err, boom) {
		t.Fatalf("error = %v, want the rename failure", err)
	}
	if got := entries(t, dir); len(got) != 0 {
		t.Errorf("directory = %v, want nothing left behind", got)
	}
}

func TestWriteFileAtomicReportsAMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope", "config.yaml")
	if err := WriteFileAtomic(path, []byte("x"), 0o600); err == nil {
		t.Fatal("want an error when the directory does not exist")
	}
}

func TestWriteTempReturnsAFileTheCallerOwns(t *testing.T) {
	dir := t.TempDir()
	name, err := WriteTemp(dir, ".staged-*.tmp", []byte("body\n"), 0o644)
	if err != nil {
		t.Fatalf("WriteTemp: %v", err)
	}
	if filepath.Dir(name) != dir {
		t.Errorf("staged in %s, want %s: a rename across filesystems is not atomic", filepath.Dir(name), dir)
	}
	base := filepath.Base(name)
	if !strings.HasPrefix(base, ".staged-") || !strings.HasSuffix(base, ".tmp") {
		t.Errorf("name = %q, want the caller's pattern", base)
	}
	body, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "body\n" {
		t.Errorf("contents = %q", body)
	}
	// It is the caller's now: WriteTemp does not rename or remove it.
	if got := entries(t, dir); len(got) != 1 {
		t.Errorf("directory = %v, want the staged file", got)
	}
}

func TestWriteTempReportsAMissingDirectory(t *testing.T) {
	if _, err := WriteTemp(filepath.Join(t.TempDir(), "nope"), "*.tmp", []byte("x"), 0o644); err == nil {
		t.Fatal("want an error when the directory does not exist")
	}
}
