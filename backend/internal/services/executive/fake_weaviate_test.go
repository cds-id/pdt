package executive

import (
	"context"
	"time"
)

type fakeWeaviate struct {
	jira    []JiraCard
	commits []Commit
	wa      []WAMessage

	semanticCommits map[string][]CommitHit
	semanticWA      map[string][]WAHit
}

func (f *fakeWeaviate) ListJiraCards(_ context.Context, _ uint, _ *uint, start, end time.Time, limit int) ([]JiraCard, error) {
	var out []JiraCard
	for _, c := range f.jira {
		if (c.UpdatedAt.Equal(start) || c.UpdatedAt.After(start)) && !c.UpdatedAt.After(end) {
			out = append(out, c)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeWeaviate) ListCommits(_ context.Context, _ uint, start, end time.Time) ([]Commit, error) {
	var out []Commit
	for _, c := range f.commits {
		if (c.CommittedAt.Equal(start) || c.CommittedAt.After(start)) && !c.CommittedAt.After(end) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeWeaviate) ListWAMessages(_ context.Context, _ uint, start, end time.Time) ([]WAMessage, error) {
	var out []WAMessage
	for _, m := range f.wa {
		if (m.Timestamp.Equal(start) || m.Timestamp.After(start)) && !m.Timestamp.After(end) {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f *fakeWeaviate) SemanticCommits(_ context.Context, _ uint, query string, _, _ time.Time, _ int) ([]CommitHit, error) {
	return f.semanticCommits[query], nil
}

func (f *fakeWeaviate) SemanticWA(_ context.Context, _ uint, query string, _, _ time.Time, _ int) ([]WAHit, error) {
	return f.semanticWA[query], nil
}
