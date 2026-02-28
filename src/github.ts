import { execFile } from "node:child_process";

async function ghApi(
  method: string,
  endpoint: string,
  body?: object,
): Promise<string> {
  const args = [
    "api",
    "--method",
    method,
    "-H",
    "Accept: application/vnd.github+json",
    endpoint,
  ];

  if (body) {
    args.push("--input", "-");
  }

  return new Promise((resolve, reject) => {
    const child = execFile(
      "gh",
      args,
      { encoding: "utf-8" },
      (error, stdout, stderr) => {
        if (error) {
          reject(new Error(stderr?.trim() || error.message));
          return;
        }
        resolve(stdout);
      },
    );

    if (body && child.stdin) {
      child.stdin.end(JSON.stringify(body));
    }
  });
}

// --- Types ---

export interface BranchProtectionSettings {
  blockDirectPushes: boolean;
  requirePrReviews: boolean;
  requiredApprovals: number;
  requireStatusChecks: boolean;
  requireConversationResolution: boolean;
  enforceAdmins: boolean;
  allowForcePushes: boolean;
  blockDeletion: boolean;
}

export interface RepoSettings {
  deleteBranchOnMerge: boolean;
  allowSquashMerge: boolean;
  allowMergeCommit: boolean;
  allowRebaseMerge: boolean;
}

export interface SecuritySettings {
  dependabotAlerts: boolean;
  dependabotSecurityUpdates: boolean;
  secretScanning: boolean;
  secretScanningPushProtection: boolean;
}

// --- API Functions ---

export async function getRepoInfo(repo: string): Promise<{
  defaultBranch: string;
  visibility: string;
}> {
  const result = await ghApi("GET", `/repos/${repo}`);
  const data = JSON.parse(result);
  return {
    defaultBranch: data.default_branch as string,
    visibility: data.visibility as string,
  };
}

export async function updateBranchProtection(
  repo: string,
  branch: string,
  settings: BranchProtectionSettings
): Promise<void> {
  const body: Record<string, unknown> = {
    required_status_checks: settings.requireStatusChecks
      ? { strict: true, contexts: [] }
      : null,
    enforce_admins: settings.enforceAdmins,
    required_pull_request_reviews:
      settings.requirePrReviews
        ? {
            required_approving_review_count: settings.requiredApprovals,
            dismiss_stale_reviews: false,
            require_code_owner_reviews: false,
          }
        : settings.blockDirectPushes
          ? // Block direct pushes without requiring reviews by setting
            // required_pull_request_reviews with 0 approvals.
            {
              required_approving_review_count: 0,
              dismiss_stale_reviews: false,
              require_code_owner_reviews: false,
            }
          : null,
    restrictions: null,
    allow_force_pushes: settings.allowForcePushes,
    allow_deletions: !settings.blockDeletion,
    required_conversation_resolution: settings.requireConversationResolution,
    block_creations: false,
    lock_branch: false,
    allow_fork_syncing: false,
  };

  await ghApi("PUT", `/repos/${repo}/branches/${branch}/protection`, body);
}

export async function updateRepoSettings(
  repo: string,
  settings: RepoSettings
): Promise<void> {
  const body: Record<string, unknown> = {
    delete_branch_on_merge: settings.deleteBranchOnMerge,
    allow_squash_merge: settings.allowSquashMerge,
    allow_merge_commit: settings.allowMergeCommit,
    allow_rebase_merge: settings.allowRebaseMerge,
  };

  await ghApi("PATCH", `/repos/${repo}`, body);
}

export async function enableDependabotAlerts(repo: string): Promise<void> {
  await ghApi("PUT", `/repos/${repo}/vulnerability-alerts`);
}

export async function enableDependabotSecurityUpdates(
  repo: string,
): Promise<void> {
  await ghApi("PUT", `/repos/${repo}/automated-security-fixes`);
}

export async function enableSecretScanning(repo: string): Promise<void> {
  await ghApi("PATCH", `/repos/${repo}`, {
    security_and_analysis: {
      secret_scanning: { status: "enabled" },
    },
  });
}

export async function enableSecretScanningPushProtection(
  repo: string,
): Promise<void> {
  await ghApi("PATCH", `/repos/${repo}`, {
    security_and_analysis: {
      secret_scanning_push_protection: { status: "enabled" },
    },
  });
}
