-- 前台用户客服身份：后台可指定，H5 点击「客服」进入接线台
ALTER TABLE chatgpt_users
  ADD COLUMN `is_cs` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否客服' AFTER `is_auditor`;
