package service

import (
	"database/sql"
	"os"
)

type SQLiteService struct{}

func NewSQLiteService() *SQLiteService {
	return &SQLiteService{}
}

func (s *SQLiteService) Execute(csvText string, query string) (string, error) {

	// 一時ファイルを作成
	tmpFile, err := os.CreateTemp("", "sqlite-playground-*.sqlite")
	if err != nil {
		return "", err
	}

	dbPath := tmpFile.Name()
	tmpFile.Close()

	defer os.Remove(dbPath)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return "", err
	}
	defer db.Close()

	// 1. csvText からテーブル作成
	// 2. query を実行
	// 3. 結果を CSV 文字列にして返す

	return "", nil
}
