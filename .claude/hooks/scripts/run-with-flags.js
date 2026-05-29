#!/usr/bin/env node
/**
 * Hook middleware — executes a hook script only when enabled by profile flags.
 *
 * Usage:
 *   node run-with-flags.js <hookId> <scriptRelativePath> [profilesCsv]
 *
 * Reads stdin, checks profile, runs the target hook, writes stdout.
 * If the hook exports a run(rawInput) function, calls it directly (saves ~50-100ms).
 * Otherwise spawns a child Node process (legacy path).
 */

'use strict';

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');
const { isHookEnabled } = require('./lib/hook-flags');

const MAX_STDIN = 1024 * 1024;

function readStdinRaw() {
  return new Promise((resolve) => {
    let raw = '';
    process.stdin.setEncoding('utf8');
    process.stdin.on('data', (chunk) => {
      if (raw.length < MAX_STDIN) {
        const remaining = MAX_STDIN - raw.length;
        raw += chunk.substring(0, remaining);
      }
    });
    process.stdin.on('end', () => resolve(raw));
    process.stdin.on('error', () => resolve(raw));
  });
}

function getScriptsRoot() {
  return path.resolve(__dirname);
}

async function main() {
  const [, , hookId, relScriptPath, profilesCsv] = process.argv;
  const raw = await readStdinRaw();

  if (!hookId || !relScriptPath) {
    process.stdout.write(raw);
    process.exit(0);
  }

  if (!isHookEnabled(hookId, { profiles: profilesCsv })) {
    process.stdout.write(raw);
    process.exit(0);
  }

  const scriptsRoot = getScriptsRoot();
  const scriptPath = path.resolve(scriptsRoot, relScriptPath);

  // Prevent path traversal outside scripts root
  if (!scriptPath.startsWith(scriptsRoot + path.sep) && scriptPath !== scriptsRoot) {
    process.stderr.write(`[Toolkit Hook] Path traversal rejected for ${hookId}: ${scriptPath}\n`);
    process.stdout.write(raw);
    process.exit(0);
  }

  if (!fs.existsSync(scriptPath)) {
    process.stderr.write(`[Toolkit Hook] Script not found for ${hookId}: ${scriptPath}\n`);
    process.stdout.write(raw);
    process.exit(0);
  }

  // Prefer direct require() when the hook exports a run(rawInput) function.
  // Saves ~50-100ms per hook by avoiding child process spawn.
  let hookModule;
  const src = fs.readFileSync(scriptPath, 'utf8');
  const hasRunExport = /module\.exports/.test(src) && /exports\.\s*run\b|{\s*run\s*[,}]/.test(src);

  if (hasRunExport) {
    try {
      hookModule = require(scriptPath);
    } catch (requireErr) {
      process.stderr.write(`[Toolkit Hook] require() failed for ${hookId}: ${requireErr.message}\n`);
    }
  }

  if (hookModule && typeof hookModule.run === 'function') {
    try {
      // Pass hookId so scripts can detect pre vs post phase
      process.env.TOOLKIT_HOOK_ID = hookId;

      // Extract session_id from payload and set as env var
      // so getSessionId() returns the real Claude Code session ID
      try {
        const payload = JSON.parse(raw);
        if (payload && payload.session_id) {
          process.env.CLAUDE_SESSION_ID = payload.session_id;
        }
      } catch { /* ignore parse errors */ }

      const result = hookModule.run(raw);
      if (result !== null && result !== undefined) process.stdout.write(String(result));
    } catch (runErr) {
      process.stderr.write(`[Toolkit Hook] run() error for ${hookId}: ${runErr.message}\n`);
      process.stdout.write(raw);
    }
    process.exit(0);
  }

  // Legacy path: spawn a child Node process
  const result = spawnSync('node', [scriptPath], {
    input: raw,
    encoding: 'utf8',
    env: process.env,
    cwd: process.cwd(),
    timeout: 30000,
  });

  if (result.stdout) process.stdout.write(result.stdout);
  if (result.stderr) process.stderr.write(result.stderr);

  const code = Number.isInteger(result.status) ? result.status : 0;
  process.exit(code);
}

main().catch((err) => {
  process.stderr.write(`[Toolkit Hook] run-with-flags error: ${err.message}\n`);
  process.exit(0);
});
