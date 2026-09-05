CREATE TABLE IF NOT EXISTS services (
    name TEXT PRIMARY KEY,
    container TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS service_actions (
    service_name TEXT NOT NULL REFERENCES services(name) ON DELETE CASCADE,
    action TEXT NOT NULL,
    PRIMARY KEY (service_name, action)
);
