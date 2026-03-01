import * as p from "@clack/prompts";
import { preflight } from "./preflight.ts";
import { loadConfig, type LabelConfig } from "./config.ts";
import { listLabels, createLabel, updateLabel } from "./github.ts";

export async function labels(): Promise<void> {
  p.intro("gh-setup labels");

  const preflightResult = await preflight();
  if (!preflightResult) {
    p.outro("Cancelled.");
    process.exitCode = 1;
    return;
  }

  const { repo } = preflightResult;
  p.log.info(`Repository: ${repo}`);

  // Load config
  const config = loadConfig();
  if (!config?.labels || config.labels.length === 0) {
    p.log.error("No labels defined in .gh-setup.yml");
    p.outro("Add a 'labels' section to .gh-setup.yml and try again.");
    process.exitCode = 1;
    return;
  }

  const desiredLabels = config.labels;

  // Fetch existing labels
  const s = p.spinner();
  s.start("Fetching existing labels...");

  let existingLabels: { name: string; color: string; description: string | null }[];
  try {
    existingLabels = await listLabels(repo);
    s.stop(`Found ${existingLabels.length} existing labels.`);
  } catch (e) {
    s.stop("Failed to fetch labels.");
    p.log.error(e instanceof Error ? e.message : String(e));
    process.exitCode = 1;
    return;
  }

  // Calculate diff
  const existingMap = new Map(
    existingLabels.map((l) => [l.name.toLowerCase(), l]),
  );
  const toCreate: LabelConfig[] = [];
  const toUpdate: LabelConfig[] = [];
  let unchanged = 0;

  for (const desired of desiredLabels) {
    const existing = existingMap.get(desired.name.toLowerCase());
    if (!existing) {
      toCreate.push(desired);
    } else {
      const colorChanged =
        existing.color !== desired.color.replace(/^#/, "");
      const descChanged =
        (existing.description ?? "") !== (desired.description ?? "");
      if (colorChanged || descChanged) {
        toUpdate.push(desired);
      } else {
        unchanged++;
      }
    }
  }

  // Show diff summary
  if (toCreate.length === 0 && toUpdate.length === 0) {
    p.log.info("All labels are up to date.");
    p.outro("Nothing to do.");
    return;
  }

  const lines: string[] = [];
  if (toCreate.length > 0) {
    lines.push(`Create (${toCreate.length}):`);
    for (const l of toCreate) lines.push(`  + ${l.name} (#${l.color})`);
  }
  if (toUpdate.length > 0) {
    lines.push(`Update (${toUpdate.length}):`);
    for (const l of toUpdate) lines.push(`  ~ ${l.name} (#${l.color})`);
  }
  lines.push(`Unchanged: ${unchanged}`);

  p.note(lines.join("\n"), "Label changes");

  const confirmed = await p.confirm({ message: "Apply these changes?" });
  if (p.isCancel(confirmed) || !confirmed) {
    p.outro("Cancelled.");
    return;
  }

  // Apply changes
  let createdCount = 0;
  let updatedCount = 0;
  let failedCount = 0;
  const failures: { name: string; error: string }[] = [];

  s.start("Syncing labels...");

  for (const l of toCreate) {
    s.message(`Creating: ${l.name}`);
    try {
      await createLabel(
        repo,
        l.name,
        l.color.replace(/^#/, ""),
        l.description ?? "",
      );
      createdCount++;
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      failures.push({ name: l.name, error: msg });
      failedCount++;
    }
  }

  for (const l of toUpdate) {
    s.message(`Updating: ${l.name}`);
    try {
      await updateLabel(
        repo,
        l.name,
        l.color.replace(/^#/, ""),
        l.description ?? "",
      );
      updatedCount++;
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      failures.push({ name: l.name, error: msg });
      failedCount++;
    }
  }

  s.stop("Done.");

  for (const f of failures) {
    p.log.error(`  ${f.name}: ${f.error}`);
  }

  p.log.info(
    `Created: ${createdCount} / Updated: ${updatedCount} / Failed: ${failedCount}`,
  );
  if (failedCount > 0) {
    process.exitCode = 1;
  }
  p.outro("Labels complete!");
}
