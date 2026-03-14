#!/usr/bin/env node
'use strict';

/**
 * quality-gate.js — PostToolUse(Edit|Write) hook
 *
 * Automatically formats the file that was just edited or written by Claude Code.
 * Non-blocking by design: if the formatter is missing or fails, the hook logs and
 * continues. Code correctness takes precedence over formatting enforcement.
 *
 * Hook ID : post:edit:quality-gate
 * Profiles: standard, strict
 *
 * Formatter map:
 *   .go              gofmt -w <file>
 *   .py              ruff format <file>
 *   .ts/.tsx/.js/.jsx  biome check --write <file>  (falls back to prettier --write)
 *   .json/.md        prettier --write <file>        (if available)
 */

const path = require('path');
const { spawnSync } = require('child_process');
const { log, parseJSON } = require('./lib/utils');

const FORMATTER_TIMEOUT_MS = 15_000;

/**
 * Check whether a CLI tool is available on PATH.
 *
 * @param {string} tool
 * @returns {boolean}
 */
function isAvailable(tool) {
  const result = spawnSync('which', [tool], { encoding: 'utf8', timeout: 5_000 });
  return result.status === 0 && Boolean(result.stdout.trim());
}

/**
 * Run a formatter synchronously.
 *
 * @param {string} formatter - Executable name
 * @param {string[]} args
 * @param {string} filePath - Used only for logging
 * @returns {{ ok: boolean, reason?: string }}
 */
function runFormatter(formatter, args, filePath) {
  const result = spawnSync(formatter, args, {
    encoding: 'utf8',
    timeout: FORMATTER_TIMEOUT_MS,
    stdio: ['ignore', 'pipe', 'pipe'],
  });

  if (result.error) {
    return { ok: false, reason: result.error.message };
  }
  if (result.status !== 0) {
    const stderr = (result.stderr || '').trim();
    return { ok: false, reason: stderr || `exit ${result.status}` };
  }
  return { ok: true };
}

/**
 * Determine and execute the appropriate formatter for a given file.
 *
 * @param {string} filePath - Absolute or relative path to the edited file
 */
function formatFile(filePath) {
  const ext = path.extname(filePath).toLowerCase();

  if (ext === '.go') {
    if (!isAvailable('gofmt')) {
      log(`[Toolkit] No formatter for ${ext} (gofmt not found)`);
      return;
    }
    const { ok, reason } = runFormatter('gofmt', ['-w', filePath], filePath);
    ok
      ? log(`[Toolkit] Formatted ${filePath} with gofmt`)
      : log(`[Toolkit] gofmt failed for ${filePath}: ${reason}`);
    return;
  }

  if (ext === '.py') {
    if (!isAvailable('ruff')) {
      log(`[Toolkit] No formatter for ${ext} (ruff not found)`);
      return;
    }
    const { ok, reason } = runFormatter('ruff', ['format', filePath], filePath);
    ok
      ? log(`[Toolkit] Formatted ${filePath} with ruff`)
      : log(`[Toolkit] ruff failed for ${filePath}: ${reason}`);
    return;
  }

  if (['.ts', '.tsx', '.js', '.jsx'].includes(ext)) {
    if (isAvailable('biome')) {
      const { ok, reason } = runFormatter('biome', ['check', '--write', filePath], filePath);
      if (ok) {
        log(`[Toolkit] Formatted ${filePath} with biome`);
        return;
      }
      log(`[Toolkit] biome failed for ${filePath}: ${reason} — falling back to prettier`);
    }

    if (isAvailable('prettier')) {
      const { ok, reason } = runFormatter('prettier', ['--write', filePath], filePath);
      ok
        ? log(`[Toolkit] Formatted ${filePath} with prettier`)
        : log(`[Toolkit] prettier failed for ${filePath}: ${reason}`);
      return;
    }

    log(`[Toolkit] No formatter for ${ext} (biome and prettier both unavailable)`);
    return;
  }

  if (['.json', '.md'].includes(ext)) {
    if (!isAvailable('prettier')) {
      log(`[Toolkit] No formatter for ${ext} (prettier not found)`);
      return;
    }
    const { ok, reason } = runFormatter('prettier', ['--write', filePath], filePath);
    ok
      ? log(`[Toolkit] Formatted ${filePath} with prettier`)
      : log(`[Toolkit] prettier failed for ${filePath}: ${reason}`);
    return;
  }

  log(`[Toolkit] No formatter for ${ext}`);
}

/**
 * @param {string} rawInput - Raw stdin JSON string from Claude Code
 * @returns {string} rawInput unchanged (passthrough)
 */
function run(rawInput) {
  try {
    const payload = parseJSON(rawInput);
    const filePath = (payload.tool_input || payload.input || {}).file_path || '';

    if (!filePath) {
      log('[Toolkit] quality-gate: no file_path in tool input, skipping');
      return rawInput;
    }

    formatFile(filePath);
  } catch (err) {
    log(`[Toolkit] quality-gate error: ${err.message}`);
  }

  return rawInput;
}

module.exports = { run };

// CLI entrypoint (legacy spawn path)
if (require.main === module) {
  const { readStdin } = require('./lib/utils');
  readStdin().then((raw) => {
    const result = run(raw);
    if (result != null) process.stdout.write(String(result));
    process.exit(0);
  });
}
