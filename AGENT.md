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

## 性能計測

- 通常の提出は `POST /api/tasks/:number/submit`、性能計測は `POST /api/tasks/:number/benchmark` とし、両者を混在させない。
- `TaskService` は採点と提出保存だけを担当し、大量データ生成を行わない。性能計測は `BenchmarkService` の専用一時SQLite DBだけで行う。
- benchmark は task JSON で明示的に有効化された課題だけを対象とし、各 rowCount で独立したDBを作成して必ず破棄する。
- benchmark は既定で正解時のみ起動する。インデックス差を不正解時にも教材として示す課題だけ `benchmark.runOnFailed: true` を明示する。
- 課題固有のデータ分布は `benchmark.dataset` または依存順に並べた `benchmark.datasets` のテーブル名・列名・`row_number` を使うSQLite式で宣言する。新しい課題のために `BenchmarkService` へテーブル別分岐を追加しない。
- datasetの各列には `type` を必須で指定し、`INTEGER`, `REAL`, `TEXT`, `BLOB` のみを使う。SQL問題の計測用テーブルはこの型定義から生成し、投入式も指定型へ変換する。
- ある段階が失敗したら、成功済みの結果を残して後続の大きな段階を実行しない。benchmark の失敗を採点結果へ反映しない。
- benchmark全体には未指定でも5秒のtimeoutを適用する。task JSONの `benchmark.timeoutMs` で短縮でき、`SQLITE_BENCHMARK_TIMEOUT_MS` をサーバー側の上限とする。
