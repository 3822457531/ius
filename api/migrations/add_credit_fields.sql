-- 授信额度三字段（独立于可用算力，仅可用于授信报单；不可转账/AI 消费）
-- 与 main.go 启动时的 AddColumn 兜底等价，此文件用于手工建库/运维执行。
ALTER TABLE `chatgpt_users`
  ADD COLUMN `credit_max` DOUBLE NOT NULL DEFAULT 0 COMMENT '最大可用额度 = 自投合计 × 杠杆' AFTER `is_cs`,
  ADD COLUMN `credit_available` DOUBLE NOT NULL DEFAULT 0 COMMENT '当前可用额度（直推投入累积，≤ credit_max；还款不回补）' AFTER `credit_max`,
  ADD COLUMN `credit_debt` DOUBLE NOT NULL DEFAULT 0 COMMENT '已消耗待还额度' AFTER `credit_available`;