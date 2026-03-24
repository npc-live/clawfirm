import { defineConfig } from "vitest/config";
export default defineConfig({
  test: {
    include: ["tests/e2e/**/*.ts"],
    environment: "node",
    globals: false,
    pool: "forks",
  },
});
