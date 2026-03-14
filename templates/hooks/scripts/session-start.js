#!/usr/bin/env node
'use strict';

/**
 * session-start.js — SessionStart hook
 *
 * Fires when Claude Code begins a new session.
 * Detects the current project, logs session start to stderr,
 * and injects project context into Claude's context via stdout.
 *
 * Hook ID : session:start
 * Profiles: minimal, standard, strict
 */

const { log, output, parseJSON, getProjectName, getSessionId } = require('./lib/utils');

/**
 * @param {string} rawInput - Raw stdin JSON string from Claude Code
 * @returns {string} rawInput unchanged (passthrough)
 */
function run(rawInput) {
  try {
    // parseJSON returns {} on failure — intentionally unused here;
    // we parse for forward compatibility if the payload gains fields we need.
    parseJSON(rawInput);

    const project = getProjectName();
    const sessionId = getSessionId();

    log(`[Toolkit] Session started (project: ${project}, session: ${sessionId})`);
    output(`[Toolkit Context] Project: ${project}\n`);
  } catch (err) {
    // Never crash Claude Code — swallow and log
    log(`[Toolkit] session-start error: ${err.message}`);
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
