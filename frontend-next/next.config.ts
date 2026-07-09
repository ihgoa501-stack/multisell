import type { NextConfig } from "next";
import { withSentryConfig } from "@sentry/nextjs";

const nextConfig: NextConfig = {
  output: "standalone",
  turbopack: {
    root: process.cwd(),
  },
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: "http://localhost:8080/api/:path*",
      },
      {
        source: "/ws/:path*",
        destination: "http://localhost:8080/ws/:path*",
      },
    ];
  },
};

const sentryAuthToken = process.env.SENTRY_AUTH_TOKEN;

export default sentryAuthToken
  ? withSentryConfig(nextConfig, {
      authToken: sentryAuthToken,
    })
  : nextConfig;
