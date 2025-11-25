package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 数据库连接信息
	dsn := "host=localhost port=5432 user=imserver password=imserver123 dbname=im_db sslmode=disable"

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}

	fmt.Println("🔍 验证 messages 表结构...")
	fmt.Println()

	// 查询主键
	var pks []struct {
		ColumnName string `gorm:"column:column_name"`
	}
	db.Raw(`
		SELECT a.attname AS column_name
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		WHERE i.indrelid = 'messages'::regclass AND i.indisprimary
		ORDER BY a.attnum
	`).Scan(&pks)

	fmt.Println("✅ 主键字段:")
	for _, pk := range pks {
		fmt.Printf("   - %s\n", pk.ColumnName)
	}

	// 查询索引
	var indexes []struct {
		IndexName string `gorm:"column:indexname"`
		IndexDef  string `gorm:"column:indexdef"`
	}
	db.Raw(`
		SELECT indexname, indexdef
		FROM pg_indexes
		WHERE tablename = 'messages' AND schemaname = CURRENT_SCHEMA()
		ORDER BY indexname
	`).Scan(&indexes)

	fmt.Println("\n✅ 索引列表:")
	for _, idx := range indexes {
		fmt.Printf("   - %s\n     %s\n", idx.IndexName, idx.IndexDef)
	}

	// 验证唯一索引
	var uniqueIndexes []struct {
		IndexName string `gorm:"column:indexname"`
	}
	db.Raw(`
		SELECT i.relname AS indexname
		FROM pg_index ix
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_class t ON t.oid = ix.indrelid
		WHERE t.relname = 'messages' 
		  AND t.relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = CURRENT_SCHEMA())
		  AND ix.indisunique = true
		  AND NOT ix.indisprimary
	`).Scan(&uniqueIndexes)

	fmt.Println("\n✅ 唯一索引:")
	for _, idx := range uniqueIndexes {
		fmt.Printf("   - %s\n", idx.IndexName)
	}

	fmt.Println("\n✅ 验证完成！")
	fmt.Println("\n📊 设计说明：")
	fmt.Println("   • 主键: (conversation_id, seq)")
	fmt.Println("   • 唯一索引: (conversation_id, client_msg_id) - 会话内幂等")
	fmt.Println("   • 查询模式: 所有查询都基于主键，性能最优")
}

