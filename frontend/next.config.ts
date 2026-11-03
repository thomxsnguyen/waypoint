import type { NextConfig } from "next";
import { PHASE_DEVELOPMENT_SERVER } from "next/constants";
export default function config(phase: string): NextConfig {
  const testing = process.env.WAYPOINT_E2E === "1";
  if (testing && phase !== PHASE_DEVELOPMENT_SERVER)
    throw new Error(
      "The E2E harness cannot be included in a production build.",
    );
  return {
    output: "standalone",
    poweredByHeader: false,
    ...(testing
      ? {
          turbopack: {
            resolveAlias: {
              "@/components/session": "./tests/fixtures/session.tsx",
            },
          },
        }
      : {}),
  };
}
