package archive

import "strings"

func trimSQL(query string) string {
	return strings.TrimSpace(query)
}

func IsReadOnlySQL(query string) bool {
	query = strings.TrimLeft(strings.ToLower(trimSQL(query)), " \t\r\n(")
	return strings.HasPrefix(query, "select ") ||
		strings.HasPrefix(query, "select\n") ||
		strings.HasPrefix(query, "with ") ||
		strings.HasPrefix(query, "with\n") ||
		strings.HasPrefix(query, "pragma ")
}
