package factorio

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestListSavesInDirOnlyReturnsDirectZipFilesNewestFirst(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	writeTestFile(t, filepath.Join(dir, "old.zip"), base)
	writeTestFile(t, filepath.Join(dir, "same-b.zip"), base.Add(time.Minute))
	writeTestFile(t, filepath.Join(dir, "same-a.zip"), base.Add(time.Minute))
	writeTestFile(t, filepath.Join(dir, "new.zip"), base.Add(2*time.Minute))
	writeTestFile(t, filepath.Join(dir, "notes.txt"), base.Add(3*time.Minute))
	writeTestFile(t, filepath.Join(dir, "nested", "nested.zip"), base.Add(4*time.Minute))
	writeTestFile(t, filepath.Join(dir, "server-2", "other-server.zip"), base.Add(5*time.Minute))

	saves, err := ListSavesInDir(dir)
	if err != nil {
		t.Fatalf("ListSavesInDir returned error: %s", err)
	}

	expected := []string{"new.zip", "same-a.zip", "same-b.zip", "old.zip"}
	if got := saveNames(saves); !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected saves %v, got %v", expected, got)
	}
}

func TestListSavesWithLatestInDirUsesOnlyThatServerDirectory(t *testing.T) {
	serverOne := t.TempDir()
	serverTwo := t.TempDir()
	base := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)

	writeTestFile(t, filepath.Join(serverOne, "server-one-old.zip"), base)
	writeTestFile(t, filepath.Join(serverOne, "server-one-new.zip"), base.Add(time.Minute))
	writeTestFile(t, filepath.Join(serverTwo, "server-two-newer.zip"), base.Add(2*time.Minute))

	saves, err := ListSavesWithLatestInDir(serverOne)
	if err != nil {
		t.Fatalf("ListSavesWithLatestInDir returned error: %s", err)
	}

	expected := []string{"Load Latest (server-one-new.zip)", "server-one-new.zip", "server-one-old.zip"}
	if got := saveNames(saves); !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected saves %v, got %v", expected, got)
	}
}

func TestSaveNameValidationAndCreateNormalization(t *testing.T) {
	dir := t.TempDir()

	name, err := ValidateSaveName(" factory.zip ")
	if err != nil {
		t.Fatalf("ValidateSaveName returned error: %s", err)
	}
	if name != "factory.zip" {
		t.Fatalf("expected trimmed name factory.zip, got %q", name)
	}

	path, name, err := SavePathForCreateInDir(dir, "new-map")
	if err != nil {
		t.Fatalf("SavePathForCreateInDir returned error: %s", err)
	}
	if name != "new-map.zip" {
		t.Fatalf("expected create name new-map.zip, got %q", name)
	}
	if path != filepath.Join(dir, "new-map.zip") {
		t.Fatalf("expected create path in saves dir, got %q", path)
	}

	invalidNames := []string{
		"",
		"../escape.zip",
		"nested/escape.zip",
		`nested\escape.zip`,
		filepath.Join(dir, "escape.zip"),
		"Load Latest (factory.zip)",
		"factory.txt",
		".zip",
	}
	for _, invalid := range invalidNames {
		if _, err := ValidateSaveName(invalid); err == nil {
			t.Fatalf("expected %q to be invalid", invalid)
		}
	}

	if _, _, err := SavePathForCreateInDir(dir, ""); err == nil {
		t.Fatalf("expected blank create name to be invalid")
	}
}

func TestFindAndRemoveSaveRejectInvalidOrNestedNames(t *testing.T) {
	dir := t.TempDir()
	outsideDir := t.TempDir()
	outside := filepath.Join(outsideDir, "outside.zip")

	writeTestFile(t, filepath.Join(dir, "good.zip"), time.Now())
	writeTestFile(t, filepath.Join(dir, "nested", "nested.zip"), time.Now())
	writeTestFile(t, outside, time.Now())

	save, err := FindSaveInDir(dir, "good.zip")
	if err != nil {
		t.Fatalf("FindSaveInDir returned error: %s", err)
	}
	if save.Name != "good.zip" {
		t.Fatalf("expected good.zip, got %q", save.Name)
	}

	if _, err := FindSaveInDir(dir, "nested.zip"); err == nil {
		t.Fatalf("expected nested save not to be found")
	}
	if _, err := FindSaveInDir(dir, "../outside.zip"); err == nil {
		t.Fatalf("expected traversal save name to be rejected")
	}

	traversal := Save{Name: "../outside.zip"}
	if err := traversal.RemoveFromDir(dir); err == nil {
		t.Fatalf("expected traversal remove to be rejected")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside save should not have been touched: %s", err)
	}

	if err := save.RemoveFromDir(dir); err != nil {
		t.Fatalf("RemoveFromDir returned error: %s", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "good.zip")); !os.IsNotExist(err) {
		t.Fatalf("expected good.zip to be removed, stat err: %v", err)
	}
}

func writeTestFile(t *testing.T, path string, modTime time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("creating test dir: %s", err)
	}
	if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
		t.Fatalf("writing test file: %s", err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("setting test file time: %s", err)
	}
}

func saveNames(saves []Save) []string {
	names := make([]string, 0, len(saves))
	for _, save := range saves {
		names = append(names, save.Name)
	}
	return names
}
