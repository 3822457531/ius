-- 音乐任务：歌词与时长（用于播放器同步滚动歌词）
ALTER TABLE `chatgpt_music_jobs`
  ADD COLUMN `lyrics` LONGTEXT NULL COMMENT '歌词全文' AFTER `image_url`,
  ADD COLUMN `duration` DOUBLE NULL DEFAULT 0 COMMENT '音频时长(秒)' AFTER `lyrics`;
