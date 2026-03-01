import { execFileSync } from "node:child_process";
import * as p from "@clack/prompts";

function isGhInstalled(): boolean {
  try {
    execFileSync("gh", ["--version"], { stdio: "ignore" });
    return true;
  } catch {
    return false;
  }
}

function isGhAuthenticated(): boolean {
  try {
    execFileSync("gh", ["auth", "status"], { stdio: "ignore" });
    return true;
  } catch {
    return false;
  }
}

function matchGitHubUrl(url: string): string | null {
  const normalized = url.trimEnd().replace(/\/+$/, "");

  // SSH: git@github.com:owner/repo.git or ssh://git@github.com/owner/repo.git
  const sshMatch = normalized.match(/git@github\.com[:/](.+?)(?:\.git)?$/);
  if (sshMatch) return sshMatch[1];

  // HTTPS: https://github.com/owner/repo.git
  const httpsMatch = normalized.match(/github\.com\/(.+?)(?:\.git)?$/);
  if (httpsMatch) return httpsMatch[1];

  return null;
}

function detectRepo(): string | null {
  // Try "origin" first for efficiency
  try {
    const url = execFileSync("git", ["remote", "get-url", "origin"], {
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "ignore"],
    }).trim();

    const match = matchGitHubUrl(url);
    if (match) return match;
  } catch {
    // origin remote doesn't exist, fall through
  }

  // Fall through to checking all remotes
  try {
    const remotes = execFileSync("git", ["remote"], {
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "ignore"],
    })
      .trim()
      .split("\n")
      .filter((r) => r && r !== "origin");

    for (const remote of remotes) {
      try {
        const url = execFileSync("git", ["remote", "get-url", remote], {
          encoding: "utf-8",
          stdio: ["pipe", "pipe", "ignore"],
        }).trim();

        const match = matchGitHubUrl(url);
        if (match) return match;
      } catch {
        // skip this remote
      }
    }
  } catch {
    // git remote failed
  }

  return null;
}

export interface PreflightResult {
  repo: string;
}

export async function preflight(): Promise<PreflightResult | null> {
  if (!isGhInstalled()) {
    p.log.error(
      "GitHub CLI (gh) is not installed. Install it from https://cli.github.com"
    );
    return null;
  }

  if (!isGhAuthenticated()) {
    p.log.error(
      "GitHub CLI is not authenticated. Run 'gh auth login' first."
    );
    return null;
  }

  const detected = detectRepo();

  if (detected) {
    const useDetected = await p.confirm({
      message: `Use detected repository: ${detected}?`,
    });
    if (p.isCancel(useDetected)) return null;
    if (useDetected) return { repo: detected };
  }

  const input = await p.text({
    message: "Enter repository (owner/repo):",
    placeholder: "owner/repo",
    validate: (v) =>
      v && /^[^/]+\/[^/]+$/.test(v) ? undefined : "Format: owner/repo",
  });
  if (p.isCancel(input)) return null;

  return { repo: input };
}
