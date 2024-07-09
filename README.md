# pgautositemap - Discord bot

- [INVITE LINK](https://discord.com/oauth2/authorize?client_id=1260115842582974496&permissions=11280&integration_type=0&scope=applications.commands+bot)
	- 現在開発者 (Ama) のみ招待できるようになっています。

## 機能

- [x] サイトマップの自動生成
	- [x] チャンネルの作成時にサイトマップを自動生成
	- [x] チャンネルの移動時にサイトマップを自動更新
	- [x] チャンネルの削除時にサイトマップを自動更新
	- [x] 既存のサイトマップを読み込み差分更新
	- [ ] 権限設定の複製
		- 現時点では手動で設定してください

## 開発

### 環境変数

設定すべき環境変数は [`.env.example`](.env.example) に記載されています。適当な値を設定して `.env` という名前で保存してください。
各変数の説明はコメントに記載されています。

### セットアップ

1. [`mise`](https://mise.jdx.dev/) をインストールします。
2. `.env`を作成します。
3. `mise` の設定ファイルを信頼します。
   ```bash
   mise trust
   ```
4. 依存パッケージをインストールします。
   ```bash
   mise install
   ```
