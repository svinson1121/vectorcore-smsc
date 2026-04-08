-- VectorCore SMSC — Migration 002 (SQLite)
-- S6c-to-SGd MME name mapping table.
-- No LISTEN/NOTIFY triggers; hot-reload uses polling instead.

CREATE TABLE IF NOT EXISTS sgd_mme_mappings (
    id          TEXT PRIMARY KEY,
    s6c_result  TEXT NOT NULL UNIQUE,
    sgd_host    TEXT NOT NULL,
    enabled     INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
