-- 将历史数据中的 Q2 文案统一为 SOHA（表名/字段名保持不变，仅更新展示文本）

UPDATE chatgpt_q2_logs
SET model = REPLACE(model, 'Q2', 'SOHA'),
    remark = REPLACE(remark, 'Q2', 'SOHA')
WHERE model LIKE '%Q2%' OR remark LIKE '%Q2%';

UPDATE chatgpt_power_logs
SET model = REPLACE(model, 'Q2', 'SOHA'),
    remark = REPLACE(remark, 'Q2', 'SOHA')
WHERE model LIKE '%Q2%' OR remark LIKE '%Q2%';

UPDATE chatgpt_coin_logs
SET model = REPLACE(model, 'Q2', 'SOHA'),
    remark = REPLACE(remark, 'Q2', 'SOHA')
WHERE model LIKE '%Q2%' OR remark LIKE '%Q2%';
