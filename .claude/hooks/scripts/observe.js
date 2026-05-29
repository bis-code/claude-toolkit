#!/usr/bin/env node
'use strict';

/**
 * observe.js — PreToolUse / PostToolUse hook
 *
 * On PreToolUse: logs to JSONL only (lightweight, async)
 * On PostToolUse: logs to JSONL + SQLite DB with success/failure result
 *
 * The hookId is passed via TOOLKIT_HOOK_ID env var by run-with-flags.js.
 * If not set, defaults to PostToolUse behavior (writes to DB).
 *
 * Hook ID : pre:observe / post:observe
 * Profiles: standard, strict
 *
 * Event schema:
 *   { timestamp, session_id, tool, target, project }
 */

const path = require('path');
const { appendFile, log, parseJSON, getProjectName, getSessionId, getToolkitDir, ensureSession, logEventToDb } = require('./lib/utils');

const TELEMETRY_FILE = path.join(getToolkitDir(), 'telemetry', 'events.jsonl');

/**
 * Extract the most informative target from tool input.
 * @param {object} input
 * @returns {string}
 */
function extractTarget(input) {
  if (!input || typeof input !== 'object') return '';
  return (
    input.file_path ||
    input.path ||
    input.command ||
    input.url ||
    ''
  );
}

/**
 * Detect if a tool call failed based on PostToolUse payload.
 * Claude Code includes tool_result or output on PostToolUse.
 * @param {object} payload
 * @returns {string} 'success' or 'failure'
 */
function detectResult(payload) {
  // PostToolUse payloads may include:
  //   - tool_result with error field
  //   - output with error/exit code info
  //   - result field
  const result = payload.tool_result || payload.result || {};

  if (typeof result === 'string') {
    if (/error|failed|exit code [1-9]/i.test(result)) return 'failure';
    return 'success';
  }

  if (result.error || result.is_error) return 'failure';
  if (result.exitCode && result.exitCode !== 0) return 'failure';

  // Check for error patterns in output
  const output = payload.output || '';
  if (typeof output === 'string' && /^Error:|FAIL|exit code [1-9]|command failed/im.test(output)) {
    return 'failure';
  }

  return 'success';
}

/**
 * @param {string} rawInput - Raw stdin JSON string from Claude Code
 * @returns {string} rawInput unchanged (passthrough)
 */
function run(rawInput) {
  try {
    const payload = parseJSON(rawInput);
    const hookId = process.env.TOOLKIT_HOOK_ID || '';
    const isPreHook = hookId.startsWith('pre:');

    const tool = payload.tool_name || payload.tool || 'unknown';
    const target = extractTarget(payload.tool_input || payload.input || {});
    const project = getProjectName();
    const sessionId = getSessionId();

    // Always write to JSONL (lightweight)
    const event = JSON.stringify({
      timestamp: new Date().toISOString(),
      session_id: sessionId,
      tool,
      target,
      project,
    });
    appendFile(TELEMETRY_FILE, event + '\n');

    // Only write to DB on PostToolUse (avoids double-counting)
    if (!isPreHook) {
      const result = detectResult(payload);
      ensureSession(sessionId, project);
      const details = `${tool}: ${target}`.substring(0, 200);
      logEventToDb({ sessionId, type: 'tool_call', details, result });
    }
  } catch (err) {
    log(`[Toolkit] observe error: ${err.message}`);
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
  });
}
