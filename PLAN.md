# gh-setup Go リライト計画

TypeScript → Go フルリライト。Cobra CLI + go-gh + charmbracelet/huh。

---

## 技術スタック

| 役割 | ライブラリ |
|------|-----------|
| CLI フレームワーク | `github.com/spf13/cobra` |
| GitHub API | `github.com/cli/go-gh/v2` |
| 対話プロンプト | `github.com/charmbracelet/huh` |
| YAML パース | `gopkg.in/yaml.v3` |
| カラー出力 | `github.com/charmbracelet/lipgloss` |
| リリース | `gh-extension-precompile` Action |

---

## プロジェクト構成

```
gh-setup/
├── main.go
├── go.mod
├── go.sum
├── cmd/
│   ├── root.go          # Cobra root command
│   ├── init.go          # gh setup init
│   ├── milestones.go    # gh setup milestones
│   └── labels.go        # gh setup labels
├── internal/
│   ├── github/
│   │   └── client.go    # go-gh ベースの API クライアント
│   ├── config/
│   │   └── config.go    # .gh-setup.yml パーサー
│   ├── prompt/
│   │   └── prompt.go    # huh ベースの対話 UI ラッパー
│   └── preflight/
│       └── preflight.go # gh CLI / auth チェック、リポジトリ検出
├── .github/
│   └── workflows/
│       └── release.yml  # gh-extension-precompile
├── .goreleaser.yml      # (precompile Action が自動生成するが、カスタムも可)
└── README.md
```

---

## チーム構成 & 並行作業

### チーム A: 基盤 (Foundation)

**担当範囲**: プロジェクト初期化、CLI骨格、共通基盤

| # | タスク | 成果物 | 依存 |
|---|--------|--------|------|
| A1 | Go モジュール初期化 (`go mod init`) | `go.mod`, `main.go` | なし |
| A2 | Cobra root command + サブコマンド骨格 | `cmd/root.go`, `cmd/init.go`, `cmd/milestones.go`, `cmd/labels.go` | A1 |
| A3 | go-gh API クライアント実装 | `internal/github/client.go` | A1 |
| A4 | `.gh-setup.yml` パーサー実装 | `internal/config/config.go` | A1 |
| A5 | preflight チェック (gh CLI存在確認、認証確認、リポジトリ検出) | `internal/preflight/preflight.go` | A3 |
| A6 | huh ベースのプロンプトヘルパー | `internal/prompt/prompt.go` | A1 |

**完了条件**: `go build` が通り、`gh-setup --help` でサブコマンド一覧が表示される。API クライアントとプロンプトヘルパーが他チームから利用可能。

---

### チーム B: init コマンド

**担当範囲**: `gh setup init` のフル実装

**依存**: A2 (Cobra骨格), A3 (APIクライアント), A5 (preflight), A6 (プロンプト)

| # | タスク | 詳細 |
|---|--------|------|
| B1 | ブランチ保護ルール設定 UI | huh multiselect で7つの保護オプションを選択。approvals数は select (1-3) |
| B2 | リポジトリ設定 UI | マージ戦略4択 + auto-delete。選択ゼロ時の警告 |
| B3 | セキュリティ設定 UI | Dependabot alerts/updates, secret scanning, push protection |
| B4 | 設定適用ロジック | API呼び出し。spinner でリアルタイムフィードバック。成功/失敗カウント |
| B5 | サマリー表示 & 確認フロー | 適用前に設定内容を note 表示 → confirm → 適用 |

**API エンドポイント**:
- `PUT /repos/{owner}/{repo}/branches/{branch}/protection`
- `PATCH /repos/{owner}/{repo}` (merge settings)
- `PUT /repos/{owner}/{repo}/vulnerability-alerts`
- `PUT /repos/{owner}/{repo}/automated-security-fixes`
- `PATCH /repos/{owner}/{repo}` (security_and_analysis)

**完了条件**: `gh setup init` でインタラクティブにリポジトリを設定でき、TypeScript版と同じ機能が動作する。

---

### チーム C: milestones + labels コマンド

**担当範囲**: `gh setup milestones` と `gh setup labels` の実装

**依存**: A2 (Cobra骨格), A3 (APIクライアント), A4 (config), A6 (プロンプト)

