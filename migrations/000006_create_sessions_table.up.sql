CREATE TABLE sessions (
                          token TEXT PRIMARY KEY,
                          data BYTEA NOT NULL,
                          expiry TIMESTAMPTZ NOT NULL
);