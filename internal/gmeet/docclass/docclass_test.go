package docclass_test

import (
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/docclass"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

func TestClassify(t *testing.T) {
	t.Parallel()
	cases := []struct {
		title string
		want  source.Fidelity
	}{
		{"Team sync - Notes by Gemini", source.FidelityNotes},
		{"Team sync - Transcript", source.FidelityTranscript},
	}
	for _, tc := range cases {
		got, err := docclass.Classify(tc.title)
		if err != nil {
			t.Fatalf("Classify(%q) = %v", tc.title, err)
		}
		if got != tc.want {
			t.Fatalf("Classify(%q) = %v, want %v", tc.title, got, tc.want)
		}
	}
	if _, err := docclass.Classify("Random meeting notes"); err == nil {
		t.Fatal("expected error for unknown title")
	}
}
