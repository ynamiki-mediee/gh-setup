import { execFile } from "node:child_process";

async function ghApi(
  method: string,
  endpoint: string,
  body?: object,
  options?: { paginate?: boolean },
): Promise<string> {
  const args = [
    "api",
    "--method",
    method,
    "-H",
    "Accept: application/vnd.github+json",
    endpoint,
  ];

  if (options?.paginate) {
    args.splice(1, 0, "--paginate", "--slurp");
  }

  if (body) {
    args.push("--input", "-");
  }

  const raw: string = await new Promise((resolve, reject) => {
    const child = execFile(
      "gh",
      args,
      { encoding: "utf-8", maxBuffer: 10 * 1024 * 1024 },
      (error, stdout, stderr) => {
        if (error) {
          reject(new Error(stderr?.trim() || error.message));
          return;
        }
        resolve(stdout);
      },
    );

    if (body && child.stdin) {
      // Ignore EPIPE errors when the child process closes stdin early
      child.stdin.on("error", () => {});
      child.stdin.end(JSON.stringify(body));
    }
  });

  // --paginate --slurp wraps each page in an outer array: [[page1...], [page2...]]
  // Flatten so callers get a single flat array.
  if (options?.paginate && raw.length > 0) {
    const parsed = JSON.parse(raw);
    if (Array.isArray(parsed) && parsed.length > 0 && Array.isArray(parsed[0])) {
      return JSON.stringify(parsed.flat());
    }
  }

  return raw;
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

  await ghApi("PUT", `/repos/${repo}/branches/${encodeURIComponent(branch)}/protection`, body);
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

// --- Milestone Types & API ---

export interface Milestone {
  number: number;
  title: string;
  description: string | null;
  due_on: string | null;
  state: string;
}

export async function listMilestones(repo: string): Promise<Milestone[]> {
  const result = await ghApi(
    "GET",
    `/repos/${repo}/milestones?state=all&per_page=100`,
    undefined,
    { paginate: true },
  );
  return JSON.parse(result);
}

export async function createMilestone(
  repo: string,
  title: string,
  description: string,
  dueOn: string,
): Promise<void> {
  await ghApi("POST", `/repos/${repo}/milestones`, {
    title,
    description,
    due_on: dueOn,
  });
}

export async function updateMilestone(
  repo: string,
  number: number,
  title: string,
  description: string,
): Promise<void> {
  await ghApi("PATCH", `/repos/${repo}/milestones/${number}`, {
    title,
    description,
  });
}

// --- Label Types & API ---

export interface Label {
  name: string;
  color: string;
  description: string | null;
}

export async function listLabels(repo: string): Promise<Label[]> {
  const result = await ghApi(
    "GET",
    `/repos/${repo}/labels?per_page=100`,
    undefined,
    { paginate: true },
  );
  return JSON.parse(result);
}

export async function createLabel(
  repo: string,
  name: string,
  color: string,
  description: string,
): Promise<void> {
  await ghApi("POST", `/repos/${repo}/labels`, { name, color, description });
}

export async function updateLabel(
  repo: string,
  name: string,
  color: string,
  description: string,
): Promise<void> {
  await ghApi("PATCH", `/repos/${repo}/labels/${encodeURIComponent(name)}`, {
    color,
    description,
  });
}

export async function deleteLabel(
  repo: string,
  name: string,
): Promise<void> {
  await ghApi("DELETE", `/repos/${repo}/labels/${encodeURIComponent(name)}`);
}
