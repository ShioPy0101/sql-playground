package service

type SQLiteService struct{}

func NewSQLiteService() *SQLiteService {
	return &SQLiteService{}
}

func (s *SQLiteService) Execute(csvText string, query string) (string, error) {
	// ここに実際の SQLite 実行処理を書く
	// いったん仮で csvText をそのまま返す
	return csvText, nil
}
