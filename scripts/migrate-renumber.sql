-- Migration renumbering: update schema_migrations version from old to new
-- Run this on any existing database that was at version 36
-- This updates the golang-migrate version tracker to match the new 1000+ numbering

-- If current version is 36 (the old last migration), update to 1009 (the new last migration)
UPDATE schema_migrations SET version = 1009 WHERE version = 36;

-- Verify
SELECT * FROM schema_migrations;
