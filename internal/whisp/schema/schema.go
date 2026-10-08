package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// NotesLayoutV1 is the OpenWhispr local notes table layout whispcrawl supports.
var NotesLayoutV1 = []string{
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

func ValidateNotesTable(ctx context.Context, db *sql.DB) error {
	cols, err := tableColumns(ctx, db, "notes")
	if err != nil {
		return err
	}
	var missing []string
	for _, required := range NotesLayoutV1 {
		if !cols[required] {
			missing = append(missing, "notes."+required)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("PRAGMA table_info missing %s", strings.Join(missing, ", "))
	}
	return nil
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
