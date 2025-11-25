-- IM Server 数据库初始化脚本

-- 创建数据库
CREATE DATABASE IF NOT EXISTS im_db;

-- 使用数据库
\c im_db;

-- 创建用户表（由 GORM 自动迁移创建）
-- CREATE TABLE IF NOT EXISTS users (...);

-- 初始化测试数据
-- INSERT INTO users (id, username, nickname, password, status) 
-- VALUES ('test_user_1', 'test1', 'Test User 1', '$2a$10$...', 1);

-- 创建索引
-- 注意：(conversation_id, seq) 已经是主键，会自动创建索引
CREATE UNIQUE INDEX IF NOT EXISTS uk_messages_client_msg_id ON messages(client_msg_id);
CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_send_time ON messages(send_time);
CREATE INDEX IF NOT EXISTS idx_conversations_user_target ON conversations(user_id, target_id);
CREATE INDEX IF NOT EXISTS idx_friends_user_friend ON friends(user_id, friend_id);
CREATE INDEX IF NOT EXISTS idx_group_members_group_user ON group_members(group_id, user_id);

-- 完成
SELECT 'Database initialization completed!' AS status;

