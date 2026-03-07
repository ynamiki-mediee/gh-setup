# gh extension 移行計画

## 現状分析

### 現在のアーキテクチャ
- **ツール名**: `gh-setup` (v0.3.0)
- **言語**: TypeScript (ESM, ES2022)
- **ビルド**: `tsup` → `dist/index.js` (single bundle)
- **実行方法**: `npx gh-setup <command>` または `node dist/index.js <command>`
- **依存ライブラリ**: `@clack/prompts`, `yaml`
- **サブコマンド**: `init`, `milestones`, `labels`
- **GitHub API 呼び出し**: すべて `gh api` 経由（`gh` CLI に既に依存）

### 移行に有利な点
- リポジトリ名が既に `gh-setup` (`gh-` プレフィックス済み)
- GitHub API は全て `gh api` 経由で実行しており、`gh` CLI がインストール済み前提
- `tsup` で単一ファイルにバンドル済み
- `process.argv[2]` でサブコマンドをパースしており、`gh setup init` 実行時にそのまま動作する

---

## 移行方針: 2フェーズアプローチ

### フェーズ 1: スクリプトベース extension（最小限の変更で動作）
### フェーズ 2: プリコンパイル extension（バイナリ配布で UX 向上）

---

## フェーズ 1: スクリプトベース extension

### 1-1. ルートに実行可能スクリプト `gh-setup` を作成

`gh extension install` は、リポジトリルートにリポジトリ名と同名の実行可能ファイルを探す。

```bash
#!/usr/bin/env bash
set -euo pipefail

# Node.js の存在確認
if ! command -v node &> /dev/null; then
  echo "Error: Node.js is required but not installed." >&2
  echo "Install Node.js 18+ from https://nodejs.org" >&2
  exit 1
fi

# スクリプト自身のディレクトリを基準に dist/index.js を実行
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
exec node "$SCRIPT_DIR/dist/index.js" "$@"
```

### 1-2. package.json の変更

```diff
 {
   "name": "gh-setup",
+  "type": "module",
   "bin": {
-    "gh-setup": "dist/index.js"
+    "gh-setup": "./dist/index.js"
   },
+  "scripts": {
+    "build": "tsup",
+    "dev": "tsup --watch",
+    "postinstall": "npm run build || true"
+  },
 }
```

### 1-3. `.gitignore` の更新

`dist/` を gitignore から**除外**する（extension install 時にビルドステップが無いため、ビルド済みファイルをコミットする必要がある）。

**または**、`postinstall` スクリプトで自動ビルドする方法もあるが、`gh extension install` は npm install を実行しないため、`dist/` をコミットする方が確実。

```diff
  node_modules/
- dist/
```

### 1-4. dist/ をコミットに含める

```bash
npm run build
git add dist/
git commit -m "Include dist/ for gh extension compatibility"
```

### 1-5. 動作確認

```bash
# ローカルインストール
gh extension install .

# 実行テスト
gh setup --help
gh setup init
gh setup milestones
gh setup labels
```

### 1-6. README.md の更新

インストール方法を追加:

```markdown
## Installation

### As a gh extension (recommended)
```bash
gh extension install ynamiki-mediee/gh-setup
gh setup init
```

### Via npx
```bash
npx gh-setup init
```
```

### 1-7. リポジトリ設定

- リポジトリトピックに `gh-extension` を追加（GitHub の extension 検索に表示される）

---

## フェーズ 2: プリコンパイル extension（Node.js SEA）

ユーザーが Node.js をインストールしていなくても動作するように、プラットフォーム別のバイナリを配布する。

### 2-1. ビルドパイプライン構築

Node.js の Single Executable Application (SEA) 機能を使用してバイナリを生成する。

**対象プラットフォーム:**
| OS | Arch | ファイル名 |
|---------|--------|-------------------------------|
| linux   | amd64  | `gh-setup-linux-amd64`        |
| linux   | arm64  | `gh-setup-linux-arm64`        |
| darwin  | amd64  | `gh-setup-darwin-amd64`       |
| darwin  | arm64  | `gh-setup-darwin-arm64`       |
| windows | amd64  | `gh-setup-windows-amd64.exe`  |
| windows | arm64  | `gh-setup-windows-arm64.exe`  |

