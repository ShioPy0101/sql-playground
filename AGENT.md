# SQL Playground 開発メモ

## 入力形式

SQLite プレイグラウンドは CSV と SQL の2種類の入力形式を扱う。

- CSV は従来どおり、単一テーブルまたは `# table:<name>`、`-- table:<name>`、`[name]` で区切った複数テーブルを読み込む。
- SQL は SQLite の DDL と DML を使い、`CREATE TABLE` と `INSERT INTO ... VALUES ...` でスキーマとデータを定義する。主キー、UNIQUE、外部キー、およびそれぞれの複合キーを利用できる。
- 形式固有の処理は `pkg/service/helper/database_input.go` に集約し、どちらも一時 SQLite DB に変換してから共通のクエリ実行・結果描画へ渡す。
- SQL 入力の初期化はトランザクション内で行い、外部キー制約を有効にする。途中で失敗した場合は全体をロールバックする。

## URL と共有 state

- 選択中の形式は検索パラメータ `input=csv` または `input=sql` に保存する。`mode` は使用しない。
- `input` がない既存URLは後方互換性のため CSV として扱う。
- 共有用の `#state=` には CSV入力、SQL入力、選択形式、実行するSQL、dialectを保存する。
- CSVとSQLの入力内容は別々に保持し、形式切り替え時には自動変換しない。
- URL stateを変更する場合は、直接アクセスに加えてブラウザの戻る・進む操作も確認する。

## 後方互換性

- APIで `csv` だけを送る既存クライアントと、形式情報を持たない既存の共有 stateをCSV入力として扱う。
- CSV用の既存サービスメソッドは残し、新しい共通入力メソッドへ委譲する。
