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

function nextSunday(): string {
  const d = new Date();
  const daysUntil = (7 - d.getDay()) % 7 || 7;
  return formatDate(addDays(d, daysUntil));
}

function weeksUntilEndOfYear(startDate: string): number {
  const start = new Date(startDate + "T00:00:00");
  const endOfYear = new Date(start.getFullYear(), 11, 31);
  const diffDays = Math.ceil(
    (endOfYear.getTime() - start.getTime()) / (1000 * 60 * 60 * 24),
  );
  return Math.max(1, Math.ceil(diffDays / 7));
}

function sundayWeekOfYear(date: Date): number {
  const jan1 = new Date(date.getFullYear(), 0, 1);
  const firstSunday = addDays(jan1, (7 - jan1.getDay()) % 7);
  const diffMs = date.getTime() - firstSunday.getTime();
  return Math.floor(diffMs / (7 * 24 * 60 * 60 * 1000)) + 1;
}

function toUtcDueOn(dateStr: string, timezone: string): string {
  const utcRef = new Date(`${dateStr}T23:59:59Z`);
  const utcRepr = new Date(utcRef.toLocaleString("en-US", { timeZone: "UTC" }));
  const tzRepr = new Date(
    utcRef.toLocaleString("en-US", { timeZone: timezone }),
  );
  const offsetMs = tzRepr.getTime() - utcRepr.getTime();
  return new Date(utcRef.getTime() - offsetMs)
    .toISOString()
    .replace(/\.\d{3}Z$/, "Z");
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
  let timezone: string;

  if (config?.milestones) {
    startDate = config.milestones.startDate;
    weeks = config.milestones.weeks;
    timezone = config.milestones.timezone ?? "UTC";
    p.log.info(
      `Config: startDate=${startDate}, weeks=${weeks}, timezone=${timezone}`,
    );
  } else {
    const startDateInput = await p.text({
      message: "Start date (first Sunday, YYYY-MM-DD):",
      initialValue: nextSunday(),
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
      message: "Number of weeks (until end of year):",
      initialValue: String(weeksUntilEndOfYear(startDateInput)),
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

    const timezoneOptions = [
      { value: "Asia/Tokyo", label: "Asia/Tokyo (JST, UTC+9)" },
      { value: "Asia/Shanghai", label: "Asia/Shanghai (CST, UTC+8)" },
      { value: "Asia/Kolkata", label: "Asia/Kolkata (IST, UTC+5:30)" },
      { value: "Europe/Berlin", label: "Europe/Berlin (CET, UTC+1)" },
      { value: "Europe/London", label: "Europe/London (GMT, UTC+0)" },
      { value: "America/New_York", label: "America/New_York (EST, UTC-5)" },
      { value: "America/Chicago", label: "America/Chicago (CST, UTC-6)" },
      {
        value: "America/Los_Angeles",
        label: "America/Los_Angeles (PST, UTC-8)",
      },
      { value: "UTC", label: "UTC" },
    ];

    const systemTz = Intl.DateTimeFormat().resolvedOptions().timeZone;
    const defaultTz =
      timezoneOptions.find((o) => o.value === systemTz)?.value ?? "UTC";

    const tzInput = await p.select({
      message: "Timezone for due dates (Saturday 23:59:59):",
      options: timezoneOptions,
      initialValue: defaultTz,
    });
    if (p.isCancel(tzInput)) {
      p.outro("Cancelled.");
      return;
    }

    startDate = startDateInput;
    weeks = Number(weeksInput);
    timezone = tzInput;
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
    const weekNum = sundayWeekOfYear(weekStart);
    const endDateStr = formatDate(weekEnd);
    const title = `Week ${weekNum}: ${endDateStr}`;
    const description = `Period: ${formatDate(weekStart)} - ${endDateStr}`;

    const dueOn = toUtcDueOn(endDateStr, timezone);

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
