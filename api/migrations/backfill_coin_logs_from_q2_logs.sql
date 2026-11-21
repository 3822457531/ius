-- 将历史 SOHA 产出流水同步到 chatgpt_coin_logs（矿机单台产出明细）
INSERT INTO chatgpt_coin_logs (user_id, username, type, amount, balance, model, remark, mark, created_at, order_no)
SELECT q.user_id, q.username, q.type, q.amount, q.balance, q.model, q.remark, q.mark, q.created_at, q.order_no
FROM chatgpt_q2_logs q
LEFT JOIN chatgpt_coin_logs c ON c.user_id = q.user_id AND c.order_no = q.order_no AND c.amount = q.amount AND c.created_at = q.created_at
WHERE q.order_no <> '' AND q.mark = 1 AND c.id IS NULL;
