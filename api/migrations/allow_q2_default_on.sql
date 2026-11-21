-- 新用户默认允许 SOHA 交易；存量未开通用户一并开启（可抢购算力机不再被 SOHA 权限拦住）
ALTER TABLE chatgpt_users
  MODIFY COLUMN `allow_q2` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否允许SOHA交易';

UPDATE chatgpt_users
SET allow_q2 = 1
WHERE IFNULL(allow_q2, 0) = 0;
