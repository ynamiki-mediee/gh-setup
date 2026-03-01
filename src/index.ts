import { init } from "./init.ts";
import { milestones } from "./milestones.ts";
import { labels } from "./labels.ts";

function showHelp(): void {
  console.log(`
gh-setup — Interactive GitHub repository setup CLI

Usage:
  gh-setup <command>

Commands:
  init          Repository setup (branch protection, settings, security)
  milestones    Create/update weekly milestones
  labels        Sync labels from .gh-setup.yml

Options:
  --help        Show this help message
`);
}

const subcommand = process.argv[2];

switch (subcommand) {
  case "init":
    init().catch((e) => {
      console.error(e);
      process.exitCode = 1;
    });
    break;
  case "milestones":
    milestones().catch((e) => {
      console.error(e);
      process.exitCode = 1;
    });
    break;
  case "labels":
    labels().catch((e) => {
      console.error(e);
      process.exitCode = 1;
    });
    break;
  default:
    showHelp();
    if (subcommand && subcommand !== "--help") {
      process.exitCode = 1;
    }
    break;
}
