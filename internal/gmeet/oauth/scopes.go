package oauth

const (
	ScopeDriveReadonly    = "https://www.googleapis.com/auth/drive.readonly"
	ScopeCalendarReadonly = "https://www.googleapis.com/auth/calendar.events.readonly"
)

func Scopes() []string {
	return []string{ScopeDriveReadonly, ScopeCalendarReadonly}
}
