package sqlutil

import (
	"database/sql"
	"errors"
	_ "github.com/lib/pq"
	_ "github.com/sijms/go-ora/v2"
	"strings"
)

var (
	ErrUnsupportedDatabase = errors.New("unsupported database")
)

func GetDB(dbs map[string]*sql.DB, uri string) (*sql.DB, error) {
	if dbs[uri] == nil {
		var driver string
		switch {
		case strings.HasPrefix(uri, PrefixPostgres):
			driver = "postgres"
		case strings.HasPrefix(uri, PrefixOracle):
			driver = "oracle"
		default:
			return nil, ErrUnsupportedDatabase
		}
		if dbconn, err := sql.Open(driver, uri); err != nil {
			return nil, err
		} else {
			dbs[uri] = dbconn
		}
	}
	return dbs[uri], nil
}
