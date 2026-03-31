package scheduler

import "testing"

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		response  string
		status    string
		want      bool
	}{
		{"always", "always", "anything", "completed", true},
		{"empty condition", "", "anything", "completed", true},
		{"status completed match", "status:completed", "", "completed", true},
		{"status completed no match", "status:completed", "", "failed", false},
		{"status failed match", "status:failed", "", "failed", true},
		{"contains match", "contains:blocker", "There is a BLOCKER in sprint", "completed", true},
		{"contains no match", "contains:blocker", "All clear", "completed", false},
		{"unknown condition", "unknown:foo", "", "completed", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateCondition(tt.condition, tt.response, tt.status)
			if got != tt.want {
				t.Errorf("evaluateCondition(%q, %q, %q) = %v, want %v", tt.condition, tt.response, tt.status, got, tt.want)
			}
		})
	}
}

func TestSummarize(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short text", "Hello world", 500, "Hello world"},
		{"with heading", "# Daily Report\n\nSome content here", 500, "Daily Report"},
		{"long text truncated", "This is a very long text that should be truncated", 20, "This is a very long "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarize(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("summarize(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}
