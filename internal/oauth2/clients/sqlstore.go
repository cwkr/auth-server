package clients

import (
	"database/sql"
	"errors"
	"github.com/blockloop/scan/v2"
	"github.com/cwkr/authd/internal/maputil"
	"github.com/cwkr/authd/internal/sqlutil"
	"log"
	"slices"
	"strings"
)

type sqlClient struct {
	RedirectURIPattern         sql.NullString `db:"redirect_uri_pattern"`
	SecretHash                 sql.NullString `db:"secret_hash"`
	SessionName                sql.NullString `db:"session_name"`
	DisableImplicit            sql.NullBool   `db:"disable_implicit"`
	EnableRefreshTokenRotation sql.NullBool   `db:"enable_refresh_token_rotation"`
}

func (s *sqlClient) Client() *Client {
	return &Client{
		RedirectURIPattern:         s.RedirectURIPattern.String,
		SecretHash:                 s.SecretHash.String,
		SessionName:                s.SessionName.String,
		DisableImplicit:            s.DisableImplicit.Bool,
		EnableRefreshTokenRotation: s.EnableRefreshTokenRotation.Bool,
	}
}

type sqlStore struct {
	inMemoryClientStore
	dbconn   *sql.DB
	settings *StoreSettings
}

func NewSqlStore(clientMap map[string]Client, dbs map[string]*sql.DB, settings *StoreSettings) (Store, error) {
	if dbconn, err := sqlutil.GetDB(dbs, settings.URI); err != nil {
		return nil, err
	} else {
		return &sqlStore{
			inMemoryClientStore: maputil.LowerKeys(clientMap),
			dbconn:              dbconn,
			settings:            settings,
		}, nil
	}
}

func (s *sqlStore) Authenticate(clientID, clientSecret string) (*Client, error) {
	if client, err := s.inMemoryClientStore.Authenticate(clientID, clientSecret); err == nil {
		return client, nil
	}
	if client, err := s.Lookup(clientID); err != nil {
		return nil, err
	} else {
		return s.inMemoryClientStore.compareSecret(client, clientSecret)
	}
}

func (s *sqlStore) Lookup(clientID string) (*Client, error) {
	if c, err := s.inMemoryClientStore.Lookup(clientID); err == nil {
		return c, nil
	}

	if strings.TrimSpace(s.settings.Query) == "" {
		log.Print("!!! SQL query empty")
		return nil, nil
	}

	var client sqlClient
	log.Printf("SQL: %s; -- %s", s.settings.Query, clientID)
	// SELECT redirect_uri_pattern, secret_hash, session_name, disable_implicit, enable_refresh_token_rotation
	// FROM clients WHERE lower(client_id) = lower($1)
	if rows, err := s.dbconn.Query(s.settings.Query, clientID); err == nil {
		if err := scan.RowStrict(&client, rows); err != nil {
			log.Printf("!!! Scan client failed: %v", err)
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrClientNotFound
			}
			return nil, err
		}
	} else {
		log.Printf("!!! Query for client failed: %v", err)
		return nil, err
	}
	log.Printf("%#v", client)
	return client.Client(), nil
}

type clientSessionInfo struct {
	ClientID    sql.NullString `db:"client_id"`
	SessionName sql.NullString `db:"session_name"`
}

func (s *sqlStore) PerSessionNameMap(defaultSessionName string) (map[string][]string, error) {
	var clientsPerSessionName = map[string][]string{}
	if c, err := s.inMemoryClientStore.PerSessionNameMap(defaultSessionName); err == nil {
		clientsPerSessionName = c
	} else {
		return nil, err
	}

	var inMemoryClientIDs []string

	for _, cs := range clientsPerSessionName {
		inMemoryClientIDs = append(inMemoryClientIDs, cs...)
	}

	var clients []clientSessionInfo

	log.Printf("SQL: %s", s.settings.QuerySessionNames)
	// SELECT client_id, COALESCE(session_name, '') as session_name FROM clients
	if rows, err := s.dbconn.Query(s.settings.QuerySessionNames); err == nil {
		if err := scan.RowsStrict(&clients, rows); err != nil {
			log.Printf("!!! Scan session_names failed: %v", err)
			return nil, err
		}
	} else {
		log.Printf("!!! Query for session_names failed: %v", err)
		return nil, err
	}

	for _, client := range clients {
		if !slices.ContainsFunc(inMemoryClientIDs, func(cid string) bool {
			return strings.EqualFold(cid, client.ClientID.String)
		}) {
			if client.SessionName.String != "" {
				clientsPerSessionName[client.SessionName.String] = append(clientsPerSessionName[client.SessionName.String], client.ClientID.String)
			} else if defaultSessionName != "" {
				clientsPerSessionName[defaultSessionName] = append(clientsPerSessionName[defaultSessionName], client.ClientID.String)
			} else {
				return nil, ErrSessionNameMissing
			}
		}
	}
	return clientsPerSessionName, nil
}

func (s *sqlStore) Ping() error {
	return s.dbconn.Ping()
}
