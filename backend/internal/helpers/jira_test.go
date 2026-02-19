package helpers

import "testing"

func TestFilterByProjectKeys(t *testing.T) {
	tests := []struct {
		name        string
		cardKey     string
		projectKeys string
		want        bool
	}{
		{"empty keys allows all", "PDT-123", "", true},
		{"matching single key", "PDT-123", "PDT", true},
		{"matching first of multiple", "PDT-123", "PDT,CORE", true},
		{"matching second of multiple", "CORE-456", "PDT,CORE", true},
		{"no match", "OTHER-789", "PDT,CORE", false},
		{"partial prefix no match", "PDTX-123", "PDT", false},
		{"whitespace in keys", "CORE-1", " PDT , CORE ", true},
		{"empty card key", "", "PDT", false},
		{"key without dash", "PDT", "PDT", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterByProjectKeys(tt.cardKey, tt.projectKeys)
			if got != tt.want {
				t.Errorf("FilterByProjectKeys(%q, %q) = %v, want %v",
					tt.cardKey, tt.projectKeys, got, tt.want)
			}
		})
	}
}

func TestBuildProjectKeyWhereClauses(t *testing.T) {
	tests := []struct {
		name        string
		projectKeys string
		column      string
		wantClause  string
		wantArgs    int
	}{
		{"empty returns empty", "", "card_key", "", 0},
		{"single key", "PDT", "card_key", "card_key LIKE ?", 1},
		{"multiple keys", "PDT,CORE", "card_key", "(card_key LIKE ? OR card_key LIKE ?)", 2},
		{"whitespace trimmed", " PDT , CORE ", "k", "(k LIKE ? OR k LIKE ?)", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clause, args := BuildProjectKeyWhereClauses(tt.projectKeys, tt.column)
			if clause != tt.wantClause {
				t.Errorf("clause = %q, want %q", clause, tt.wantClause)
			}
			if len(args) != tt.wantArgs {
				t.Errorf("args len = %d, want %d", len(args), tt.wantArgs)
			}
		})
	}
}
