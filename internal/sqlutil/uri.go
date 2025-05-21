package sqlutil

import "strings"

const (
	PrefixPostgres = "postgresql:"
	PrefixOracle   = "oracle:"
)

func IsDatabaseURI(uri string) bool {
	return strings.HasPrefix(uri, PrefixPostgres) || strings.HasPrefix(uri, PrefixOracle)
}
