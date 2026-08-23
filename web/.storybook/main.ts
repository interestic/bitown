import type { StorybookConfig } from "@storybook/react-vite";
import { mergeConfig } from "vite";
import path from "node:path";
import { fileURLToPath } from "node:url";

const rootDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(rootDir, "../..");
const spritesDir = path.join(repoRoot, "assets/sprites-v1");

const config: StorybookConfig = {
  stories: ["../src/**/*.stories.@(ts|tsx)"],
  addons: ["@storybook/addon-essentials"],
  framework: {
    name: "@storybook/react-vite",
    options: {},
  },
  core: {
    disableTelemetry: true,
  },
  staticDirs: [{ from: spritesDir, to: "/sprites-v1" }],
  async viteFinal(config) {
    return mergeConfig(config, {
      resolve: {
        alias: {
          "@sprites": spritesDir,
        },
      },
    });
  },
};

export default config;
