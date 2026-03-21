CREATE TABLE IF NOT EXISTS messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  received_at TEXT NOT NULL,
  source TEXT NOT NULL,
  raw_json TEXT NOT NULL,
  aircraft TEXT,
  flight TEXT,
  message_type TEXT,
  station TEXT
);

CREATE TABLE IF NOT EXISTS parsed_messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  received_at TEXT NOT NULL,
  source TEXT NOT NULL,
  aircraft TEXT,
  flight TEXT,
  message_type TEXT,
  station TEXT,
  timestamp TEXT,
  label TEXT,
  message TEXT,
  text TEXT,
  channel TEXT,
  registration TEXT,
  icao TEXT
);
