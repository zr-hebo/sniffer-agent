package db

import (
	"database/sql"
)

// CloseConnection 关闭数据库连接
func CloseConnection(conn *sql.DB) (err error) {
	err = conn.Close()
	if err != nil {
		return err
	}

	_, err = GetMySQLConnInfo(conn)
	return
}
