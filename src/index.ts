import { main } from "./prompts.js";

main().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
