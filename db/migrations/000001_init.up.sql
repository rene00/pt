CREATE TABLE
IF NOT EXISTS hash
(
    id INTEGER NOT NULL PRIMARY KEY,
    hash TEXT NOT NULL,
    filepath TEXT NOT NULL,
    UNIQUE (hash, filepath)
);

CREATE TABLE
IF NOT EXISTS meta_key
(
    id INTEGER NOT NULL PRIMARY KEY,
    key_name TEXT NOT NULL,
    UNIQUE (key_name)
);

CREATE TABLE
IF NOT EXISTS meta
(
    id INTEGER NOT NULL PRIMARY KEY,
    hash_id INTEGER NOT NULL,
    meta_key_id INTEGER NOT NULL,
    value TEXT NOT NULL,
    UNIQUE (hash_id, meta_key_id),
    FOREIGN KEY(hash_id) REFERENCES hash(id),
    FOREIGN KEY(meta_key_id) REFERENCES meta_key(id)
);