### 2-2. SEA ビルドスクリプト作成

`script/build.sh` を作成:

```bash
#!/usr/bin/env bash
set -euo pipefail

# 1. tsup でバンドル（既存の仕組み）
npm run build

# 2. SEA 設定ファイル生成
cat > sea-config.json << 'EOF'
{
  "main": "dist/index.js",
  "output": "sea-prep.blob",
  "disableExperimentalSEAWarning": true
}
EOF

# 3. SEA blob 生成
node --experimental-sea-config sea-config.json

# 4. Node.js バイナリをコピーして blob を注入
cp $(command -v node) gh-setup
npx postject gh-setup NODE_SEA_BLOB sea-prep.blob \
  --sentinel-fuse NODE_SEA_FUSE_fce680ab2cc467b6e072b8b5df1996b2

# クリーンアップ
rm sea-prep.blob sea-config.json
```

### 2-3. GitHub Actions ワークフロー作成

`.github/workflows/release.yml`:

- **トリガー**: タグ `v*` のプッシュ時
- **マトリックスビルド**: 6 プラットフォーム × アーキテクチャ
- **ステップ**:
  1. チェックアウト
  2. Node.js セットアップ
  3. `npm ci`
  4. ビルド (`script/build.sh`)
  5. バイナリに実行権限付与
  6. `gh release create` でリリース作成 & アセットアップロード

**代替案**: `gh-extension-precompile` Action を使用（Go extension 向けだが、カスタムビルドスクリプト対応）

### 2-4. `.gitignore` を元に戻す

プリコンパイル版では `dist/` のコミットが不要になるため、`.gitignore` に `dist/` を戻す。

### 2-5. ルートの `gh-setup` スクリプトを削除

プリコンパイル extension では、GitHub Releases からバイナリが自動ダウンロードされるため、スクリプトは不要。

---

## 実装順序まとめ

| ステップ | 内容 | 変更ファイル |
|------|-----------------------------------|--------------------------------------|
| 1    | ルートに `gh-setup` シェルスクリプト作成 | `gh-setup` (新規) |
| 2    | `.gitignore` から `dist/` を除外 | `.gitignore` |
| 3    | `dist/` をビルド & コミット | `dist/index.js` |
| 4    | README.md にインストール方法追加 | `README.md` |
| 5    | リポジトリトピック `gh-extension` 追加 | (GitHub UI) |
| 6    | ローカルで動作確認 | - |
| 7    | (フェーズ2) `script/build.sh` 作成 | `script/build.sh` (新規) |
| 8    | (フェーズ2) GitHub Actions ワークフロー作成 | `.github/workflows/release.yml` (新規) |
| 9    | (フェーズ2) `.gitignore` に `dist/` を戻す | `.gitignore` |
| 10   | (フェーズ2) ルートスクリプト削除 | `gh-setup` (削除) |
| 11   | (フェーズ2) タグ付け & リリース | - |

---

## 注意事項・リスク

1. **引数パースの互換性**: `gh setup init` → extension は `init` を `$1` として受け取り、Node.js スクリプトに `$@` で渡すため、既存の `process.argv[2]` パースがそのまま動作する。
2. **`@clack/prompts` の TTY 依存**: `gh` extension は通常のターミナルで実行されるため、インタラクティブプロンプトはそのまま動作する。
3. **Node.js SEA の制限**: クロスコンパイルが難しいため、各プラットフォーム用の CI ランナーが必要（GitHub Actions のマトリックスで対応可能）。
4. **`npx` との共存**: extension 化後も `npx gh-setup` での利用は引き続き可能。package.json の `bin` 設定はそのまま。
5. **`dist/` のコミット**: フェーズ1ではビルド成果物をリポジトリに含める必要があり、リポジトリサイズが増加する。フェーズ2で解消される。
