#!/usr/bin/env node
'use strict';

/**
 * pre-compact.js — PreCompact hook
 *
 * Fires before Claude Code compresses the context window.
 * Saves a checkpoint marker to the telemetry feed so the MCP server
 * can correlate behavior before and after compaction events.
 *
 * Hook ID : pre:compact
 * Profiles: standard, strict
 *
 * Event schema:
 *   { timestamp, session_id, type: "checkpoint", project }
 */

const path = require('path');
const { appendFile, log, parseJSON, getProjectName, getSessionId, getToolkitDir, ensureSession, logEventToDb } = require('./lib/utils');

const TELEMETRY_FILE = path.join(getToolkitDir(), 'telemetry', 'events.jsonl');

/**
 * @param {string} rawInput - Raw stdin JSON string from Claude Code
 * @returns {string} rawInput unchanged (passthrough)
 */
function run(rawInput) {
  try {
    parseJSON(rawInput);

    const project = getProjectName();
    const sessionId = getSessionId();

    log(`[Toolkit] Saving checkpoint before compaction (project: ${project})`);

    // Write to JSONL (legacy)
    const event = JSON.stringify({
      timestamp: new Date().toISOString(),
      session_id: sessionId,
      type: 'checkpoint',
      project,
    });
    appendFile(TELEMETRY_FILE, event + '\n');

    // Write to DB
    ensureSession(sessionId, project);
    logEventToDb({ sessionId, type: 'checkpoint', details: `Compaction in ${project}` });
  } catch (err) {
    log(`[Toolkit] pre-compact error: ${err.message}`);
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
