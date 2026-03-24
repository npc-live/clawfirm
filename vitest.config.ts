import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    include: ["src/core/__tests__/**/*.test.ts"],
    environment: "node",
    globals: false,
    // ESM support
    pool: "forks",
  },
});
