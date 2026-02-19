package helpers

import "strings"

// FilterByProjectKeys checks if a Jira card key matches configured project prefixes.
// Returns true if projectKeys is empty (no filter) or cardKey matches any prefix.
func FilterByProjectKeys(cardKey string, projectKeys string) bool {
	if projectKeys == "" {
		return true
	}
	for _, k := range strings.Split(projectKeys, ",") {
		prefix := strings.TrimSpace(k) + "-"
		if prefix != "-" && strings.HasPrefix(cardKey, prefix) {
			return true
		}
	}
	return false
}

// BuildProjectKeyWhereClauses builds a SQL WHERE clause for filtering by project key prefixes.
// Returns empty string and nil args if projectKeys is empty.
func BuildProjectKeyWhereClauses(projectKeys string, column string) (string, []interface{}) {
	if projectKeys == "" {
		return "", nil
	}
	keys := strings.Split(projectKeys, ",")
	var clauses []string
	var args []interface{}
	for _, k := range keys {
		trimmed := strings.TrimSpace(k)
		if trimmed == "" {
			continue
		}
		clauses = append(clauses, column+" LIKE ?")
		args = append(args, trimmed+"-%")
	}
	if len(clauses) == 0 {
		return "", nil
	}
	if len(clauses) == 1 {
		return clauses[0], args
	}
	return "(" + strings.Join(clauses, " OR ") + ")", args
}
