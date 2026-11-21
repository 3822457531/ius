-- confirmJihuoEx 复投次数限制（每账号最多 3 次）
ALTER TABLE `chatgpt_users`
  ADD COLUMN `jihuo_ex_reinvest_count` INT NOT NULL DEFAULT 0 COMMENT 'confirmJihuoEx复投次数' AFTER `is_jh`;

-- 历史数据回填：按自定义套餐订单数估算复投次数
UPDATE `chatgpt_users` u
INNER JOIN (
  SELECT user_id, COUNT(*) AS cnt
  FROM `chatgpt_orders`
  WHERE product_id = 0
  GROUP BY user_id
) t ON t.user_id = u.id
SET u.jihuo_ex_reinvest_count = CASE
  WHEN EXISTS (
    SELECT 1 FROM `chatgpt_orders` o
    WHERE o.user_id = u.id AND o.product_id > 0
  ) THEN t.cnt
  ELSE GREATEST(t.cnt - 1, 0)
END
WHERE u.is_jh = 1;
