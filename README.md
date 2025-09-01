# go-subtitles-translation-v2
## go-subtitles-translation

## 主な機能

- **YouTube動画の音声抽出**: YouTube URLから音声データを自動抽出
- **多言語音声認識**: Google Cloud Speech-to-Text APIによる高精度な文字起こし
- **自動翻訳**: 複数言語への自動翻訳機能
- **SRT字幕生成**: タイムスタンプ付きSRT形式ファイルの自動生成
- **リアルタイム処理状況表示**: 処理進捗のリアルタイム監視
- **使用量制限管理**: API使用量の自動追跡と制限管理

## 技術スタック

### バックエンド
- **言語**: Go 1.21+
- **Webフレームワーク**: Gin (github.com/gin-gonic/gin)
- **音声認識**: Google Cloud Speech-to-Text API
- **クラウドストレージ**: Google Cloud Storage
- **CORS対応**: github.com/gin-contrib/cors
- **開発サーバー**: http://localhost:8080 

### フロントエンド
- **フレームワーク**: React 18
- **言語**: TypeScript
- **ビルドツール**: Vite
- **開発サーバー**: http://localhost:5173
- **スタイリング**: CSS3

### 外部サービス
- **Google Cloud Speech-to-Text API**: 音声認識
- **Google Cloud Storage**: 音声ファイル保存
- **翻訳API**: 多言語翻訳処理

### API設計
/video


## ディレクトリ構造

```
go-subtitles-translation-v2/
├── backend/                    # Goバックエンドアプリケーション
│   └── main.go                # メインAPIサーバー　
├── my-react-app/              # Reactフロントエンドアプリケーション
│   ├── public/                # 静的ファイル
│   ├── src/                   # ソースコード
│   │   ├── assets/           # アセットファイル
│   │   ├── App.css           # メインスタイル
│   │   ├── App.tsx           # メインコンポーネント
│   │   ├── index.css         # グローバルスタイル
│   │   ├── main.tsx          # エントリーポイント
│   │   └── vite-env.d.ts     # Vite型定義
│   ├── .gitignore
│   ├── README.md
│   ├── eslint.config.js      # ESLint設定
│   ├── index.html            # HTMLテンプレート
│   ├── package.json          # 依存関係
│   ├── tsconfig.app.json     # TypeScript設定（アプリ）
│   ├── tsconfig.json         # TypeScript設定（メイン）
│   ├── tsconfig.node.json    # TypeScript設定（Node）
│   └── vite.config.ts        # Vite設定
├── test/                      # テストファイル
├── .gitignore                # Git除外設定
├── README.md                 # プロジェクト説明
├── go.mod                    # Go依存関係
├── go.sum                    # Go依存関係チェックサム
├── go.work                   # Goワークスペース
└── go.work.sum              # Goワークスペースチェックサム
```

## 環境構築

### 前提条件

- Go 1.21以上
- Node.js 18以上
- Google Cloud Platform アカウント
- Google Cloud Speech-to-Text API の有効化
- Google Cloud Storage の設定

### はじめる
go mod init モジュール名  
go mod tidy


npm create vite@latest 新しいプロジェクト名 -- --template react-ts <br/>
cd 新しいプロジェクト名  
npm install  
npm run dev  
Local:   http://localhost:5173/  



## 改善
MVCに修正
クリーンアーキDDD検討etc…


