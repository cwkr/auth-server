# Auth Development Tools

## Auth Server

This is a simple OAuth2 authorization server implementation supporting *Implicit*,
*Authorization Code* (with and without *PKCE*), *Refresh Token*, *Password* and
*Client Credentials* grant types.

It is possible to use PostgreSQL, Oracle Database or LDAP as people store.

### Install

```shell
go install github.com/cwkr/authd/cmd/auth-server@latest
```

### Settings

#### PostgreSQL as people store

```jsonc
{
  "issuer": "http://localhost:6080/",
  "port": 6080,
  "users": {
    "user": {
      "given_name": "First Name",
      "family_name": "Last Name",
      "groups": [
        "admin"
      ],
      "password_hash": "$2a$12$yos0Nv/lfhjKjJ7CSmkCteSJRmzkirYwGFlBqeY4ss3o3nFSb5WDy"
    }
  },
  // load signing key from file 
  "key": "@mykey.pem",
  // extra public keys to include in jwks
  "additional_keys": [
    "@othe.key",
    "http://localhost:7654/jwks.json"
  ],
  "clients": {
    "app": {
      "redirect_uri_pattern": "https?:\\/\\/localhost(:\\d+)?\\/"
    }
  },
  "client_store": {
    "uri": "postgresql://authserver:trustno1@localhost:5432/dev?sslmode=disable",
    "query": "SELECT redirect_uri_pattern, secret_hash, session_name, disable_implicit, enable_refresh_token_rotation FROM clients WHERE lower(client_id) = lower($1)",
    "query_session_names": "SELECT client_id, session_name FROM clients"
  },
  // define custom access token claims
  "access_token_extra_claims": {
    "prn": "$user_id",
    "email": "$email",
    "givenName": "$given_name",
    "groups": "$groups_semicolon_delimited",
    "sn": "$family_name",
    "user_id": "$user_id"
  },
  // define custom id token claims
  "id_token_extra_claims": {
    "groups": "$groups"
  },
  // available scopes
  "extra_scope": "profile email offline_access",
  "access_token_ttl": 3600,
  "refresh_token_ttl": 28800,
  "session_secret": "AwBVrwW0boviWc3L12PplWTEgO4B4dxi",
  "session_name": "_auth",
  "session_ttl": 28800,
  "keys_ttl": 900,
  "people_store": {
    "uri": "postgresql://authserver:trustno1@localhost:5432/dev?sslmode=disable",
    "credentials_query": "SELECT user_id, password_hash FROM users WHERE lower(user_id) = lower($1)",
    "groups_query": "SELECT UNNEST(groups) FROM users WHERE lower(user_id) = lower($1)",
    "details_query": "SELECT given_name, family_name, email, TO_CHAR(birthdate, 'YYYY-MM-DD') birthdate, department, phone_number, street_address, locality, postal_code FROM people WHERE lower(user_id) = lower($1)",
    "update": "UPDATE people SET given_name = $2, family_name = $3, email = $4, department = $5, birthdate = TO_DATE($6, 'YYYY-MM-DD'), phone_number = $7, locality = $8, street_address = $9, postal_code = $10, last_modified = now() WHERE lower(user_id) = lower($1)",
    "set_password": "UPDATE people SET password_hash = $2, last_modified = now() WHERE lower(user_id) = lower($1)"
  },
  "disable_api": false,
  "roles": {
    "*": {
      "by_group": ["*"]
    },
    "all_users": {
      "by_group": ["*"]
    },
    "admin": {
      "by_user_id": [
        "user1",
        "user2"
      ]
    }
  }
}
```

#### Oracle Internt Directory (LDAP) as people store

```jsonc
{
  "issuer": "http://localhost:6080/",
  "port": 6080,
  // load signing key from file
  "key": "@mykey.pem",
  "clients": {
    "app": {
      "redirect_uri_pattern": "https?:\\/\\/localhost(:\\d+)?\\/"
    }
  },
  "access_token_extra_claims": {
    "prn": "$user_id",
    "email": "$email",
    "givenName": "$given_name",
    "groups": "$groups_semicolon_delimited",
    "sn": "$family_name",
    "user_id": "$user_id"
  },
  "extra_scope": "profile",
  "access_token_ttl": 3600,
  "refresh_token_ttl": 28800,
  "session_secret": "j2mejSKidaFJ38wjxaf2amQRmZ4Mtibp",
  "session_name": "_auth",
  "session_ttl": 28800,
  "keys_ttl": 900,
  "people_store": {
    "uri": "ldaps://cn=access_user,cn=Users,dc=example,dc=org:trustno1@oid.example.org:3070",
    "credentials_query": "(&(objectClass=person)(uid=%s))",
    "groups_query": "(&(objectClass=groupOfUniqueNames)(uniquemember=%s))",
    "details_query": "(&(objectClass=person)(uid=%s))",
    "parameters": {
      "base_dn": "dc=example,dc=org",
      "user_id_attribute": "uid",
      "group_id_attribute": "dn",      
      "department_attribute": "departmentnumber",
      "email_attribute": "mail",
      "family_name_attribute": "sn",
      "given_name_attribute": "givenname",
      "phone_number_attribute": "telephonenumber",
      "street_address_attribute": "street",
      "locality_attribute": "l",
      "postal_code_attribute": "postalcode"
    }
  },
  "disable_api": false
}
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details