| # | タスク | 詳細 |
|---|--------|------|
| C1 | milestones: 対話プロンプト | 開始日入力 (デフォルト: 次の日曜)、週数、タイムゾーン選択 (9プリセット) |
| C2 | milestones: 生成ロジック | ISO週番号計算、タイムゾーン対応 due date (土曜 23:59:59 → UTC変換) |
| C3 | milestones: API 連携 | 既存マイルストーン取得 → due_on マッチで更新 or 新規作成。重複警告 |
| C4 | labels: diff 計算 | 既存ラベル取得 → config と比較 (大文字小文字無視)。create/update/unchanged 分類 |
| C5 | labels: 適用ロジック | create/update API呼び出し。per-label フィードバック。成功/失敗カウント |

**API エンドポイント**:
- `GET /repos/{owner}/{repo}/milestones?state=all&per_page=100` (paginated)
- `POST /repos/{owner}/{repo}/milestones`
- `PATCH /repos/{owner}/{repo}/milestones/{number}`
- `GET /repos/{owner}/{repo}/labels?per_page=100` (paginated)
- `POST /repos/{owner}/{repo}/labels`
- `PATCH /repos/{owner}/{repo}/labels/{name}`

**完了条件**: 両コマンドが `.gh-setup.yml` からの読み込みとインタラクティブモードの両方で動作する。

---

### チーム D: リリース & CI/CD

**担当範囲**: ビルド・リリースパイプライン

**依存**: A1 (Go モジュール初期化完了後に着手可能)

| # | タスク | 詳細 |
|---|--------|------|
| D1 | `.github/workflows/release.yml` 作成 | `gh-extension-precompile` Action 使用。タグ `v*` トリガー |
| D2 | `.github/workflows/ci.yml` 作成 | PR / push 時に `go build`, `go vet`, `go test` |
| D3 | `.goreleaser.yml` 作成 (オプション) | カスタムビルド設定が必要な場合 |
| D4 | README.md 更新 | インストール方法、使い方を Go 版に合わせて更新 |
| D5 | TypeScript 関連ファイル削除 | `src/`, `package.json`, `tsconfig.json`, `tsup.config.ts`, `node_modules/`, `dist/` |

**完了条件**: `v*` タグ push で 6 プラットフォームのバイナリが GitHub Releases に自動公開される。

---

## 並行作業タイムライン

```
Week 1:
  チーム A: [A1]→[A2, A3, A4, A6 並行]→[A5]
  チーム D: [D1, D2 並行] (A1完了後)

Week 2:
  チーム B: [B1, B2, B3 並行]→[B4]→[B5]  (A完了後)
  チーム C: [C1, C2 並行]→[C3] + [C4]→[C5]  (A完了後)
  チーム D: [D3, D4]

Week 3:
  全チーム: 結合テスト → バグ修正 → [D5] TypeScript削除 → v1.0.0 タグ
```

---

## TypeScript → Go 対応表

| TypeScript | Go |
|-----------|-----|
| `@clack/prompts` multiselect | `huh.NewMultiSelect()` |
| `@clack/prompts` select | `huh.NewSelect()` |
| `@clack/prompts` text | `huh.NewInput()` |
| `@clack/prompts` confirm | `huh.NewConfirm()` |
| `@clack/prompts` spinner | `huh.NewSpinner()` |
| `@clack/prompts` note | `lipgloss` でカスタム表示 |
| `yaml` パッケージ | `gopkg.in/yaml.v3` |
| `child_process.execFile("gh")` | `go-gh` の `api.NewRESTClient()` |
| `Intl.DateTimeFormat` (TZ) | `time.LoadLocation()` + `time.In()` |
| `process.exit(1)` | `os.Exit(1)` or Cobra の `RunE` で error return |

---

## .gh-setup.yml スキーマ (変更なし)

```yaml
milestones:
  startDate: "2025-01-05"   # YYYY-MM-DD (日曜日)
  weeks: 52
  timezone: "Asia/Tokyo"     # オプション

labels:
  - name: "bug"
    color: "d73a4a"
    description: "Something isn't working"
  - name: "enhancement"
    color: "a2eeef"
    description: "New feature or request"
```

---

## リスク & 注意事項

1. **huh の TTY 要件**: `gh` extension はターミナルで実行されるため問題なし。CI で `--no-interactive` フラグを追加検討
2. **go-gh のページネーション**: REST client には自動ページネーションがないため、手動で `page` パラメータをループする必要がある
3. **タイムゾーン処理**: Go の `time` パッケージは IANA TZ データベースに依存。Windows では `time/tzdata` の embed が必要
4. **クロスコンパイル**: `gh-extension-precompile` が `GOOS`/`GOARCH` を自動設定するため、特別な対応不要
5. **後方互換性**: `.gh-setup.yml` のスキーマは変更しない。既存ユーザーの設定ファイルがそのまま動作すること
