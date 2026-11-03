-- +goose Up
-- Earlier deployments may have recorded a previous source migration version.
SET @has_token_source = (
    SELECT COUNT(*)
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'telegram_app_tokens'
      AND column_name = 'source'
);
SET @add_token_source_sql = IF(
    @has_token_source = 0,
    'ALTER TABLE telegram_app_tokens ADD COLUMN source ENUM(''telegram'', ''web'') NOT NULL DEFAULT ''telegram'' AFTER user_id',
    'SELECT 1'
);
PREPARE add_token_source_stmt FROM @add_token_source_sql;
EXECUTE add_token_source_stmt;
DEALLOCATE PREPARE add_token_source_stmt;

-- +goose Down
-- Keep this column on rollback; an earlier migration may have created it.
SELECT 1;
