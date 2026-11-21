package api

import (
	"strconv"
)

func idString(id int64) string { return strconv.FormatInt(id, 10) }

func intString(n int64) string { return strconv.FormatInt(n, 10) }

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func parsePositiveInt(raw string) (int, error) {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, strconv.ErrSyntax
	}
	return n, nil
}
