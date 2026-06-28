package clipman_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	. "clipman"
)

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	DataFile = filepath.Join(dir, "test.jl")
	t.Cleanup(ResetWriter)

	AppendEntry(Entry{Text: "a", Time: "t1"})
	AppendEntry(Entry{Text: "b", Time: "t2"})
	AppendEntry(Entry{Text: "c", Time: "t3"})
	AppendEntry(Entry{Text: "d", Time: "t4"})
	AppendEntry(Entry{Text: "e", Time: "t5"})
	LoadAll()

	mux := http.NewServeMux()
	mux.HandleFunc("/api", HandleAPI)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	check := func(expected int) []Entry {
		resp, err := http.Get(ts.URL + "/api")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var result []Entry
		json.NewDecoder(resp.Body).Decode(&result)
		if len(result) != expected {
			t.Errorf("expected %d entries, got %d", expected, len(result))
		}
		return result
	}

	// history order: e(0) d(1) c(2) b(3) a(4)
	t.Run("delete single", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", ts.URL+"/api?id=2", nil)
		http.DefaultClient.Do(req)
		r := check(4)
		if r[0].Text != "e" || r[1].Text != "d" || r[2].Text != "b" || r[3].Text != "a" {
			t.Errorf("unexpected order after delete: %v", r)
		}
	})

	t.Run("delete batch", func(t *testing.T) {
		// restore: reload from file (c already deleted above, so history: e d b a)
		LoadAll()
		req, _ := http.NewRequest("DELETE", ts.URL+"/api?id=0&id=3", nil)
		http.DefaultClient.Do(req)
		r := check(2)
		if r[0].Text != "d" || r[1].Text != "b" {
			t.Errorf("unexpected order after batch delete: %v", r)
		}
	})
}

func TestTokenMatch(t *testing.T) {
	tests := []struct {
		text, time, query string
		want              bool
	}{
		{"hello world", "2024-01-01 12:00:00", "hello", true},
		{"hello world", "2024-01-01 12:00:00", "hello world", true},
		{"hello world", "2024-01-01 12:00:00", "hello moon", false},
		{"hello world", "2024-01-01 12:00:00", "12:00", true},
		{"hello world", "2024-01-01 12:00:00", "", true},
		{"Go语言测试", "2024-01-01", "go", true},
	}
	for _, tt := range tests {
		got := TokenMatch(tt.text, tt.time, tt.query)
		if got != tt.want {
			t.Errorf("TokenMatch(%q, %q, %q) = %v, want %v", tt.text, tt.time, tt.query, got, tt.want)
		}
	}
}

func TestAPI(t *testing.T) {
	dir := t.TempDir()
	DataFile = filepath.Join(dir, "test.jl")
	t.Cleanup(ResetWriter)

	AppendEntry(Entry{Text: "hello world", Time: "2024-01-01 12:00:00"})
	AppendEntry(Entry{Text: "foo bar", Time: "2024-01-01 13:00:00"})
	AppendEntry(Entry{Text: "clipman test", Time: "2024-01-02 10:00:00"})
	LoadAll()

	mux := http.NewServeMux()
	mux.HandleFunc("/api", HandleAPI)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Run("no query returns all", func(t *testing.T) {
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
			t.Errorf("expected 3 entries, got %d", len(result))
		}
	})

	t.Run("query filters correctly", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api?q=hello")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var result []Entry
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
		if result[0].Text != "hello world" {
			t.Errorf("expected 'hello world', got %q", result[0].Text)
		}
	})

	t.Run("query matches time field", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api?q=13:00")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var result []Entry
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
		if result[0].Text != "foo bar" {
			t.Errorf("expected 'foo bar', got %q", result[0].Text)
		}
	})

	t.Run("multi-word AND matching", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api?q=clipman+test")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var result []Entry
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
	})
}
