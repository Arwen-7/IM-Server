-- 创建群组相关表

-- 群组表
CREATE TABLE IF NOT EXISTS `groups` (
  `group_id` VARCHAR(64) PRIMARY KEY COMMENT '群组ID',
  `group_name` VARCHAR(255) NOT NULL COMMENT '群组名称',
  `face_url` VARCHAR(500) DEFAULT '' COMMENT '群组头像',
  `owner_user_id` VARCHAR(64) NOT NULL COMMENT '群主ID',
  `member_count` INT DEFAULT 0 COMMENT '成员数量',
  `introduction` TEXT DEFAULT '' COMMENT '群组简介',
  `notification` TEXT DEFAULT '' COMMENT '群公告',
  `extra` TEXT DEFAULT '' COMMENT '额外信息',
  `status` TINYINT DEFAULT 0 COMMENT '状态（0=正常，1=已解散）',
  `create_time` BIGINT NOT NULL COMMENT '创建时间（毫秒时间戳）',
  `update_time` BIGINT NOT NULL COMMENT '更新时间（毫秒时间戳）',
  INDEX idx_owner (`owner_user_id`),
  INDEX idx_create_time (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='群组表';

-- 群成员表
CREATE TABLE IF NOT EXISTS `group_members` (
  `id` BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '自增ID',
  `group_id` VARCHAR(64) NOT NULL COMMENT '群组ID',
  `user_id` VARCHAR(64) NOT NULL COMMENT '用户ID',
  `nickname` VARCHAR(255) DEFAULT '' COMMENT '群内昵称',
  `role` TINYINT DEFAULT 0 COMMENT '角色（0=普通成员，1=管理员，2=群主）',
  `join_time` BIGINT NOT NULL COMMENT '加入时间（毫秒时间戳）',
  UNIQUE KEY uk_group_user (`group_id`, `user_id`),
  INDEX idx_group (`group_id`),
  INDEX idx_user (`user_id`),
  INDEX idx_role (`role`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='群成员表';

