import * as p from "@clack/prompts";
import { preflight } from "./preflight.ts";
import { loadConfig } from "./config.ts";
import {
  listMilestones,
  createMilestone,
  updateMilestone,
} from "./github.ts";

function addDays(date: Date, days: number): Date {
  const result = new Date(date);
  result.setDate(result.getDate() + days);
  return result;
}

function formatDate(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

export async function milestones(): Promise<void> {
  p.intro("gh-setup milestones");

  const preflightResult = await preflight();
  if (!preflightResult) {
    p.outro("Cancelled.");
    process.exitCode = 1;
    return;
  }

  const { repo } = preflightResult;
  p.log.info(`Repository: ${repo}`);

  // Load config or prompt interactively
  const config = loadConfig();
  let startDate: string;
  let weeks: number;

  if (config?.milestones) {
    startDate = config.milestones.startDate;
    weeks = config.milestones.weeks;
    p.log.info(`Config: startDate=${startDate}, weeks=${weeks}`);
  } else {
    const startDateInput = await p.text({
      message: "Start date (first Sunday, YYYY-MM-DD):",
      placeholder: "2026-01-04",
      validate: (v) =>
        v && /^\d{4}-\d{2}-\d{2}$/.test(v)
          ? undefined
          : "Format: YYYY-MM-DD",
    });
    if (p.isCancel(startDateInput)) {
      p.outro("Cancelled.");
      return;
    }

    const weeksInput = await p.text({
      message: "Number of weeks:",
      placeholder: "52",
      validate: (v) => {
        const n = Number(v);
        return Number.isInteger(n) && n > 0
          ? undefined
          : "Enter a positive integer";
      },
    });
    if (p.isCancel(weeksInput)) {
      p.outro("Cancelled.");
      return;
    }

    startDate = startDateInput;
    weeks = Number(weeksInput);
  }

  // Fetch existing milestones
  const s = p.spinner();
  s.start("Fetching existing milestones...");

  let existingMap: Map<string, number>;
  try {
    const existing = await listMilestones(repo);
    existingMap = new Map();
    for (const m of existing) {
      if (m.due_on) {
        const dateKey = m.due_on.slice(0, 10);
        if (existingMap.has(dateKey)) {
          p.log.warn(
            `Duplicate due_on date ${dateKey}: milestone #${m.number} ("${m.title}") overwrites #${existingMap.get(dateKey)}`,
          );
        }
        existingMap.set(dateKey, m.number);
      }
    }
    s.stop(`Found ${existing.length} existing milestones.`);
  } catch (e) {
    s.stop("Failed to fetch milestones.");
    p.log.error(e instanceof Error ? e.message : String(e));
    process.exitCode = 1;
    return;
  }

  // Confirmation
  const confirmed = await p.confirm({
    message: `Create/update ${weeks} weekly milestones starting from ${startDate}?`,
  });
  if (p.isCancel(confirmed) || !confirmed) {
    p.outro("Cancelled.");
    return;
  }

  // Create/update milestones
  let created = 0;
  let updated = 0;
  let failed = 0;
  const failures: { title: string; error: string }[] = [];

  s.start("Creating milestones...");

  for (let i = 0; i < weeks; i++) {
    const weekStart = addDays(new Date(startDate + "T00:00:00"), i * 7);
    const weekEnd = addDays(weekStart, 6); // Saturday
    const weekNum = i + 1;
    const endDateStr = formatDate(weekEnd);
    const title = `Week ${weekNum}: ${endDateStr}`;
    const description = `Period: ${formatDate(weekStart)} - ${endDateStr}`;

    const dueTime = config?.milestones?.dueTime ?? "14:59:59Z";
    const dueOn = `${endDateStr}T${dueTime}`;

    const existingNumber = existingMap.get(endDateStr);
    s.message(`${existingNumber ? "Updating" : "Creating"}: ${title}`);

    try {
      if (existingNumber) {
        await updateMilestone(repo, existingNumber, title, description);
        updated++;
      } else {
        await createMilestone(repo, title, description, dueOn);
        created++;
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      failures.push({ title, error: msg });
      failed++;
    }
  }

  s.stop("Done.");

  for (const f of failures) {
    p.log.error(`  ${f.title}: ${f.error}`);
  }

  p.log.info(`Created: ${created} / Updated: ${updated} / Failed: ${failed}`);
  if (failed > 0) {
    process.exitCode = 1;
  }
  p.outro("Milestones complete!");
}
