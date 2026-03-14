#!/usr/bin/env node
'use strict';

/**
 * observe.js — PreToolUse / PostToolUse hook
 *
 * Appends a JSONL telemetry event for every tool call.
 * The MCP server reads events.jsonl to power metrics and dashboards.
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
 * Extract the most informative target path from tool input.
 * Different tools expose their primary target under different keys.
 *
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
 * @param {string} rawInput - Raw stdin JSON string from Claude Code
 * @returns {string} rawInput unchanged (passthrough)
 */
function run(rawInput) {
  try {
    const payload = parseJSON(rawInput);

    // Claude Code sends tool_name on PreToolUse, tool on some variants
    const tool = payload.tool_name || payload.tool || 'unknown';
    const target = extractTarget(payload.tool_input || payload.input || {});
    const project = getProjectName();
    const sessionId = getSessionId();

    // Write to JSONL file (legacy telemetry)
    const event = JSON.stringify({
      timestamp: new Date().toISOString(),
      session_id: sessionId,
      tool,
      target,
      project,
    });
    appendFile(TELEMETRY_FILE, event + '\n');

    // Write to SQLite DB (powers TUI + web dashboard)
    ensureSession(sessionId, project);
    const details = `${tool}: ${target}`.substring(0, 200);
    logEventToDb({ sessionId, type: 'tool_call', details });
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
