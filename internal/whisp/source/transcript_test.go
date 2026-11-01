package source_test

import (
	"testing"

	wsource "github.com/TheAngryPit/meetcrawl/internal/whisp/source"
)

func TestFormatTranscript(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "resolved names and timestamps",
			raw:  `[{"text":"Hello everyone.","source":"mic","timestamp":0,"speaker":"speaker_0","speakerName":"Vitor"},{"text":"Release is on track.","source":"system","timestamp":6.5,"speaker":"speaker_1","speakerName":"Ana","suggestedProfileId":7}]`,
			want: "[00:00] Vitor: Hello everyone.\n[00:06] Ana: Release is on track.",
		},
		{
			name: "diarization label beats the audio track",
			raw:  `[{"text":"One.","source":"system","speaker":"speaker_2"}]`,
			want: "speaker_2: One.",
		},
		{
			name: "audio track is the fallback label",
			raw:  `[{"text":"Mine.","source":"mic","timestamp":5},{"text":"Theirs.","source":"system","timestamp":9}]`,
			want: "[00:05] Microphone: Mine.\n[00:09] System audio: Theirs.",
		},
		{
			name: "recorder name labels their own microphone track",
			raw:  `[{"text":"Mine.","source":"mic","recorderName":"Vitor"},{"text":"Theirs.","source":"system","recorderName":"Vitor"}]`,
			want: "Vitor: Mine.\nSystem audio: Theirs.",
		},
		{
			name: "segments with no speaker information have no label",
			raw:  `[{"text":"First paragraph."},{"text":"Second paragraph.","timestamp":3}]`,
			want: "First paragraph.\n[00:03] Second paragraph.",
		},
		{
			name: "hours use a three part clock and order is kept",
			raw:  `[{"text":"Late.","source":"mic","timestamp":3725},{"text":"Early.","source":"mic","timestamp":1}]`,
			want: "[1:02:05] Microphone: Late.\n[00:01] Microphone: Early.",
		},
		{
			name: "blank segments are dropped",
			raw:  `[{"text":"  ","source":"mic"},{"text":"Kept.","source":"mic"}]`,
			want: "Microphone: Kept.",
		},
		{
			name: "plain text from older versions is unchanged",
			raw:  "Plain text transcript from v1.6.0 era.",
			want: "Plain text transcript from v1.6.0 era.",
		},
		{
			name: "bracketed text that is not JSON is unchanged",
			raw:  "[inaudible] and then we left.",
			want: "[inaudible] and then we left.",
		},
		{
			name: "array that is not segments is unchanged",
			raw:  `["a","b"]`,
			want: `["a","b"]`,
		},
		{
			name: "array with a null element is unchanged",
			raw:  `[null]`,
			want: `[null]`,
		},
		{
			name: "objects without a text field are unchanged",
			raw:  `[{"other":"value"}]`,
			want: `[{"other":"value"}]`,
		},
		{
			name: "one object without text makes the whole array unchanged",
			raw:  `[{"text":"Kept?","source":"mic"},{"source":"system"}]`,
			want: `[{"text":"Kept?","source":"mic"},{"source":"system"}]`,
		},
		{
			name: "segment array with no text is an empty transcript",
			raw:  `[]`,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := wsource.FormatTranscript(tt.raw); got != tt.want {
				t.Fatalf("FormatTranscript() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestArtifactsForNoteIndexesReadableTranscript(t *testing.T) {
	t.Parallel()
	note := wsource.MeetingNote{
		ID:             1,
		Transcript:     `[{"text":"Hello.","source":"mic","timestamp":0,"speakerName":"Vitor"}]`,
		SourceRevision: "2026-10-20 14:35:00",
	}
	artifacts, err := wsource.ArtifactsForNote(note, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("artifacts = %d, want 1", len(artifacts))
	}
	if got, want := artifacts[0].NormalizedText, "[00:00] Vitor: Hello."; got != want {
		t.Fatalf("NormalizedText = %q, want %q", got, want)
	}
}
