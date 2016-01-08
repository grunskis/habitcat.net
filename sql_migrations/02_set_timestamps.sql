UPDATE activities SET modified = created WHERE modified IS NULL;

ALTER TABLE activities
      ALTER created DROP DEFAULT;

ALTER TABLE activities
      ALTER created SET NOT NULL;

ALTER TABLE activities
      ALTER modified SET NOT NULL;

CREATE OR REPLACE FUNCTION set_initial_timestamps()
RETURNS TRIGGER AS $$
BEGIN
    NEW.created = now();
    NEW.modified = NEW.created;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER set_initial_timestamps
BEFORE INSERT ON activities
FOR EACH ROW EXECUTE PROCEDURE set_initial_timestamps();
