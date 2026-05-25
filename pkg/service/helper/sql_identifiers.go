package helper

import "strings"

// quoteIdentifiers joins SQLite-quoted identifiers for generated SQL.
func quoteIdentifiers(identifiers []string) string {
	quoted := make([]string, len(identifiers))
	for i, identifier := range identifiers {
		quoted[i] = quoteIdentifier(identifier)
	}
	return strings.Join(quoted, ", ")
}

// quoteIdentifier quotes a SQLite identifier and escapes embedded quotes.
func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
