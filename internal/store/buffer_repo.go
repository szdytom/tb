package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/szdytom/tb/internal/buffer"
)

// SortField specifies a column to sort by.
type SortField string

const (
	SortByUpdatedAt SortField = "updated_at"
	SortByCreatedAt SortField = "created_at"
	SortByLabel     SortField = "label"
	SortByID        SortField = "id"
)

// ListFilter specifies filter, sort, and pagination options for List queries.
type ListFilter struct {
	Keyword string
	Since   *time.Time
	Until   *time.Time
	Limit   int
	Offset  int
	SortBy  SortField
	SortAsc bool
}

// Repository provides CRUD operations for buffers backed by SQLite.
type Repository struct {
	db *DB
}

// NewRepository creates a Repository wrapping the given DB.
func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

// Insert creates a new buffer and returns its assigned ID.
func (r *Repository) Insert(buf *buffer.Buffer) (int64, error) {
	res, err := r.db.Exec(`
		INSERT INTO buffers (label, content, line_count, byte_count, tags, created_at, updated_at, trash_status, trashed_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		buf.Label, buf.Content, buf.Metadata.LineCount, buf.Metadata.ByteCount,
		joinTags(buf.Tags), buf.CreatedAt.Format(time.RFC3339), buf.UpdatedAt.Format(time.RFC3339),
		buf.TrashStatus, nullTime(buf.TrashedAt), nullTime(buf.ExpiresAt),
	)
	if err != nil {
		return 0, fmt.Errorf("insert buffer: %w", err)
	}
	return res.LastInsertId()
}

// Get retrieves a single buffer by ID.
func (r *Repository) Get(id int64) (*buffer.Buffer, error) {
	row := r.db.QueryRow(`
		SELECT id, label, content, line_count, byte_count, tags, created_at, updated_at, trash_status, trashed_at, expires_at
		FROM buffers WHERE id = ?`, id)
	return scanBuffer(row)
}

// buildListFilter builds the WHERE + ORDER BY + LIMIT/OFFSET clause and its args.
// Used by both List and ListBufferSummaries.
func (r *Repository) buildListFilter(filter ListFilter) (string, []interface{}) {
	var clauses []string
	var args []interface{}

	clauses = append(clauses, "trash_status = ?")
	args = append(args, buffer.TrashStatusActive)

	if filter.Keyword != "" {
		clauses = append(clauses, "(content LIKE '%' || ? || '%' OR label LIKE '%' || ? || '%')")
		args = append(args, filter.Keyword, filter.Keyword)
	}
	if filter.Since != nil {
		clauses = append(clauses, "updated_at >= ?")
		args = append(args, filter.Since.Format(time.RFC3339))
	}
	if filter.Until != nil {
		clauses = append(clauses, "updated_at <= ?")
		args = append(args, filter.Until.Format(time.RFC3339))
	}

	query := "WHERE " + strings.Join(clauses, " AND ")

	// Validate sort field, default to updated_at
	sortBy := SortByUpdatedAt
	switch filter.SortBy {
	case SortByCreatedAt, SortByLabel, SortByID, SortByUpdatedAt:
		sortBy = filter.SortBy
	}
	order := "DESC"
	if filter.SortAsc {
		order = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, order)

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	return query, args
}

// List returns active (non-trashed) buffers matching the filter, sorted and paginated.
func (r *Repository) List(filter ListFilter) ([]*buffer.Buffer, error) {
	filterSQL, args := r.buildListFilter(filter)
	query := `
		SELECT id, label, content, line_count, byte_count, tags, created_at, updated_at, trash_status, trashed_at, expires_at
		FROM buffers ` + filterSQL

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list buffers: %w", err)
	}
	defer rows.Close()

	var bufs []*buffer.Buffer
	for rows.Next() {
		buf, err := scanBuffer(rows)
		if err != nil {
			return nil, err
		}
		bufs = append(bufs, buf)
	}
	return bufs, rows.Err()
}

// ListBufferSummaries returns light-weight summaries (no full content) for active buffers.
func (r *Repository) ListBufferSummaries(filter ListFilter) ([]buffer.BufferSummary, error) {
	filterSQL, args := r.buildListFilter(filter)
	query := `
		SELECT id, label, SUBSTR(content, 1, 80) AS preview,
		       line_count, byte_count, tags, created_at, updated_at
		FROM buffers ` + filterSQL

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list buffer summaries: %w", err)
	}
	defer rows.Close()

	var summaries []buffer.BufferSummary
	for rows.Next() {
		var (
			id, lineCount, byteCount int
			label, preview, tags     string
			createdAt, updatedAt     string
		)
		if err := rows.Scan(&id, &label, &preview, &lineCount, &byteCount, &tags, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		s := buffer.BufferSummary{
			ID:        int64(id),
			Label:     label,
			Preview:   preview,
			LineCount: lineCount,
			ByteCount: byteCount,
			Tags:      splitTags(tags),
		}
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			s.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			s.UpdatedAt = t
		}
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if summaries == nil {
		summaries = []buffer.BufferSummary{}
	}
	return summaries, nil
}

// UpdateContent replaces the content of a buffer and recomputes metadata.
func (r *Repository) UpdateContent(id int64, content string) error {
	meta := buffer.ComputeMetadata(content)
	now := time.Now().Format(time.RFC3339)
	res, err := r.db.Exec(
		`UPDATE buffers SET content = ?, line_count = ?, byte_count = ?, updated_at = ? WHERE id = ?`,
		content, meta.LineCount, meta.ByteCount, now, id)
	if err != nil {
		return fmt.Errorf("update buffer %d content: %w", id, err)
	}
	return checkRowsAffected(res, id)
}

// UpdateLabel changes the label of a buffer.
func (r *Repository) UpdateLabel(id int64, label string) error {
	res, err := r.db.Exec(
		`UPDATE buffers SET label = ?, updated_at = ? WHERE id = ?`,
		label, time.Now().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, id)
}

// UpdateTags replaces the tags of a buffer.
func (r *Repository) UpdateTags(id int64, tags []string) error {
	res, err := r.db.Exec(
		`UPDATE buffers SET tags = ?, updated_at = ? WHERE id = ?`,
		joinTags(tags), time.Now().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, id)
}

// SoftDelete moves a buffer to the trash.
func (r *Repository) SoftDelete(id int64, ttl time.Duration) error {
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	now := time.Now()
	res, err := r.db.Exec(
		`UPDATE buffers SET trash_status = ?, trashed_at = ?, expires_at = ?, updated_at = ? WHERE id = ? AND trash_status = ?`,
		buffer.TrashStatusTrashed, now.Format(time.RFC3339), now.Add(ttl).Format(time.RFC3339),
		now.Format(time.RFC3339), id, buffer.TrashStatusActive)
	if err != nil {
		return fmt.Errorf("soft-delete buffer %d: %w", id, err)
	}
	return checkRowsAffected(res, id)
}

// PermanentlyDelete removes a buffer from the database entirely.
func (r *Repository) PermanentlyDelete(id int64) error {
	res, err := r.db.Exec(`DELETE FROM buffers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("permanently delete buffer %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListTrash returns trashed buffers ordered by when they were trashed.
func (r *Repository) ListTrash() ([]*buffer.Buffer, error) {
	rows, err := r.db.Query(`
		SELECT id, label, content, line_count, byte_count, tags, created_at, updated_at, trash_status, trashed_at, expires_at
		FROM buffers WHERE trash_status = ? ORDER BY trashed_at DESC`, buffer.TrashStatusTrashed)
	if err != nil {
		return nil, fmt.Errorf("list trash: %w", err)
	}
	defer rows.Close()

	var bufs []*buffer.Buffer
	for rows.Next() {
		buf, err := scanBuffer(rows)
		if err != nil {
			return nil, err
		}
		bufs = append(bufs, buf)
	}
	return bufs, rows.Err()
}

// RestoreFromTrash moves a buffer from trash back to active.
func (r *Repository) RestoreFromTrash(id int64) error {
	res, err := r.db.Exec(
		`UPDATE buffers SET trash_status = ?, trashed_at = NULL, expires_at = NULL, updated_at = ? WHERE id = ? AND trash_status = ?`,
		buffer.TrashStatusActive, time.Now().Format(time.RFC3339), id, buffer.TrashStatusTrashed)
	if err != nil {
		return fmt.Errorf("restore buffer %d: %w", id, err)
	}
	return checkRowsAffected(res, id)
}

// Count returns the number of active buffers.
func (r *Repository) Count() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM buffers WHERE trash_status = ?`, buffer.TrashStatusActive).Scan(&n)
	return n, err
}

// DeleteExpiredTrash permanently removes all trashed buffers whose expiration has passed.
func (r *Repository) DeleteExpiredTrash() (int64, error) {
	now := time.Now().Format(time.RFC3339)
	res, err := r.db.Exec(
		`DELETE FROM buffers WHERE trash_status = ? AND expires_at IS NOT NULL AND expires_at <= ?`,
		buffer.TrashStatusTrashed, now)
	if err != nil {
		return 0, fmt.Errorf("delete expired trash: %w", err)
	}
	return res.RowsAffected()
}

// scanBuffer scans a row into a Buffer struct.
func scanBuffer(row interface {
	Scan(dest ...interface{}) error
}) (*buffer.Buffer, error) {
	var (
		id          int64
		label       string
		content     string
		lineCount   int
		byteCount   int
		tags        string
		createdAt   string
		updatedAt   string
		trashStatus int
		trashedAt   sql.NullString
		expiresAt   sql.NullString
	)
	if err := row.Scan(&id, &label, &content, &lineCount, &byteCount, &tags, &createdAt, &updatedAt, &trashStatus, &trashedAt, &expiresAt); err != nil {
		return nil, err
	}

	buf := &buffer.Buffer{
		ID:      id,
		Label:   label,
		Content: content,
		Metadata: buffer.Metadata{
			LineCount: lineCount,
			ByteCount: byteCount,
		},
		Tags:        splitTags(tags),
		TrashStatus: buffer.TrashStatus(trashStatus),
	}

	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		buf.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		buf.UpdatedAt = t
	}
	if trashedAt.Valid {
		if t, err := time.Parse(time.RFC3339, trashedAt.String); err == nil {
			buf.TrashedAt = &t
		}
	}
	if expiresAt.Valid {
		if t, err := time.Parse(time.RFC3339, expiresAt.String); err == nil {
			buf.ExpiresAt = &t
		}
	}

	return buf, nil
}

func joinTags(tags []string) string {
	b, err := json.Marshal(tags)
	if err != nil {
		return ""
	}
	return string(b)
}

func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(s), &tags); err != nil {
		return nil
	}
	return tags
}

func checkRowsAffected(res interface{ RowsAffected() (int64, error) }, id int64) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("buffer %d: %w", id, sql.ErrNoRows)
	}
	return nil
}

func nullTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}
