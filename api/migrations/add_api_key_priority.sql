-- API KEY 优先级：数值越大越优先（生图等多供应商时按此选用）
-- 兼容 MySQL 5.7 / 8.0（不支持 ADD COLUMN IF NOT EXISTS）
-- 若提示 Duplicate column name 'priority'，说明字段已存在，可忽略本脚本。

ALTER TABLE chatgpt_api_keys
    ADD COLUMN priority INT NOT NULL DEFAULT 0 COMMENT '优先级，越大越优先';

-- 索引若已存在会报错，可忽略
-- CREATE INDEX idx_api_keys_type_priority ON chatgpt_api_keys (type, enabled, priority);
