#!/bin/bash
# 清空测试数据

echo "清空数据库数据..."

# 连接数据库并清空表
docker exec im-postgres psql -U postgres -d im_db << SQL
-- 清空消息相关表
TRUNCATE TABLE messages CASCADE;
TRUNCATE TABLE message_sequences CASCADE;
TRUNCATE TABLE conversations CASCADE;

-- 保留用户和群组表（因为刚创建的）
-- TRUNCATE TABLE users CASCADE;
-- TRUNCATE TABLE groups CASCADE;
-- TRUNCATE TABLE group_members CASCADE;

SELECT 'Data cleared successfully!' AS result;
SQL

echo "✅ 数据清空完成！"
