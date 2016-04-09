CREATE TABLE IF NOT EXISTS links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    url TEXT NOT NULL,
    createdate DATE DEFAULT (datetime('now','localtime'))
);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    slug TEXT NOT NULL
);
CREATE INDEX fk_tags_slug ON tags (slug);

CREATE TABLE IF NOT EXISTS rel_links_tags (
    link_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL
);

CREATE INDEX fk_links_tags ON rel_links_tags (link_id, tag_id);
