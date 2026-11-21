-- 签到改为「连续签到」模型：记录连续签到天数与最近签到自然日
ALTER TABLE chatgpt_users
  ADD COLUMN `sign_in_streak` INT NOT NULL DEFAULT 0 COMMENT '连续签到天数（0=今日未签到）' AFTER `is_partner`,
  ADD COLUMN `last_sign_in_date` DATE DEFAULT NULL COMMENT '最近一次签到的自然日（用于判断是否断签）' AFTER `sign_in_streak`;