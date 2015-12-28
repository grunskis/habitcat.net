CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.modified = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TABLE IF NOT EXISTS activities (
       id uuid NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
       description text NOT NULL,
       points_done integer NOT NULL DEFAULT 0,
       points_total integer NOT NULL,
       created timestamp DEFAULT current_timestamp,
       modified timestamp
);

CREATE TRIGGER update_activity_modtime
BEFORE UPDATE ON activities
FOR EACH ROW EXECUTE PROCEDURE update_modified_column();
