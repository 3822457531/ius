-- 授信最大额度历史保底：避免「单笔≥1万才计入」新规则把旧 Max 一次性打成 0
ALTER TABLE `chatgpt_users`
  ADD COLUMN `credit_max_floor` DOUBLE NOT NULL DEFAULT 0
    COMMENT '投入侧保底基数（不含报单消耗）；历史额度保底，仅可被新规则自投抬升'
    AFTER `credit_debt`;
