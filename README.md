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

Admin とイベント作成を利用する場合は Basic 認証情報も設定します。

```sh
ADMIN_USERNAME=admin ADMIN_PASSWORD=secret go run ./backend/cmd/server
```

Admin の `/admin` でイベント名、slug、開始・終了日時、既存課題と表示順を指定すると、参加URL `/events/:slug` が作成されます。イベント情報と提出は通常の提出履歴と同じ SQLite / PostgreSQL に保存され、通常提出は `event_id = NULL` のまま扱われます。

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

イベント画面とAdminのイベント状況を定期更新する場合は、再取得間隔を秒数で指定します。未設定または `0` の場合は自動更新しません。

```sh
VITE_EVENT_REFRESH_INTERVAL_SECONDS=5 npm run dev
```

Viteの環境変数はフロントエンドのビルド時に埋め込まれるため、デプロイ環境で変更した場合は再ビルドしてください。

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
- `GET /api/events/:slug`: イベント情報・参加状況・課題一覧
- `POST /api/events/:slug/join`: 表示名を登録してイベントへ参加
- `GET /api/events/:slug/tasks/:number`: イベント内の問題詳細
- `POST /api/events/:slug/tasks/:number/submit`: イベント内で回答提出
- `POST /api/sqlite/execute`: SQL 実行
- `GET /api/admin/submissions`: 提出履歴一覧
- `GET, POST /api/admin/events`: イベント一覧・作成
- `GET /api/admin/events/:slug`: イベント別の参加者・提出履歴
