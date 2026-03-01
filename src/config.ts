import { readFileSync } from "node:fs";
import { parse } from "yaml";

export interface LabelConfig {
  name: string;
  color: string;
  description?: string;
}

export interface MilestonesConfig {
  startDate: string;
  weeks: number;
}

export interface GhSetupConfig {
  milestones?: MilestonesConfig;
  labels?: LabelConfig[];
}

export function loadConfig(): GhSetupConfig | undefined {
  try {
    const content = readFileSync(".gh-setup.yml", "utf-8");
    return parse(content) as GhSetupConfig;
  } catch {
    return undefined;
  }
}
