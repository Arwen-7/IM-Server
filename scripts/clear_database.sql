-- =============================================
-- IM Server - 清空数据库脚本
-- =============================================
-- 警告：此脚本会删除所有数据，请谨慎使用！
-- 执行方式：psql -h localhost -p 5432 -U imserver -d im_db -f clear_database.sql

BEGIN;

-- 1. 清空消息相关表
TRUNCATE TABLE messages CASCADE;
TRUNCATE TABLE message_sequences CASCADE;
TRUNCATE TABLE message_read_receipts CASCADE;

-- 2. 清空会话表
TRUNCATE TABLE conversations CASCADE;

-- 3. 清空用户相关表（保留用户账号，但可以选择清空）
-- TRUNCATE TABLE users CASCADE;  -- 如果需要清空用户，取消注释
TRUNCATE TABLE user_sessions CASCADE;
TRUNCATE TABLE online_status CASCADE;

-- 4. 清空好友关系表
TRUNCATE TABLE friends CASCADE;
TRUNCATE TABLE friend_requests CASCADE;

-- 5. 清空群组相关表（如果有的话）
-- TRUNCATE TABLE groups CASCADE;
-- TRUNCATE TABLE group_members CASCADE;

COMMIT;

SELECT '✅ 数据库清空完成！' AS status;

