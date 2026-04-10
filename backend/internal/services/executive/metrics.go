package executive

import (
	"fmt"
	"sort"
	"time"
)

func computeMetrics(topics []Topic, orphanCommits []Commit, orphanWA []WAGroup) Metrics {
	var m Metrics
	m.CardsActive = len(topics)

	linkedCommitShas := make(map[string]struct{})
	for _, t := range topics {
		if len(t.Commits) > 0 {
			m.CardsWithCommits++
		}
		if t.Stale {
			m.StaleCardCount++
		}
		for _, c := range t.Commits {
			linkedCommitShas[c.SHA] = struct{}{}
		}
		if len(t.Messages) > 0 {
			m.WATopicsTicketed++
		}
	}
	m.CommitsLinked = len(linkedCommitShas)
	m.CommitsTotal = m.CommitsLinked + len(orphanCommits)
	m.WATopicsOrphan = len(orphanWA)

	if m.CommitsTotal > 0 {
		m.LinkagePctCommits = float64(m.CommitsLinked) / float64(m.CommitsTotal)
	}
	if m.CardsActive > 0 {
		m.LinkagePctCards = float64(m.CardsWithCommits) / float64(m.CardsActive)
	}
	return m
}

func buildDailyBuckets(r DateRange, commits []Commit, jiraChanges []JiraCard, wa []WAMessage) []DailyBucket {
	startDay := truncateDay(r.Start)
	endDay := truncateDay(r.End)

	index := map[time.Time]*DailyBucket{}
	var buckets []DailyBucket
	for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
		buckets = append(buckets, DailyBucket{Day: d})
	}
	for i := range buckets {
		index[buckets[i].Day] = &buckets[i]
	}
	for _, c := range commits {
		if b, ok := index[truncateDay(c.CommittedAt)]; ok {
			b.Commits++
		}
	}
	for _, j := range jiraChanges {
		if b, ok := index[truncateDay(j.UpdatedAt)]; ok {
			b.JiraChanges++
		}
	}
	for _, m := range wa {
		if b, ok := index[truncateDay(m.Timestamp)]; ok {
			b.WAMessages++
		}
	}
	return buckets
}

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// groupMessages clusters messages by (sender, time window). Messages must be
// sorted; we sort defensively.
func groupMessages(msgs []WAMessage, window time.Duration) []WAGroup {
	if len(msgs) == 0 {
		return nil
	}
	sorted := make([]WAMessage, len(msgs))
	copy(sorted, msgs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Timestamp.Before(sorted[j].Timestamp) })

	var groups []WAGroup
	var cur *WAGroup
	for _, m := range sorted {
		if cur != nil && cur.Messages[len(cur.Messages)-1].SenderName == m.SenderName &&
			m.Timestamp.Sub(cur.Messages[len(cur.Messages)-1].Timestamp) <= window {
			cur.Messages = append(cur.Messages, m)
			continue
		}
		groups = append(groups, WAGroup{
			Summary:   summaryFor(m),
			Messages:  []WAMessage{m},
			StartedAt: m.Timestamp,
		})
		cur = &groups[len(groups)-1]
	}
	return groups
}

func summaryFor(m WAMessage) string {
	c := m.Content
	if len(c) > 80 {
		c = c[:80]
	}
	return fmt.Sprintf("%s: %s", m.SenderName, c)
}
