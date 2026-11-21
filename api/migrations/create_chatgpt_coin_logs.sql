-- 矿机单台 SOHA 产出明细（与 user.coin 钱包总账 chatgpt_q2_logs 区分）
CREATE TABLE IF NOT EXISTS `chatgpt_coin_logs` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID',
  `username` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '用户名',
  `type` INT NOT NULL DEFAULT 0 COMMENT '流水类型',
  `amount` DOUBLE NOT NULL DEFAULT 0 COMMENT '变动数量',
  `balance` DOUBLE NOT NULL DEFAULT 0 COMMENT '变动后余额',
  `model` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '业务类型/模型名',
  `remark` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '备注',
  `mark` INT NOT NULL DEFAULT 0 COMMENT '1增加 0减少',
  `order_no` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '矿机订单号',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_coin_logs_user_id` (`user_id`),
  KEY `idx_coin_logs_order_no` (`order_no`),
  KEY `idx_coin_logs_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='SOHA矿机产出明细';
