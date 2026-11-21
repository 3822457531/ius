-- 算力机抢购白名单：默认关闭，仅管理员开通后可购买
ALTER TABLE chatgpt_users
  ADD COLUMN `allow_mint` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否允许购买算力机' AFTER `allow_q2`;
