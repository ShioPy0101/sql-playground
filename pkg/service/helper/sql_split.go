package helper

import "strings"

// SplitSQLStatements splits on semicolons while preserving strings and comments.
func SplitSQLStatements(query string) []string {
	var statements []string
	var current strings.Builder
	var quote rune
	inLineComment := false
	inBlockComment := false
	escaped := false

	for i, r := range query {
		next := rune(0)
		if i+1 < len(query) {
			next = rune(query[i+1])
		}

		if inLineComment {
			current.WriteRune(r)
			if r == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			current.WriteRune(r)
			if r == '*' && next == '/' {
				continue
			}
			if r == '/' && i > 0 && query[i-1] == '*' {
				inBlockComment = false
			}
			continue
		}

		if quote == 0 && r == '-' && next == '-' {
			inLineComment = true
			current.WriteRune(r)
			continue
		}
		if quote == 0 && r == '/' && next == '*' {
			inBlockComment = true
			current.WriteRune(r)
			continue
		}

		if quote != 0 {
			current.WriteRune(r)
			if r == quote && !escaped {
				quote = 0
			}
			escaped = r == '\\' && !escaped
			if r != '\\' {
				escaped = false
			}
			continue
		}

		if r == '\'' || r == '"' || r == '`' {
			quote = r
			current.WriteRune(r)
			continue
		}

		if r == ';' {
			statement := strings.TrimSpace(current.String())
			if statement != "" {
				statements = append(statements, statement)
			}
			current.Reset()
			continue
		}

		current.WriteRune(r)
	}

	statement := strings.TrimSpace(current.String())
	if statement != "" {
		statements = append(statements, statement)
	}

	return statements
}
