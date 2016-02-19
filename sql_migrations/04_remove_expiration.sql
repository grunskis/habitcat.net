DELETE FROM activities WHERE expires IS NOT NULL;

ALTER TABLE activities
  DROP COLUMN IF EXISTS expires;
