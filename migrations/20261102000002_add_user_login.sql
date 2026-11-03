-- +goose Up
-- The login column is provisioned by the shared users schema; this service only
-- ensures the uniqueness constraint needed by credential lookup.
SET @has_login_index = (
    SELECT COUNT(*)
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'users'
      AND index_name = 'users_login_unique'
);
SET @add_login_index_sql = IF(
    @has_login_index = 0,
    'ALTER TABLE users ADD UNIQUE INDEX users_login_unique (login)',
    'SELECT 1'
);
PREPARE add_login_index_stmt FROM @add_login_index_sql;
EXECUTE add_login_index_stmt;
DEALLOCATE PREPARE add_login_index_stmt;

-- +goose Down
-- Leave the index in place because the login column and index may be shared.
SELECT 1;
