-- 默认算力机档位（可在管理后台 /admin/mint 动态调整）
-- 仅在表为空时插入，避免重复执行

INSERT INTO chatgpt_mints (name, fee, days, power, enabled, sales, sort_num, created_at, updated_at)
SELECT * FROM (
  SELECT '体验算力机' AS name, 0.001 AS fee, 0 AS days, 20000 AS power, 1 AS enabled, 0 AS sales, 0 AS sort_num, NOW() AS created_at, NOW() AS updated_at
  UNION ALL SELECT '1号算力机', 0.001, 0, 50000, 1, 0, 1, NOW(), NOW()
  UNION ALL SELECT '2号算力机', 0.001, 0, 100000, 1, 0, 2, NOW(), NOW()
  UNION ALL SELECT '3号算力机', 0.001, 0, 200000, 1, 0, 3, NOW(), NOW()
  UNION ALL SELECT '4号算力机', 0.001, 0, 400000, 1, 0, 4, NOW(), NOW()
  UNION ALL SELECT '5号算力机', 0.001, 0, 800000, 1, 0, 5, NOW(), NOW()
  UNION ALL SELECT '6号算力机', 0.001, 0, 1600000, 1, 0, 6, NOW(), NOW()
) AS tmp
WHERE NOT EXISTS (SELECT 1 FROM chatgpt_mints LIMIT 1);
