CREATE TABLE IF NOT EXISTS devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    name TEXT NOT NULL,
    ip_address TEXT,
    device_type TEXT,
    serial_number TEXT NOT NULL,
    last_seen DATETIME
);
CREATE INDEX idx_devices_deleted_at ON devices(deleted_at);
CREATE UNIQUE INDEX idx_devices_serial_number ON devices(serial_number);
