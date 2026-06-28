package clipman_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	. "clipman"
)

func TestStorageRoundTrip(t *testing.T) {
	dir := t.TempDir()
	DataFile = filepath.Join(dir, "test.jl")
	t.Cleanup(ResetWriter)

	AppendEntry(Entry{Text: "first", Time: "2024-01-01"})
	AppendEntry(Entry{Text: "second", Time: "2024-01-02"})
	AppendEntry(Entry{Text: "third", Time: "2024-01-03"})
	LoadAll()

	mux := http.NewServeMux()
	mux.HandleFunc("/api", HandleAPI)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var result []Entry
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}
	if result[0].Text != "third" {
		t.Errorf("expected newest first 'third', got %q", result[0].Text)
	}
	if result[1].Text != "second" {
		t.Errorf("expected 'second', got %q", result[1].Text)
	}
	if result[2].Text != "first" {
		t.Errorf("expected 'first', got %q", result[2].Text)
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	dir := t.TempDir()
	DataFile = filepath.Join(dir, "nonexistent.jl")

	LoadAll()

	mux := http.NewServeMux()
	mux.HandleFunc("/api", HandleAPI)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var result []Entry
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 entries for nonexistent file, got %d", len(result))
	}
}
