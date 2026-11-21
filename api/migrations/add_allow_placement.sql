-- 大小区节点查看白名单：默认关闭，仅管理员开通后手机端显示
ALTER TABLE chatgpt_users
  ADD COLUMN `allow_placement` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否允许查看大小区节点' AFTER `allow_mint`;
