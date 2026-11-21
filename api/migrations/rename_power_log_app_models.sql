-- 算力明细 model 统一为应用展示名（历史数据）
-- AI音乐 / AI-图片 / AI-视频 / AI-PPT / AI-Logo
-- 执行前建议先备份：CREATE TABLE chatgpt_power_logs_bak_YYYYMMDD AS SELECT * FROM chatgpt_power_logs;

-- 预览将影响的行数（可选）
-- SELECT model, COUNT(*) AS cnt FROM chatgpt_power_logs GROUP BY model ORDER BY cnt DESC;

-- 音乐
UPDATE chatgpt_power_logs
SET model = 'AI音乐'
WHERE model IN ('音乐生成', '音乐', 'Music', 'music', 'suno');

-- 图片（含历史 gpt-image / mid-journey / gemini-mj 等）
UPDATE chatgpt_power_logs
SET model = 'AI-图片'
WHERE model IN (
    'gpt-image-2',
    'gpt-image-2-all',
    'GPT-Image2',
    'GPT-Image-2',
    'gemini-mj',
    'mid-journey',
    'MidJourney',
    'stable-diffusion',
    'dall-e-3',
    '图片生成',
    'AI图片'
)
   OR model LIKE 'gpt-image%'
   OR model LIKE 'GPT-Image%';

-- Logo（与普通图片区分）
UPDATE chatgpt_power_logs
SET model = 'AI-Logo'
WHERE model IN ('ai-logo', 'AI Logo', 'AI-logo', 'logo');

-- 视频
UPDATE chatgpt_power_logs
SET model = 'AI-视频'
WHERE model IN ('sd2', 'SD2', 'sora2', 'Sora2', '视频生成', 'AI视频', 'video');

-- PPT
UPDATE chatgpt_power_logs
SET model = 'AI-PPT'
WHERE model IN ('PPT', 'ppt', 'Ppt', 'AI PPT', 'AI-ppt');

-- 执行后核对
-- SELECT model, COUNT(*) AS cnt FROM chatgpt_power_logs GROUP BY model ORDER BY cnt DESC;
