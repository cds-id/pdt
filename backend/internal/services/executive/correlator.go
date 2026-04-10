package executive

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type Correlator struct {
	Client WeaviateClient
	Now    func() time.Time
}

func NewCorrelator(client WeaviateClient) *Correlator {
	return &Correlator{Client: client, Now: time.Now}
}

func (c *Correlator) Build(ctx context.Context, userID uint, workspaceID *uint, r DateRange, staleDays int) (*CorrelatedDataset, error) {
	if staleDays <= 0 {
		staleDays = StaleThresholdDefault
	}

	var (
		anchors    []JiraCard
		rawCommits []Commit
		rawWA      []WAMessage
		truncated  bool
	)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		list, err := c.Client.ListJiraCards(gctx, userID, workspaceID, r.Start, r.End, MaxAnchors+1)
		if err != nil {
			return err
		}
		if len(list) > MaxAnchors {
			truncated = true
			list = list[:MaxAnchors]
		}
		anchors = list
		return nil
	})
	g.Go(func() error {
		list, err := c.Client.ListCommits(gctx, userID, r.Start, r.End)
		rawCommits = list
		return err
	})
	g.Go(func() error {
		list, err := c.Client.ListWAMessages(gctx, userID, r.Start, r.End)
		rawWA = list
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	topics := c.buildTopics(ctx, userID, anchors, rawCommits, r, staleDays)

	orphanCommits := subtractCommits(rawCommits, topicCommitSet(topics))
	orphanWAMsgs := subtractWA(rawWA, topicWASet(topics))
	orphanWA := filterNoise(groupMessages(orphanWAMsgs, OrphanWAGroupWindow), OrphanWANoiseFloor)

	ds := &CorrelatedDataset{
		UserID:        userID,
		WorkspaceID:   workspaceID,
		Range:         r,
		Topics:        topics,
		OrphanWA:      orphanWA,
		OrphanCommits: orphanCommits,
		DailyBuckets:  buildDailyBuckets(r, rawCommits, anchors, rawWA),
	}
	ds.Metrics = computeMetrics(topics, orphanCommits, orphanWA)
	ds.Metrics.Truncated = truncated
	return ds, nil
}

func (c *Correlator) buildTopics(ctx context.Context, userID uint, anchors []JiraCard, rawCommits []Commit, r DateRange, staleDays int) []Topic {
	sem := make(chan struct{}, PerAnchorWorkers)
	var mu sync.Mutex
	topics := make([]Topic, len(anchors))

	var wg sync.WaitGroup
	for i, card := range anchors {
		i, card := i, card
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			commits := matchCommitsForCard(ctx, c.Client, userID, card, rawCommits, r)
			wa := matchWAForCard(ctx, c.Client, userID, card, r)

			daysIdle := int(c.Now().Sub(card.UpdatedAt).Hours() / 24)
			stale := strings.EqualFold(card.Status, "In Progress") && len(commits) == 0 && daysIdle >= staleDays

			mu.Lock()
			topics[i] = Topic{
				Anchor:   card,
				Messages: flattenGroupMessages(groupMessages(wa, WAGroupWindow)),
				Commits:  commits,
				Stale:    stale,
				DaysIdle: daysIdle,
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	return topics
}

var cardKeyRe = regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)

func matchCommitsForCard(ctx context.Context, client WeaviateClient, userID uint, card JiraCard, rawCommits []Commit, r DateRange) []Commit {
	found := map[string]Commit{}
	for _, c := range rawCommits {
		for _, key := range cardKeyRe.FindAllString(c.Message, -1) {
			if key == card.CardKey {
				found[c.SHA] = c
			}
		}
	}
	hits, err := client.SemanticCommits(ctx, userID, card.Content, r.Start, r.End, SemanticCommitLimit)
	if err == nil {
		for _, h := range hits {
			if h.Distance > CommitDistanceMax {
				continue
			}
			if _, ok := found[h.Commit.SHA]; !ok {
				found[h.Commit.SHA] = h.Commit
			}
		}
	}
	out := make([]Commit, 0, len(found))
	for _, v := range found {
		out = append(out, v)
	}
	return out
}

func matchWAForCard(ctx context.Context, client WeaviateClient, userID uint, card JiraCard, r DateRange) []WAMessage {
	hits, err := client.SemanticWA(ctx, userID, card.Content, r.Start, r.End, SemanticWALimit)
	if err != nil {
		return nil
	}
	var out []WAMessage
	for _, h := range hits {
		if h.Distance > WADistanceMax {
			continue
		}
		out = append(out, h.Message)
	}
	return out
}

func flattenGroupMessages(groups []WAGroup) []WAMessage {
	var out []WAMessage
	for _, g := range groups {
		out = append(out, g.Messages...)
	}
	return out
}

func topicCommitSet(topics []Topic) map[string]struct{} {
	s := map[string]struct{}{}
	for _, t := range topics {
		for _, c := range t.Commits {
			s[c.SHA] = struct{}{}
		}
	}
	return s
}

func topicWASet(topics []Topic) map[string]struct{} {
	s := map[string]struct{}{}
	for _, t := range topics {
		for _, m := range t.Messages {
			s[m.MessageID] = struct{}{}
		}
	}
	return s
}

func subtractCommits(all []Commit, exclude map[string]struct{}) []Commit {
	var out []Commit
	for _, c := range all {
		if _, found := exclude[c.SHA]; !found {
			out = append(out, c)
		}
	}
	return out
}

func subtractWA(all []WAMessage, exclude map[string]struct{}) []WAMessage {
	var out []WAMessage
	for _, m := range all {
		if _, found := exclude[m.MessageID]; !found {
			out = append(out, m)
		}
	}
	return out
}

func filterNoise(groups []WAGroup, floor int) []WAGroup {
	var out []WAGroup
	for _, g := range groups {
		if len(g.Messages) >= floor {
			out = append(out, g)
		}
	}
	return out
}
