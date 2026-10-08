package build

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/index"
	"github.com/TheAngryPit/meetcrawl/internal/index/archive"
	"github.com/TheAngryPit/meetcrawl/internal/index/crawler"
	"github.com/TheAngryPit/meetcrawl/internal/index/privacy"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type SourceArchive struct {
	Kind source.Kind
	Path string
}

type Options struct {
	IndexDBPath string
	Sources     []SourceArchive
	Privacy     privacy.Config
}

type Result struct {
	Meetings    int `json:"meetings"`
	Contents    int `json:"contents"`
	SourceLinks int `json:"source_links"`
}

func Run(ctx context.Context, opts Options) (Result, error) {
	var rows []crawler.Row
	for _, src := range opts.Sources {
		loaded, err := crawler.LoadArtifacts(ctx, src.Path, src.Kind)
		if err != nil {
			return Result{}, err
		}
		rows = append(rows, loaded...)
	}
	if len(rows) == 0 {
		return Result{}, fmt.Errorf("index: no artifacts in crawler archives")
	}
	privacyCfg, err := opts.Privacy.Compile()
	if err != nil {
		return Result{}, err
	}
	meetingIDs, err := index.AssignMeetingIDs(rows)
	if err != nil {
		return Result{}, err
	}
	for i := range rows {
		class, err := privacyCfg.ClassFor(rows[i], "")
		if err != nil {
			return Result{}, err
		}
		rows[i].Privacy = class
	}
	meetings, contents, err := assemble(rows, meetingIDs)
	if err != nil {
		return Result{}, err
	}
	store, err := archive.Open(ctx, opts.IndexDBPath)
	if err != nil {
		return Result{}, err
	}
	defer store.Close()
	finishedAt := time.Now().UTC()
	if err := store.ReplaceIndex(ctx, meetings, contents, finishedAt); err != nil {
		return Result{}, err
	}
	sourceLinks := 0
	for _, content := range contents {
		sourceLinks += len(content.Sources)
	}
	return Result{
		Meetings:    len(meetings),
		Contents:    len(contents),
		SourceLinks: sourceLinks,
	}, nil
}

func assemble(rows []crawler.Row, meetingIDs map[int]string) ([]archive.MeetingRecord, []archive.ContentRecord, error) {
	byMeeting := map[string][]int{}
	for i, id := range meetingIDs {
		byMeeting[id] = append(byMeeting[id], i)
	}
	meetingKeys := sortedKeys(byMeeting)
	var meetings []archive.MeetingRecord
	var contents []archive.ContentRecord
	for _, meetingID := range meetingKeys {
		idxs := byMeeting[meetingID]
		meetingRows := make([]crawler.Row, len(idxs))
		for j, idx := range idxs {
			meetingRows[j] = rows[idx]
		}
		fidelitySet := map[source.Fidelity]struct{}{}
		privacyClasses := make([]source.PrivacyClass, len(meetingRows))
		for i, row := range meetingRows {
			fidelitySet[row.Fidelity] = struct{}{}
			privacyClasses[i] = row.Privacy
		}
		var fidelities []source.Fidelity
		for f := range fidelitySet {
			fidelities = append(fidelities, f)
		}
		sort.Slice(fidelities, func(i, j int) bool {
			return fidelities[i].Rank() < fidelities[j].Rank()
		})
		ical, eventStart := calendarFields(meetingID, meetingRows)
		meetings = append(meetings, archive.MeetingRecord{
			MeetingID:     meetingID,
			BestFidelity:  index.BestMeetingFidelity(meetingRows),
			Privacy:       index.StrictestPrivacy(privacyClasses),
			ICalUID:       ical,
			EventStartUTC: eventStart,
			Fidelities:    fidelities,
		})
		contentGroups := map[string][]int{}
		for _, idx := range idxs {
			hash := rows[idx].ContentHash
			contentGroups[hash] = append(contentGroups[hash], idx)
		}
		for _, hash := range sortedKeys(contentGroups) {
			group := contentGroups[hash]
			first := rows[group[0]]
			content := archive.ContentRecord{
				MeetingID:      meetingID,
				ContentHash:    hash,
				NormalizedText: first.NormalizedText,
				Fidelity:       bestContentFidelity(group, rows),
				Language:       first.Language,
			}
			sort.Slice(group, func(i, j int) bool {
				a, b := rows[group[i]], rows[group[j]]
				if a.Source.String() != b.Source.String() {
					return a.Source.String() < b.Source.String()
				}
				return a.SourceID < b.SourceID
			})
			for _, idx := range group {
				row := rows[idx]
				content.Sources = append(content.Sources, archive.SourceLink{
					Source:         row.Source.String(),
					SourceID:       row.SourceID,
					SourceRevision: row.SourceRevision,
					ProvenanceHash: row.ProvenanceHash,
					Fidelity:       row.Fidelity,
				})
			}
			contents = append(contents, content)
		}
	}
	sort.Slice(contents, func(i, j int) bool {
		if contents[i].MeetingID != contents[j].MeetingID {
			return contents[i].MeetingID < contents[j].MeetingID
		}
		return contents[i].ContentHash < contents[j].ContentHash
	})
	return meetings, contents, nil
}

func calendarFields(meetingID string, rows []crawler.Row) (string, string) {
	if strings.HasPrefix(meetingID, "adhoc:") {
		return "", ""
	}
	var ical string
	var start time.Time
	for _, row := range rows {
		if row.CalendarICal == "" || row.WindowStart.IsZero() {
			continue
		}
		if ical == "" {
			ical = row.CalendarICal
			start = row.WindowStart
			continue
		}
		if row.WindowStart.Before(start) {
			start = row.WindowStart
		}
	}
	if ical == "" {
		return "", ""
	}
	return ical, start.UTC().Format(time.RFC3339)
}

func bestContentFidelity(idxs []int, rows []crawler.Row) source.Fidelity {
	var best source.Fidelity
	for _, idx := range idxs {
		best = source.BestFidelity(best, rows[idx].Fidelity)
	}
	return best
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
