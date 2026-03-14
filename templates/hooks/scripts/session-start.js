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

const { log, parseJSON, getProjectName, getSessionId, ensureSession, logEventToDb } = require('./lib/utils');

/**
 * @param {string} rawInput - Raw stdin JSON string from Claude Code
 * @returns {string} rawInput unchanged (passthrough)
 */
function run(rawInput) {
  try {
    parseJSON(rawInput);

    const project = getProjectName();
    const sessionId = getSessionId();

    // Register session in the database
    ensureSession(sessionId, project);
    logEventToDb({ sessionId, type: 'session_start', details: `Project: ${project}` });

    log(`[Toolkit] Session started (project: ${project}, session: ${sessionId})`);
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
