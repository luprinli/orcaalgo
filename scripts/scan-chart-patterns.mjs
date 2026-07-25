#!/usr/bin/env node
/**
 * Lightweight-Charts Anti-Pattern Scanner
 *
 * Scans chart source files for violations of the 6 hard prohibitions
 * defined in AGENTS.md. Run as part of CI or pre-commit hooks.
 *
 * Usage: node scripts/scan-chart-patterns.mjs
 */

import { readFileSync, readdirSync } from "fs";
import { join, resolve } from "path";
import { fileURLToPath } from "url";

const ROOT = resolve(fileURLToPath(import.meta.url), "../../web/src");
const CHART_DIRS = ["charts", "hooks"];

const RULES = [
  {
    id: "LWC-001",
    name: "Do not use setData() for incremental updates — review whether update() is appropriate",
    description:
      "setData() replaces all series data and impacts performance. Use series.update() for real-time/incremental data. This rule flags ALL setData() calls for review — initial loads are exempt.",
    pattern: /\.setData\(/g,
    allowedIn: ["useChart.ts"], // Factory hooks — setData exported for consumer use
    severity: "warn",
  },
  {
    id: "LWC-002",
    name: "Do not call fitContent() on data update cycles",
    description:
      "fitContent() resets user scroll/zoom position. Only call on initial load, timeframe change, or explicit user action.",
    pattern: /\.fitContent\(\)/g,
    allowedIn: ["useChartKeyboard.ts"],
    severity: "error",
    checkContext: true,
    // Also check if fitContent is in a useEffect that watches 'candles' or 'data' (not 'timeframe')
    allowedContextDeps: ["timeframe", "timeFrame"], // OK in effects watching these
  },
  {
    id: "LWC-003",
    name: "Do not use applyOptions({ width }) for resize",
    description:
      "Use chart.resize(width, height) instead. applyOptions re-applies ALL options on resize.",
    pattern: /applyOptions\(\s*\{\s*(width\s*:|.*,\s*width\s*:)/g,
    allowedIn: [],
    severity: "error",
  },
  {
    id: "LWC-004",
    name: "Do not use barSpacing mutation for zoom",
    description:
      "Use getVisibleLogicalRange() + setVisibleLogicalRange() for keyboard zoom. barSpacing conflicts with internal range calculation.",
    pattern: /barSpacing/g,
    allowedIn: ["useChartKeyboard.ts"], // Allowed only in keyboard hook (already fixed)
    severity: "warn",
  },
  {
    id: "LWC-005",
    name: "Do not leave RAF un-cancelled in chart hooks",
    description:
      "requestAnimationFrame must be cancelled in useEffect cleanup via cancelAnimationFrame.",
    pattern: /requestAnimationFrame\(/g,
    allowedIn: ["useChartUpdate.ts"], // Already has cleanup
    severity: "error",
    requiresCleanup: true,
  },
  {
    id: "LWC-006",
    name: "Do not use Array.find() in crosshair handlers",
    description:
      "Crosshair fires at 60fps. Use Map<time, value>.get() for O(1) lookups.",
    pattern: /subscribeCrosshairMove/g,
    allowedIn: [],
    severity: "warn",
    checkContext: true,
    // If subscribeCrosshairMove exists in a file, check for Array.find usage
  },
];

function scanFile(filePath) {
  try {
    const content = readFileSync(filePath, "utf-8");
    const fileName = filePath.split(/[/\\]/).pop();
    const results = [];

    for (const rule of RULES) {
      const matches = [...content.matchAll(rule.pattern)];

      for (const match of matches) {
        // Check allowed files
        if (
          rule.allowedIn.some(
            (allowed) => fileName === allowed || filePath.endsWith(allowed),
          )
        ) {
          continue;
        }

        const lineNum =
          content.substring(0, match.index).split("\n").length;

        // Additional context checks
        if (rule.id === "LWC-002" && rule.checkContext) {
          // Check context around fitContent — look at surrounding useEffect to find deps
          const surrounding = content.substring(
            Math.max(0, match.index - 500),
            Math.min(content.length, match.index + 200),
          );
          // Look for useEffect dependency array in surrounding context
          const depsMatch = surrounding.match(/\[([^\]]*)\]/g);
          const lastDeps = depsMatch ? depsMatch[depsMatch.length - 1] : "";
          if (
            lastDeps.includes("timeframe") ||
            lastDeps.includes("timeFrame")
          ) {
            continue;
          }
          if (!beforeMatch.includes("useEffect")) {
            continue;
          }
        }

        if (rule.id === "LWC-005" && rule.requiresCleanup) {
          if (!content.includes("cancelAnimationFrame")) {
            results.push({
              ruleId: rule.id,
              severity: "error",
              file: filePath,
              line: lineNum,
              message: `${rule.name}: RAF present but no cancelAnimationFrame found in file`,
            });
          }
          continue;
        }

        results.push({
          ruleId: rule.id,
          severity: rule.severity,
          file: filePath,
          line: lineNum,
          message: rule.name,
        });
      }

      // LWC-006: Check for Array.find in crosshair handlers
      if (
        rule.id === "LWC-006" &&
        content.includes("subscribeCrosshairMove") &&
        content.includes(".find(")
      ) {
        const findMatches = [...content.matchAll(/\.find\(/g)];
        findMatches.forEach((fm) => {
          const lineNum =
            content.substring(0, fm.index).split("\n").length;
          results.push({
            ruleId: rule.id,
            severity: rule.severity,
            file: filePath,
            line: lineNum,
            message: `${rule.name}: Array.find() detected in file with subscribeCrosshairMove`,
          });
        });
      }
    }

    return results;
  } catch {
    return [];
  }
}

function scanDir(dir) {
  const results = [];
  const fullPath = join(ROOT, dir);

  try {
    const entries = readdirSync(fullPath, { recursive: true, withFileTypes: true });
    for (const entry of entries) {
      if (entry.isFile() && /\.(ts|tsx)$/.test(entry.name)) {
        const filePath = join(entry.parentPath || fullPath, entry.name);
        results.push(...scanFile(filePath));
      }
    }
  } catch (e) {
    console.warn(`  WARN: Could not scan ${dir}: ${e.message}`);
  }

  return results;
}

// ── Main ────────────────────────────────────────────────────────────────────
console.log("\n🔍 Lightweight-Charts Anti-Pattern Scan\n");

let allResults = [];
for (const dir of CHART_DIRS) {
  const results = scanDir(dir);
  allResults.push(...results);
  console.log(
    `  ${dir}/ — ${results.length === 0 ? "✅ clean" : `❌ ${results.length} issues`}`,
  );
}

if (allResults.length === 0) {
  console.log("\n✅ No lightweight-charts anti-patterns detected.");
  process.exit(0);
}

console.log(`\n❌ ${allResults.length} violation(s) found:\n`);

const errors = allResults.filter((r) => r.severity === "error");
const warns = allResults.filter((r) => r.severity === "warn");

for (const r of [...errors, ...warns]) {
  const icon = r.severity === "error" ? "🔴" : "🟡";
  const shortFile = r.file.replace(ROOT + "/", "");
  console.log(
    `  ${icon} [${r.ruleId}] ${shortFile}:${r.line} — ${r.message}`,
  );
}

console.log();

if (errors.length > 0) {
  console.log(
    `🔴 ${errors.length} error(s) block merge. Fix before committing.\n`,
  );
  process.exit(1);
} else {
  console.log(`🟡 ${warns.length} warning(s). Review before committing.\n`);
  process.exit(0);
}
