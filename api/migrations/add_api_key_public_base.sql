-- API KEY 公网域名：图生图相对路径拼接（如 /static/upload/... → https://域名/static/upload/...）
-- 兼容 MySQL 5.7 / 8.0。若提示 Duplicate column name 'public_base'，说明字段已存在可忽略。

ALTER TABLE chatgpt_api_keys
    ADD COLUMN public_base VARCHAR(255) NOT NULL DEFAULT '' COMMENT '公网域名，图生图相对路径拼接';
