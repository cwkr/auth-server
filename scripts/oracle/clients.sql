CREATE TABLE clients (
    client_id VARCHAR2(255 CHAR) NOT NULL CONSTRAINT clients_pk PRIMARY KEY,
    redirect_uri_pattern VARCHAR2(255 CHAR),
    secret_hash VARCHAR2(255 CHAR),
    session_name VARCHAR2(255 CHAR),
    disable_implicit NUMBER(1, 0) DEFAULT 0 NOT NULL,
    enable_refresh_token_rotation NUMBER(1, 0) DEFAULT 0 NOT NULL,
    created TIMESTAMP(3) WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP(3) NOT NULL,
    last_modified TIMESTAMP(3) WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP(3) NOT NULL
);
