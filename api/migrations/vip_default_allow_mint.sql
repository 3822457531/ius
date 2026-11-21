-- VIP 身份用户（chatgpt_users.vip=1）默认开通算力机抢购
-- 购买资格以 vip 字段为准，不要求激活满 1 万算力
UPDATE chatgpt_users
SET allow_mint = 1
WHERE vip = 1
  AND IFNULL(allow_mint, 0) = 0;
