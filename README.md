# SQL Playground

SQL の練習問題を解き、ブラウザ上でクエリ実行と採点ができるプレイグラウンドです。

## 必要なもの

- Go 1.25 以上
- Node.js と npm

## ローカルでの起動方法

バックエンドとフロントエンドを別々のターミナルで起動します。

### 1. バックエンドを起動

リポジトリのルートで実行します。

```sh
go run ./backend/cmd/server
```

デフォルトでは `http://localhost:8080` で起動します。
ポートを変えたい場合は `PORT` を指定します。

```sh
PORT=18080 go run ./backend/cmd/server
```

提出履歴の保存先 SQLite ファイルを変えたい場合は `SUBMISSIONS_DB_PATH` を指定します。

```sh
SUBMISSIONS_DB_PATH=/tmp/sql-playground-submissions.sqlite go run ./backend/cmd/server
```

### 2. フロントエンドを起動

別のターミナルで `frontend` ディレクトリに移動し、依存関係をインストールして起動します。

```sh
cd frontend
npm install
npm run dev
```

デフォルトでは `http://localhost:5173` で起動します。
Vite の開発サーバーは `/api` へのリクエストを `http://localhost:8080` にプロキシします。
バックエンドのポートを変更した場合は `API_TARGET` を指定してください。

```sh
API_TARGET=http://localhost:18080 npm run dev
```

## テストとビルド

バックエンドのテストはリポジトリのルートで実行します。

```sh
go test ./...
```

フロントエンドの型チェックとビルドは `frontend` ディレクトリで実行します。

```sh
cd frontend
npm run build
```

## ファイル構成

```text
.
├── api/                 # API ハンドラーの薄いエントリポイント
├── backend/cmd/server/  # Echo サーバーの起動コード
├── data/                # ローカル用の SQLite データ
├── frontend/            # React + Vite のフロントエンド
│   └── src/
│       ├── api/         # フロントエンドから API を呼び出すクライアント
│       ├── components/  # 画面を構成する React コンポーネント
│       ├── pages/       # ルーティング単位のページ
│       └── utils/       # CSV などのユーティリティ
├── pkg/
│   ├── handler/         # Echo のリクエストハンドラー
│   ├── httpapi/         # API の共通レスポンス型
│   └── service/         # SQL 実行、問題、提出履歴などのドメインロジック
└── tasks/               # SQL 練習問題の JSON 定義
```

## 主なエンドポイント

- `GET /api/me`: 現在のユーザー情報
- `GET /api/tasks`: 問題一覧
- `GET /api/tasks/:number`: 問題詳細
- `POST /api/tasks/:number/submit`: 回答提出
- `POST /api/sqlite/execute`: SQL 実行
- `GET /api/admin/submissions`: 提出履歴一覧
