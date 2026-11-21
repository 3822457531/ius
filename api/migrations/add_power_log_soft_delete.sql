-- 算力流水软删除（回滚后用户不可见，管理员库内保留）
ALTER TABLE chatgpt_power_logs
    ADD COLUMN IF NOT EXISTS deleted_at DATETIME NULL DEFAULT NULL COMMENT '软删除时间',
    ADD COLUMN IF NOT EXISTS rollback_batch_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '回滚批次号';

CREATE INDEX IF NOT EXISTS idx_power_logs_deleted_at ON chatgpt_power_logs (deleted_at);
CREATE INDEX IF NOT EXISTS idx_power_logs_user_active ON chatgpt_power_logs (user_id, deleted_at, created_at);
