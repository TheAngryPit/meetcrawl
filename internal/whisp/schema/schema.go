package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Layout names a supported OpenWhispr notes table shape (macOS desktop or iOS).
type Layout int

const (
	LayoutUnknown Layout = iota
	LayoutDesktop
	LayoutIOS
)

func (l Layout) String() string {
	switch l {
	case LayoutDesktop:
		return "desktop"
	case LayoutIOS:
		return "ios"
	default:
		return "unknown"
	}
}

// NotesLayoutDesktop is the macOS desktop OpenWhispr notes table whispcrawl supports.
var NotesLayoutDesktop = []string{
	"id",
	"content",
	"enhanced_content",
	"note_type",
	"transcript",
	"created_at",
	"updated_at",
	"calendar_event_id",
	"participants",
	"deleted_at",
}

// NotesLayoutIOS is the OpenWhispr mobile (expo-sqlite / Drizzle) notes table from
// openwhispr-mobile/src/db/schema.ts. Detection requires every column listed here.
var NotesLayoutIOS = []string{
	"id",
	"title",
	"content",
	"folder_id",
	"note_type",
	"source_file",
	"audio_duration_seconds",
	"enhanced_content",
	"enhancement_prompt",
	"enhanced_at_content_hash",
	"transcript",
	"diarization_enabled",
	"expected_speaker_count",
	"transcription_status",
	"calendar_event_id",
	"participants",
	"client_note_id",
	"remote_id",
	"deleted_at",
	"pending_sync",
	"is_private",
	"space_id",
	"owner_user_id",
	"updated_by_user_id",
	"cloud_updated_at",
	"conflict_server_note",
	"left_team",
	"created_at",
	"updated_at",
}

// DetectNotesLayout classifies the notes table using PRAGMA table_info.
func DetectNotesLayout(ctx context.Context, db *sql.DB) (Layout, error) {
	cols, err := tableColumns(ctx, db, "notes")
	if err != nil {
		return LayoutUnknown, err
	}
	if missing := missingColumns(cols, NotesLayoutIOS); len(missing) == 0 {
		return LayoutIOS, nil
	}
	if missing := missingColumns(cols, NotesLayoutDesktop); len(missing) == 0 {
		return LayoutDesktop, nil
	}
	return LayoutUnknown, fmt.Errorf("PRAGMA table_info: unsupported notes layout")
}

// ValidateNotesTable accepts desktop or iOS layouts; anything else fails closed.
func ValidateNotesTable(ctx context.Context, db *sql.DB) error {
	layout, err := DetectNotesLayout(ctx, db)
	if err != nil {
		return err
	}
	if layout == LayoutUnknown {
		return fmt.Errorf("PRAGMA table_info: unsupported notes layout")
	}
	return nil
}

func missingColumns(cols map[string]bool, required []string) []string {
	var missing []string
	for _, name := range required {
		if !cols[name] {
			missing = append(missing, "notes."+name)
		}
	}
	return missing
}

func tableColumns(ctx context.Context, db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, "pragma table_info("+quoteIdent(table)+")")
	if err != nil {
		return nil, fmt.Errorf("pragma table_info(%s): %w", table, err)
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
