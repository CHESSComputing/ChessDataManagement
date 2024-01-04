--https://stackoverflow.com/questions/18387209/sqlite-syntax-for-creating-table-with-foreign-key

create TABLE processing (
    processing_id INTEGER PRIMARY KEY AUTOINCREMENT,
    processing VARCHAR(255) NOT NULL UNIQUE,
    create_at INTEGER,
    create_by VARCHAR(255),
    modify_at INTEGER,
    modify_by VARCHAR(255)
);
create TABLE parents (
    parent_id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent VARCHAR(255) NOT NULL UNIQUE,
    create_at INTEGER,
    create_by VARCHAR(255),
    modify_at INTEGER,
    modify_by VARCHAR(255)
);
create TABLE sites (
    site_id INTEGER PRIMARY KEY AUTOINCREMENT,
    site VARCHAR(255) NOT NULL UNIQUE,
    create_at INTEGER,
    create_by VARCHAR(255),
    modify_at INTEGER,
    modify_by VARCHAR(255)
);
create TABLE buckets (
    bucket_id INTEGER PRIMARY KEY AUTOINCREMENT,
    bucket VARCHAR(255) NOT NULL UNIQUE,
    meta_id VARCHAR(255),
    dataset_id INTEGER REFERENCES datasets(dataset_id) ON UPDATE CASCADE,
    create_at INTEGER,
    create_by VARCHAR(255),
    modify_at INTEGER,
    modify_by VARCHAR(255)
);
create TABLE metadata (
    meta_id INTEGER PRIMARY KEY AUTOINCREMENT,
    did VARCHAR(255) NOT NULL UNIQUE,
    create_at INTEGER,
    create_by VARCHAR(255),
    modify_at INTEGER,
    modify_by VARCHAR(255)
);
create TABLE datasets (
    dataset_id INTEGER PRIMARY KEY AUTOINCREMENT,
    dataset VARCHAR(255) NOT NULL UNIQUE,
    meta_id INTEGER REFERENCES metadata(meta_id) ON UPDATE CASCADE,
    site_id INTEGER REFERENCES sites(site_id) ON UPDATE CASCADE,
    processing_id INTEGER REFERENCES processingS(processing_id) ON UPDATE CASCADE,
    parent_id INTEGER REFERENCES parents(parent_id) ON UPDATE CASCADE,
    create_at INTEGER,
    create_by VARCHAR(255),
    modify_at INTEGER,
    modify_by VARCHAR(255)
);
create TABLE files (
    file_id INTEGER PRIMARY KEY AUTOINCREMENT,
    file VARCHAR(255) NOT NULL UNIQUE,
    is_file_valid INTEGER DEFAULT 1,
    meta_id INTEGER REFERENCES metadata(meta_id) ON UPDATE CASCADE,
    dataset_id INTEGER REFERENCES datasets(dataset_id) ON UPDATE CASCADE,
    create_at INTEGER,
    create_by VARCHAR(255),
    modify_at INTEGER,
    modify_by VARCHAR(255)
);
