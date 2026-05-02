package store_test

import (
	"testing"

	"github.com/szdytom/tb/internal/buffer"
	"github.com/szdytom/tb/internal/store"
)

func TestSearchLiteral(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	makeBuffer(t, repo, "the quick brown fox", "a", nil)
	makeBuffer(t, repo, "jumps over the lazy dog", "b", nil)
	makeBuffer(t, repo, "nothing matches here", "c", nil)

	results, err := repo.Search("quick brown", store.SearchModeLiteral)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Buffer.Label != "a" {
		t.Errorf("expected label 'a', got %q", results[0].Buffer.Label)
	}
}

func TestSearchLiteralCaseInsensitive(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	makeBuffer(t, repo, "Hello World", "", nil)

	results, err := repo.Search("hello", store.SearchModeLiteral)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestSearchLiteralMultiMatch(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	makeBuffer(t, repo, "abc def ghi", "", nil)
	makeBuffer(t, repo, "def ghi jkl", "", nil)
	makeBuffer(t, repo, "ghi jkl mno", "", nil)

	results, err := repo.Search("def", store.SearchModeLiteral)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'def', got %d", len(results))
	}
}

func TestSearchLiteralNoMatch(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	makeBuffer(t, repo, "abc def", "", nil)

	results, err := repo.Search("xyz", store.SearchModeLiteral)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestSearchRegex(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	makeBuffer(t, repo, "error: timeout occurred", "", nil)
	makeBuffer(t, repo, "warning: low memory", "", nil)
	makeBuffer(t, repo, "error: something else", "", nil)

	results, err := repo.Search("error.*time", store.SearchModeRegex)
	if err != nil {
		t.Fatalf("regex search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 regex match, got %d", len(results))
	}
}

func TestSearchRegexNoMatch(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	makeBuffer(t, repo, "hello world", "", nil)

	results, err := repo.Search("^xyz$", store.SearchModeRegex)
	if err != nil {
		t.Fatalf("regex search: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestSearchRegexInvalid(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	makeBuffer(t, repo, "hello", "", nil)

	_, err := repo.Search("[invalid", store.SearchModeRegex)
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestSearchSnippet(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	// Content long enough to trigger snippet truncation.
	content := "aaa bbb ccc ddd eee fff ggg hhh iii jjj kkk lll mmm nnn ooo ppp"
	makeBuffer(t, repo, content, "", nil)

	results, err := repo.Search("hhh", store.SearchModeLiteral)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	snippet := results[0].Snippet
	if len(snippet) == 0 {
		t.Fatal("expected non-empty snippet")
	}

	if len(snippet) >= len(content) {
		t.Logf("snippet covers full content (length %d)", len(snippet))
	}
}

func TestSearchIgnoresTrashed(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	id, _ := repo.Insert(buffer.NewBuffer("searchable content", "", nil))
	repo.SoftDelete(id, 0)

	results, err := repo.Search("searchable", store.SearchModeLiteral)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 results from trashed buffer, got %d", len(results))
	}
}

func TestSearchFuzzy(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	makeBuffer(t, repo, "hello world", "", nil)
	makeBuffer(t, repo, "something else", "", nil)

	// "hld" matches "hello world" but not "something else"
	results, err := repo.Search("hld", store.SearchModeFuzzy)
	if err != nil {
		t.Fatalf("fuzzy search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 fuzzy match, got %d", len(results))
	}
}

func TestSearchFuzzyCaseInsensitive(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	makeBuffer(t, repo, "Hello World", "", nil)

	results, err := repo.Search("hw", store.SearchModeFuzzy)
	if err != nil {
		t.Fatalf("fuzzy search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 fuzzy match, got %d", len(results))
	}
}

func TestSearchFuzzyNoMatch(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	makeBuffer(t, repo, "hello world", "", nil)

	results, err := repo.Search("xyz", store.SearchModeFuzzy)
	if err != nil {
		t.Fatalf("fuzzy search: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 fuzzy results, got %d", len(results))
	}
}

func makeBuffer(t *testing.T, repo *store.Repository, content, label string, tags []string) int64 {
	t.Helper()

	id, err := repo.Insert(buffer.NewBuffer(content, label, tags))
	if err != nil {
		t.Fatalf("makeBuffer: %v", err)
	}

	return id
}
