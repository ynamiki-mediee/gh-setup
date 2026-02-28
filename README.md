# gh-setup

Interactive CLI for GitHub repository setup — branch protection, auto-delete, Dependabot & more.

![gh-setup preview](gh-setup-preview.gif)

## Install

```bash
npx gh-setup
```

## Prerequisites

- [GitHub CLI (`gh`)](https://cli.github.com) installed and authenticated
- Node.js 18+

## What it does

`gh-setup` walks you through configuring a GitHub repository interactively:

1. **Branch protection** — block direct pushes, require PR reviews, status checks, etc.
2. **Repository settings** — auto-delete branches, merge strategies
3. **Security** — Dependabot alerts, secret scanning, push protection

All settings are applied via the GitHub API using `gh` CLI.

## Usage

Run inside a git repository with a GitHub remote:

```bash
npx gh-setup
```

The tool auto-detects the repository from `git remote`. If detection fails, you can enter it manually.

## Development

```bash
npm install
npm run build
node dist/index.js
```

## License

MIT
