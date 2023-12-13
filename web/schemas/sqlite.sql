--https://stackoverflow.com/questions/18387209/sqlite-syntax-for-creating-table-with-foreign-key

CREATE TABLE cycles (
 cycle_id INTEGER PRIMARY KEY AUTOINCREMENT,
 name TEXT NOT NULL UNIQUE
);

CREATE TABLE beamlines (
 beamline_id INTEGER PRIMARY KEY AUTOINCREMENT,
 name TEXT NOT NULL UNIQUE
);

CREATE TABLE btrs (
 btr_id INTEGER PRIMARY KEY AUTOINCREMENT,
 name TEXT NOT NULL UNIQUE
);

CREATE TABLE samples (
 sample_id INTEGER PRIMARY KEY AUTOINCREMENT,
 name TEXT NOT NULL UNIQUE
);

CREATE TABLE datasets (
 dataset_id INTEGER PRIMARY KEY,
 cycle_id integer REFERENCES cycles(cycle_id) ON UPDATE CASCADE,
 beamline_id integer REFERENCES beamlines(beamline_id) ON UPDATE CASCADE,
 btr_id integer REFERENCES btrs(btr_id) ON UPDATE CASCADE,
 sample_id integer REFERENCES sample(sample_id) ON UPDATE CASCADE,
 tstamp INTEGER NOT NULL UNIQUE
);

CREATE TABLE files (
 file_id INTEGER PRIMARY KEY AUTOINCREMENT,
 dataset_id INTEGER REFERENCES datasets(dataset_id) ON UPDATE CASCADE,
 name TEXT NOT NULL
);

