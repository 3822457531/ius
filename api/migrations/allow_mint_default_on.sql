-- 算力机抢购：默认所有用户可购买，仅由 allow_mint 控制（不再依赖 vip）
ALTER TABLE chatgpt_users
  MODIFY COLUMN `allow_mint` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否允许购买算力机';

UPDATE chatgpt_users
SET allow_mint = 1
WHERE IFNULL(allow_mint, 0) = 0;
