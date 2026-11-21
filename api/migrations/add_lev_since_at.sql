-- 等级分红时间参数：从「升级到当前 lev」起算的天数
ALTER TABLE `chatgpt_users`
  ADD COLUMN `lev_since_at` DATETIME NULL DEFAULT NULL COMMENT '最近一次变更到当前lev的时间' AFTER `lev`;

-- 历史用户：有等级的用注册时间近似，避免分红时间参数从 0 天误判
UPDATE `chatgpt_users`
SET `lev_since_at` = `created_at`
WHERE `lev` > 0 AND `lev_since_at` IS NULL;
