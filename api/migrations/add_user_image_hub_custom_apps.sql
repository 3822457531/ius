ALTER TABLE `chatgpt_users`
  ADD COLUMN `image_hub_custom_apps_json` LONGTEXT NULL COMMENT '图片导航页自定义应用 JSON'
  AFTER `chat_config_json`;
