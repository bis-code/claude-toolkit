#!/usr/bin/env node
'use strict';

/**
 * secret-detector.js — PreToolUse hook
 *
 * Advisory guard that warns when credential-like patterns are detected
 * in a tool's input payload. This hook NEVER blocks — it is informational
 * only. The rationale: a hook that silently blocks causes confusion and
 * erodes trust in the tool. Loud warnings are more actionable.
 *
 * Hook ID : pre:secret-detector
 * Profiles: standard, strict
 *
 * Patterns checked:
 *   sk-          OpenAI API keys
 *   ghp_         GitHub personal access tokens
 *   AKIA         AWS access key IDs
 *   xox          Slack tokens (xoxb-, xoxp-, xoxa-)
 *   -----BEGIN.*PRIVATE KEY-----   PEM private keys
 *   gho_         GitHub OAuth tokens
 *   glpat-       GitLab personal access tokens
 */

const { log, parseJSON } = require('./lib/utils');

/**
 * Credential patterns with human-readable labels.
 * Each entry: { label, regex }
 * Using regex rather than plain indexOf to support anchored patterns like the PEM header.
 */
const CREDENTIAL_PATTERNS = [
  { label: 'OpenAI API key (sk-)',          regex: /sk-[A-Za-z0-9]{10,}/  },
  { label: 'GitHub personal token (ghp_)',  regex: /ghp_[A-Za-z0-9]{10,}/ },
  { label: 'AWS access key (AKIA)',         regex: /AKIA[A-Z0-9]{12,}/     },
  { label: 'Slack token (xox)',             regex: /xox[bpas]-[A-Za-z0-9]/ },
  { label: 'PEM private key',              regex: /-----BEGIN[\s\S]{0,30}PRIVATE KEY-----/ },
  { label: 'GitHub OAuth token (gho_)',     regex: /gho_[A-Za-z0-9]{10,}/  },
  { label: 'GitLab token (glpat-)',         regex: /glpat-[A-Za-z0-9-]{10,}/ },
];

/**
 * Scan a string for known credential patterns.
 *
 * @param {string} text
 * @returns {{ label: string }[]} array of matched pattern descriptors
 */
function findCredentials(text) {
  return CREDENTIAL_PATTERNS.filter(({ regex }) => regex.test(text));
}

/**
 * @param {string} rawInput - Raw stdin JSON string from Claude Code
 * @returns {string} rawInput unchanged (passthrough — advisory only)
 */
function run(rawInput) {
  try {
    const payload = parseJSON(rawInput);
    const toolInput = payload.tool_input || payload.input || {};
    const serialized = JSON.stringify(toolInput);

    const matches = findCredentials(serialized);

    for (const { label } of matches) {
      log(`[Toolkit] WARNING: Potential credential detected in tool input! Pattern: ${label}`);
    }
  } catch (err) {
    // Never disrupt Claude Code — swallow silently
    log(`[Toolkit] secret-detector error: ${err.message}`);
  }

  // Always passthrough — this hook is advisory, never blocking
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
