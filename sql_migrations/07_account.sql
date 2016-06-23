CREATE TABLE IF NOT EXISTS account (
       id uuid NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
       email text NOT NULL UNIQUE,
       password text NOT NULL,
       created timestamp NOT NULL,
       modified timestamp NOT NULL
);

CREATE TRIGGER update_account_modtime
BEFORE UPDATE ON account
FOR EACH ROW EXECUTE PROCEDURE update_modified_column();

CREATE TRIGGER set_initial_account_timestamps
BEFORE INSERT ON account
FOR EACH ROW EXECUTE PROCEDURE set_initial_timestamps();

ALTER TABLE goal
  ADD COLUMN account_id uuid REFERENCES account (id);

ALTER TABLE habit
  ADD COLUMN account_id uuid REFERENCES account (id);

-- ersatz-analog-edgy-slavish-calling-footrace
INSERT INTO account (email, password) VALUES ('martins@grunskis.com', '$2a$10$oWd9Xh8NOx/nSJYxYKwh3..yVdjPynUYVFgEo7mhPBsoAe24MVyOG');
UPDATE goal SET account_id = (SELECT id FROM account LIMIT 1);
UPDATE habit SET account_id = (SELECT id FROM account LIMIT 1);

ALTER TABLE goal
      ALTER account_id SET NOT NULL;

ALTER TABLE habit
      ALTER account_id SET NOT NULL;
