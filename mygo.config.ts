import { defineConfig } from "mygo-cli";

export default defineConfig({
  name: "GDUT Course Helper",
  identifier: "com.kirishima.gdutcoursehelper",
  version: "2.0.0",
  devUrl: "http://localhost:5173",
  devCommand: "bun run dev:web",
  buildCommand: "bun run build:web",
  frontendDist: "dist",
  bindings: "src/mygo.ts",
  out: "build",
});
