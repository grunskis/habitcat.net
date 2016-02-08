CREATE TYPE period_type AS ENUM ('day', 'week', 'month', 'quarter', 'year');

CREATE TABLE IF NOT EXISTS habit (
       id uuid NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
       description text NOT NULL,
       points integer NOT NULL,
       period period_type NOT NULL,
       start date NOT NULL,
       created timestamp NOT NULL,
       modified timestamp NOT NULL
);

CREATE TABLE IF NOT EXISTS habit_progress (
       habit_id uuid REFERENCES habit (id),
       delta integer NOT NULL,
       created timestamp NOT NULL DEFAULT current_timestamp
);

CREATE TRIGGER update_habit_modtime
BEFORE UPDATE ON habit
FOR EACH ROW EXECUTE PROCEDURE update_modified_column();

CREATE TRIGGER set_initial_habit_timestamps
BEFORE INSERT ON habit
FOR EACH ROW EXECUTE PROCEDURE set_initial_timestamps();
