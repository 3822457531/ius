-- SOHA 指导价基准从 0.001 调整为 0.0001（历史 K 线同比例缩放）
UPDATE chatgpt_price_histories
SET price = price * 0.1
WHERE price > 0;
