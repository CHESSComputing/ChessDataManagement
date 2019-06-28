--https://stackoverflow.com/questions/18387209/sqlite-syntax-for-creating-table-with-foreign-key

CREATE TABLE experiments (
 experiment_id INTEGER PRIMARY KEY AUTOINCREMENT,
 name TEXT NOT NULL
);

CREATE TABLE tiers (
 tier_id INTEGER PRIMARY KEY AUTOINCREMENT,
 name TEXT NOT NULL
);

CREATE TABLE processing (
 processing_id INTEGER PRIMARY KEY AUTOINCREMENT,
 name TEXT NOT NULL
);

CREATE TABLE datasets (
 dataset_id INTEGER PRIMARY KEY AUTOINCREMENT,
 experiment_id integer REFERENCES experiments(experiment_id) ON UPDATE CASCADE,
 processing_id integer REFERENCES processing(processing_id) ON UPDATE CASCADE,
 tier_id integer REFERENCES tiers(tier_id) ON UPDATE CASCADE,
 tstamp INTEGER NOT NULL UNIQUE
);

CREATE TABLE files (
 file_id INTEGER PRIMARY KEY AUTOINCREMENT,
 dataset_id INTEGER REFERENCES datasets(dataset_id) ON UPDATE CASCADE,
 name TEXT NOT NULL
);
