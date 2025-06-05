CREATE TABLE IF NOT EXISTS measurements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    device_id INTEGER NOT NULL,
    timestamp DATETIME NOT NULL,
    temperature REAL,
    humidity REAL,
    co2 REAL,
    voc REAL,
    pm25 REAL,
    score REAL,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);
CREATE INDEX idx_measurements_deleted_at ON measurements(deleted_at);
CREATE INDEX idx_measurements_timestamp ON measurements(timestamp);
CREATE INDEX idx_measurements_device_id ON measurements(device_id);
