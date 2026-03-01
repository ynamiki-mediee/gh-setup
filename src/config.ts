import { readFileSync } from "node:fs";
import { parse } from "yaml";
import * as p from "@clack/prompts";

export interface LabelConfig {
  name: string;
  color: string;
  description?: string;
}

export interface MilestonesConfig {
  startDate: string;
  weeks: number;
  dueTime?: string; // e.g. "14:59:59Z" for JST 23:59:59
}

export interface GhSetupConfig {
  milestones?: MilestonesConfig;
  labels?: LabelConfig[];
}

export function loadConfig(): GhSetupConfig | undefined {
  try {
    const content = readFileSync(".gh-setup.yml", "utf-8");
    const parsed = parse(content);

    // yaml.parse returns null for empty/blank files
    if (parsed == null || typeof parsed !== "object") {
      return undefined;
    }

    // Validate labels field if present
    if ("labels" in parsed && !Array.isArray(parsed.labels)) {
      p.log.warn("Invalid config: 'labels' must be an array.");
      return undefined;
    }

    if (Array.isArray(parsed.labels)) {
      for (const label of parsed.labels) {
        if (
          typeof label !== "object" ||
          label == null ||
          typeof label.name !== "string" ||
          typeof label.color !== "string"
        ) {
          p.log.warn(
            "Invalid config: each label must have a string 'name' and 'color'.",
          );
          return undefined;
        }
      }
    }

    // Validate milestones field if present
    if ("milestones" in parsed && parsed.milestones != null) {
      const ms = parsed.milestones;
      if (
        typeof ms !== "object" ||
        typeof ms.startDate !== "string" ||
        typeof ms.weeks !== "number"
      ) {
        p.log.warn(
          "Invalid config: 'milestones' must have a string 'startDate' and a number 'weeks'.",
        );
        return undefined;
      }
    }

    return parsed as GhSetupConfig;
  } catch (e: unknown) {
    if (
      e instanceof Error &&
      "code" in e &&
      (e as NodeJS.ErrnoException).code === "ENOENT"
    ) {
      return undefined;
    }
    p.log.warn(
      `Failed to load .gh-setup.yml: ${e instanceof Error ? e.message : String(e)}`,
    );
    return undefined;
  }
}
