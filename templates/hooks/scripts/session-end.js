#!/usr/bin/env node
'use strict';

/**
 * session-end.js — Stop hook
 *
 * Fires when Claude Code is about to exit.
 * Logs session end to stderr and appends a session_end marker
 * to the telemetry feed so the MCP server can compute session duration.
 *
 * Hook ID : session:end
 * Profiles: standard, strict
 *
 * Event schema:
 *   { timestamp, session_id, type: "session_end", project }
 */

const path = require('path');
const { appendFile, log, parseJSON, getProjectName, getSessionId, getToolkitDir } = require('./lib/utils');

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

    log(`[Toolkit] Session ending (project: ${project}, session: ${sessionId})`);

    const event = JSON.stringify({
      timestamp: new Date().toISOString(),
      session_id: sessionId,
      type: 'session_end',
      project,
    });

    appendFile(TELEMETRY_FILE, event + '\n');
  } catch (err) {
    log(`[Toolkit] session-end error: ${err.message}`);
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
