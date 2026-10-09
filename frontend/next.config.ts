import type { NextConfig } from "next";
const config: NextConfig = { distDir: process.env.NODE_ENV === "production" ? ".next-production" : ".next", poweredByHeader: false, devIndicators: false };
export default config;
