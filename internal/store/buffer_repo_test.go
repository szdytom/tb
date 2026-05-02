package store_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/szdytom/tb/internal/buffer"
	"github.com/szdytom/tb/internal/store"
)

func openTestDB(t *testing.T) *store.DB {
	t.Helper()

	dir, err := os.MkdirTemp("", "tmpbuffer-test-*")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { os.RemoveAll(dir) })

	db, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	t.Cleanup(func() { db.Close() })

	return db
}

func TestInsert(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	buf := buffer.NewBuffer("hello world", "test-label", []string{"tag1", "tag2"})

	id, err := repo.Insert(buf)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	if id == 0 {
		t.Fatal("expected non-zero ID")
	}

	if buf.ID != 0 {
		t.Fatal("should not modify the original buffer's ID")
	}
}

func TestGet(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	// Insert a buffer and retrieve it.
	orig := buffer.NewBuffer("content", "label", nil)

	id, err := repo.Insert(orig)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := repo.Get(id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.ID != id {
		t.Errorf("ID = %d, want %d", got.ID, id)
	}

	if got.Label != "label" {
		t.Errorf("Label = %q, want %q", got.Label, "label")
	}

	if got.Content != "content" {
		t.Errorf("Content = %q, want %q", got.Content, "content")
	}

	if got.Metadata.ByteCount != len("content") {
		t.Errorf("ByteCount = %d, want %d", got.Metadata.ByteCount, len("content"))
	}

	if got.TrashStatus != buffer.TrashStatusActive {
		t.Errorf("TrashStatus = %d, want %d", got.TrashStatus, buffer.TrashStatusActive)
	}
}

func TestGetNotFound(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	_, err := repo.Get(999)
	if err == nil {
		t.Fatal("expected error for non-existent buffer")
	}
}

func TestList(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	// Insert three buffers.
	for i := range 3 {
		buf := buffer.NewBuffer("content", "", nil)
		// Stagger creation times.
		time.Sleep(time.Millisecond)

		if _, err := repo.Insert(buf); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	bufs, err := repo.List(store.ListFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(bufs) != 3 {
		t.Fatalf("got %d buffers, want 3", len(bufs))
	}

	// Most recent first by default.
	if bufs[0].ID > bufs[1].ID && bufs[0].ID > bufs[2].ID {
		t.Log("list is sorted most-recent-first (DESC)")
	}
}

func TestListFilterKeyword(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	repo.Insert(buffer.NewBuffer("apple banana", "fruit", nil))
	repo.Insert(buffer.NewBuffer("banana cherry", "fruit", nil))
	repo.Insert(buffer.NewBuffer("date", "fruit", nil))

	bufs, err := repo.List(store.ListFilter{Keyword: "banana"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(bufs) != 2 {
		t.Fatalf("got %d results for 'banana', want 2", len(bufs))
	}
}

func TestListFilterTimeRange(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	old := buffer.NewBuffer("old", "", nil)
	old.CreatedAt = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	old.UpdatedAt = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := repo.Insert(old); err != nil {
		t.Fatal(err)
	}

	buf := buffer.NewBuffer("new", "", nil)
	if _, err := repo.Insert(buf); err != nil {
		t.Fatal(err)
	}

	since := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	bufs, err := repo.List(store.ListFilter{Since: &since})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(bufs) != 1 {
		t.Fatalf("expected 1 buffer after 2025, got %d", len(bufs))
	}
}

func TestListSortAndPagination(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	for _, label := range []string{"c", "b", "a"} {
		repo.Insert(buffer.NewBuffer("content", label, nil))
	}

	// Sort ascending by label (SortAsc = true reverses the default DESC order).
	bufs, err := repo.List(store.ListFilter{SortBy: store.SortByLabel, SortAsc: true})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(bufs) < 3 {
		t.Fatal("expected at least 3 buffers")
	}

	if bufs[0].Label != "a" || bufs[2].Label != "c" {
		t.Errorf("expected a→c sorted ascending, got %q→%q", bufs[0].Label, bufs[2].Label)
	}

	// Limit + offset.
	bufs, err = repo.List(store.ListFilter{SortBy: store.SortByLabel, SortAsc: true, Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(bufs) != 2 {
		t.Fatalf("expected 2 buffers with limit, got %d", len(bufs))
	}
}

func TestUpdateContent(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	id, _ := repo.Insert(buffer.NewBuffer("original", "", nil))

	if err := repo.UpdateContent(id, "modified content"); err != nil {
		t.Fatalf("update content: %v", err)
	}

	got, _ := repo.Get(id)
	if got.Content != "modified content" {
		t.Errorf("Content = %q, want %q", got.Content, "modified content")
	}

	if got.Metadata.ByteCount != len("modified content") {
		t.Errorf("ByteCount = %d, want %d", got.Metadata.ByteCount, len("modified content"))
	}
}

func TestUpdateLabelAndTags(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	id, _ := repo.Insert(buffer.NewBuffer("content", "old", []string{"a"}))

	if err := repo.UpdateLabel(id, "new"); err != nil {
		t.Fatal(err)
	}

	if err := repo.UpdateTags(id, []string{"x", "y"}); err != nil {
		t.Fatal(err)
	}

	got, _ := repo.Get(id)
	if got.Label != "new" {
		t.Errorf("Label = %q, want %q", got.Label, "new")
	}

	if len(got.Tags) != 2 || got.Tags[0] != "x" {
		t.Errorf("Tags = %v, want [x y]", got.Tags)
	}
}

func TestSoftDeleteAndRestore(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	id, _ := repo.Insert(buffer.NewBuffer("content", "", nil))

	if err := repo.SoftDelete(id, 0); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	// Should not appear in list.
	bufs, _ := repo.List(store.ListFilter{})
	if len(bufs) != 0 {
		t.Errorf("expected 0 active buffers after delete, got %d", len(bufs))
	}

	// Should appear in trash.
	trash, err := repo.ListTrash()
	if err != nil {
		t.Fatalf("list trash: %v", err)
	}

	if len(trash) != 1 {
		t.Fatalf("expected 1 trashed buffer, got %d", len(trash))
	}

	if trash[0].ID != id {
		t.Errorf("trashed buffer ID = %d, want %d", trash[0].ID, id)
	}

	// Restore.
	if err := repo.RestoreFromTrash(id); err != nil {
		t.Fatalf("restore: %v", err)
	}

	bufs, _ = repo.List(store.ListFilter{})
	if len(bufs) != 1 {
		t.Errorf("expected 1 buffer after restore, got %d", len(bufs))
	}
}

func TestPermanentlyDelete(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	id, _ := repo.Insert(buffer.NewBuffer("content", "", nil))

	if err := repo.PermanentlyDelete(id); err != nil {
		t.Fatalf("permanently delete: %v", err)
	}

	_, err := repo.Get(id)
	if err == nil {
		t.Fatal("expected error for deleted buffer")
	}

	// Double-delete should fail.
	if err := repo.PermanentlyDelete(id); err == nil {
		t.Fatal("expected error when deleting non-existent buffer")
	}
}

func TestCount(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	n, _ := repo.Count()
	if n != 0 {
		t.Errorf("initial count = %d, want 0", n)
	}

	repo.Insert(buffer.NewBuffer("a", "", nil))
	repo.Insert(buffer.NewBuffer("b", "", nil))

	n, _ = repo.Count()
	if n != 2 {
		t.Errorf("count = %d, want 2", n)
	}
}

func TestDeleteExpiredTrash(t *testing.T) {
	db := openTestDB(t)
	repo := store.NewRepository(db)

	id, _ := repo.Insert(buffer.NewBuffer("content", "", nil))

	// Soft-delete with zero TTL (uses default 24h) — won't be expired.
	repo.SoftDelete(id, 0)

	n, err := repo.DeleteExpiredTrash()
	if err != nil {
		t.Fatalf("delete expired trash: %v", err)
	}

	if n != 0 {
		t.Errorf("expected 0 expired entries, got %d", n)
	}

	// Soft-delete with a negative TTL to force expiration.
	repo.Insert(buffer.NewBuffer("expired", "", nil))
	id2, _ := repo.Insert(buffer.NewBuffer("expired", "", nil))
	repo.SoftDelete(id2, -1*time.Hour)

	n, _ = repo.DeleteExpiredTrash()
	if n != 1 {
		t.Errorf("expected 1 expired entry removed, got %d", n)
	}
}

func TestWALMode(t *testing.T) {
	db := openTestDB(t)

	var mode string

	err := db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("read journal mode: %v", err)
	}

	if mode != "wal" && mode != "WAL" {
		t.Errorf("journal mode = %q, want WAL", mode)
	}
}
