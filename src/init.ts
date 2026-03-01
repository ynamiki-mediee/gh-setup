import * as p from "@clack/prompts";
import { preflight } from "./preflight.ts";
import {
  getRepoInfo,
  updateBranchProtection,
  updateRepoSettings,
  enableDependabotAlerts,
  enableDependabotSecurityUpdates,
  enableSecretScanning,
  enableSecretScanningPushProtection,
  type BranchProtectionSettings,
  type RepoSettings,
  type SecuritySettings,
} from "./github.ts";

type BranchProtectionOption =
  | "blockDirectPushes"
  | "requirePrReviews"
  | "requireStatusChecks"
  | "requireConversationResolution"
  | "enforceAdmins"
  | "allowForcePushes"
  | "blockDeletion";

type RepoOption =
  | "deleteBranchOnMerge"
  | "allowSquashMerge"
  | "allowMergeCommit"
  | "allowRebaseMerge";

type SecurityOption =
  | "dependabotAlerts"
  | "dependabotSecurityUpdates"
  | "secretScanning"
  | "secretScanningPushProtection";

export async function init(): Promise<void> {
  p.intro("gh-setup init");

  // Step 1: Preflight checks & repo confirmation
  const preflightResult = await preflight();
  if (!preflightResult) {
    p.outro("Setup cancelled.");
    process.exitCode = 1;
    return;
  }

  const { repo } = preflightResult;
  p.log.info(`Repository: ${repo}`);

  // Step 2: Branch protection
  let defaultBranch = "main";
  try {
    defaultBranch = (await getRepoInfo(repo)).defaultBranch;
  } catch {
    // fallback to "main"
  }

  const branch = await p.text({
    message: "Branch to protect:",
    initialValue: defaultBranch,
    validate: (v) => (v && v.length > 0 ? undefined : "Branch name is required"),
  });

  if (p.isCancel(branch)) {
    p.outro("Setup cancelled.");
    return;
  }

  const branchProtectionChoices = await p.multiselect<BranchProtectionOption>({
    message: "Branch protection rules:",
    options: [
      { value: "blockDirectPushes", label: "Block direct pushes" },
      { value: "requirePrReviews", label: "Require PR reviews" },
      { value: "requireStatusChecks", label: "Require status checks" },
      {
        value: "requireConversationResolution",
        label: "Require conversation resolution",
      },
      { value: "enforceAdmins", label: "Enforce for admins" },
      { value: "allowForcePushes", label: "Allow force pushes" },
      { value: "blockDeletion", label: "Block branch deletion" },
    ],
    initialValues: [
      "blockDirectPushes",
      "blockDeletion",
    ] as BranchProtectionOption[],
    required: false,
  });

  if (p.isCancel(branchProtectionChoices)) {
    p.outro("Setup cancelled.");
    return;
  }

  let requiredApprovals = 1;
  if (branchProtectionChoices.includes("requirePrReviews")) {
    const approvals = await p.select({
      message: "Required number of approvals:",
      options: [
        { value: 1, label: "1" },
        { value: 2, label: "2" },
        { value: 3, label: "3" },
      ],
    });

    if (p.isCancel(approvals)) {
      p.outro("Setup cancelled.");
      return;
    }

    requiredApprovals = approvals;
  }

  const branchProtection: BranchProtectionSettings = {
    blockDirectPushes: branchProtectionChoices.includes("blockDirectPushes"),
    requirePrReviews: branchProtectionChoices.includes("requirePrReviews"),
    requiredApprovals,
    requireStatusChecks: branchProtectionChoices.includes(
      "requireStatusChecks"
    ),
    requireConversationResolution: branchProtectionChoices.includes(
      "requireConversationResolution"
    ),
    enforceAdmins: branchProtectionChoices.includes("enforceAdmins"),
    allowForcePushes: branchProtectionChoices.includes("allowForcePushes"),
    blockDeletion: branchProtectionChoices.includes("blockDeletion"),
  };

  // Step 3: Repository settings
  const repoChoices = await p.multiselect<RepoOption>({
    message: "Repository settings:",
    options: [
      {
        value: "deleteBranchOnMerge",
        label: "Auto-delete branches after merge",
      },
      { value: "allowSquashMerge", label: "Allow squash merge" },
      { value: "allowMergeCommit", label: "Allow merge commit" },
      { value: "allowRebaseMerge", label: "Allow rebase merge" },
    ],
    initialValues: [
      "deleteBranchOnMerge",
      "allowSquashMerge",
    ] as RepoOption[],
    required: false,
  });

  if (p.isCancel(repoChoices)) {
    p.outro("Setup cancelled.");
    return;
  }

  const mergeOptions: RepoOption[] = [
    "allowSquashMerge",
    "allowMergeCommit",
    "allowRebaseMerge",
  ];
  const hasMergeStrategy = repoChoices.some((c) => mergeOptions.includes(c));
  if (!hasMergeStrategy) {
    p.log.warn("No merge strategy selected — PRs cannot be merged.");
    const proceed = await p.confirm({ message: "Continue anyway?" });
    if (p.isCancel(proceed) || !proceed) {
      p.outro("Setup cancelled.");
      return;
    }
  }

  const repoSettings: RepoSettings = {
    deleteBranchOnMerge: repoChoices.includes("deleteBranchOnMerge"),
    allowSquashMerge: repoChoices.includes("allowSquashMerge"),
    allowMergeCommit: repoChoices.includes("allowMergeCommit"),
    allowRebaseMerge: repoChoices.includes("allowRebaseMerge"),
  };

  // Step 4: Security settings
  const securityChoices = await p.multiselect<SecurityOption>({
    message: "Security features:",
    options: [
      { value: "dependabotAlerts", label: "Dependabot alerts" },
      {
        value: "dependabotSecurityUpdates",
        label: "Dependabot security updates",
      },
      {
        value: "secretScanning",
        label: "Secret scanning",
        hint: "public repos or GHAS",
      },
      {
        value: "secretScanningPushProtection",
        label: "Secret scanning push protection",
        hint: "public repos or GHAS",
      },
    ],
    initialValues: ["dependabotAlerts"] as SecurityOption[],
    required: false,
  });

  if (p.isCancel(securityChoices)) {
    p.outro("Setup cancelled.");
    return;
  }

  const securitySettings: SecuritySettings = {
    dependabotAlerts: securityChoices.includes("dependabotAlerts"),
    dependabotSecurityUpdates: securityChoices.includes(
      "dependabotSecurityUpdates"
    ),
    secretScanning: securityChoices.includes("secretScanning"),
    secretScanningPushProtection: securityChoices.includes(
      "secretScanningPushProtection"
    ),
  };

  // Step 5: Confirmation summary
  const summaryLines: string[] = [
    `Repository: ${repo}`,
    "",
    `Branch protection (${branch}):`,
  ];

  for (const opt of branchProtectionChoices) {
    const labels: Record<BranchProtectionOption, string> = {
      blockDirectPushes: "Block direct pushes",
      requirePrReviews: `Require PR reviews (${requiredApprovals} approval${requiredApprovals > 1 ? "s" : ""})`,
      requireStatusChecks: "Require status checks",
      requireConversationResolution: "Require conversation resolution",
      enforceAdmins: "Enforce for admins",
      allowForcePushes: "Allow force pushes",
      blockDeletion: "Block branch deletion",
    };
    summaryLines.push(`  + ${labels[opt]}`);
  }
  if (branchProtectionChoices.length === 0) {
    summaryLines.push("  (none)");
  }

  summaryLines.push("", "Repository settings:");
  for (const opt of repoChoices) {
    const labels: Record<RepoOption, string> = {
      deleteBranchOnMerge: "Auto-delete branches after merge",
      allowSquashMerge: "Allow squash merge",
      allowMergeCommit: "Allow merge commit",
      allowRebaseMerge: "Allow rebase merge",
    };
    summaryLines.push(`  + ${labels[opt]}`);
  }
  if (repoChoices.length === 0) {
    summaryLines.push("  (none)");
  }

  summaryLines.push("", "Security:");
  for (const opt of securityChoices) {
    const labels: Record<SecurityOption, string> = {
      dependabotAlerts: "Dependabot alerts",
      dependabotSecurityUpdates: "Dependabot security updates",
      secretScanning: "Secret scanning",
      secretScanningPushProtection: "Secret scanning push protection",
    };
    summaryLines.push(`  + ${labels[opt]}`);
  }
  if (securityChoices.length === 0) {
    summaryLines.push("  (none)");
  }

  p.note(summaryLines.join("\n"), "Settings to apply");

  const confirmed = await p.confirm({
    message: "Apply these settings?",
  });

  if (p.isCancel(confirmed) || !confirmed) {
    p.outro("Setup cancelled.");
    return;
  }

  // Step 6: Apply settings
  const tasks: { title: string; task: () => Promise<void> }[] = [];

  if (branchProtectionChoices.length > 0) {
    tasks.push({
      title: `Branch protection (${branch})`,
      task: () => updateBranchProtection(repo, branch, branchProtection),
    });
  }

  if (repoChoices.length > 0) {
    tasks.push({
      title: "Repository settings",
      task: () => updateRepoSettings(repo, repoSettings),
    });
  }

  if (securitySettings.dependabotAlerts) {
    tasks.push({
      title: "Dependabot alerts",
      task: () => enableDependabotAlerts(repo),
    });
  }

  if (securitySettings.dependabotSecurityUpdates) {
    tasks.push({
      title: "Dependabot security updates",
      task: () => enableDependabotSecurityUpdates(repo),
    });
  }

  if (securitySettings.secretScanning) {
    tasks.push({
      title: "Secret scanning",
      task: () => enableSecretScanning(repo),
    });
  }

  if (securitySettings.secretScanningPushProtection) {
    tasks.push({
      title: "Secret scanning push protection",
      task: () => enableSecretScanningPushProtection(repo),
    });
  }

  if (tasks.length === 0) {
    p.outro("No settings to apply.");
    return;
  }

  const s = p.spinner();
  const results: { title: string; success: boolean; error?: string }[] = [];

  for (const t of tasks) {
    s.start(t.title);
    try {
      await t.task();
      s.stop(`${t.title} — done`);
      results.push({ title: t.title, success: true });
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      s.stop(`${t.title} — failed`);
      results.push({ title: t.title, success: false, error: msg });
    }
  }

  // Final summary
  const succeeded = results.filter((r) => r.success);
  const failed = results.filter((r) => !r.success);

  if (failed.length > 0) {
    p.log.warn(
      `${succeeded.length}/${results.length} settings applied. Failures:`
    );
    for (const f of failed) {
      p.log.error(`  ${f.title}: ${f.error}`);
    }
  }

  p.outro(
    failed.length === 0
      ? "Setup complete!"
      : "Setup finished with some errors."
  );
}
