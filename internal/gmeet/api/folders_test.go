package api

import "testing"

func TestEscapeDriveQuery(t *testing.T) {
	t.Parallel()
	got := escapeDriveQuery(`O'Reilly\lab`)
	want := `O\'Reilly\\lab`
	if got != want {
		t.Fatalf("escapeDriveQuery() = %q, want %q", got, want)
	}
}
