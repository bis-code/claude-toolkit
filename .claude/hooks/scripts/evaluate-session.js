#!/usr/bin/env node
'use strict';

/**
 * evaluate-session.js — Stop hook
 *
 * Fires when a Claude Code session ends.
 * Reads recent telemetry events, detects which skill was likely in use,
 * computes a simple effectiveness score, and appends a skill_eval event
 * to the telemetry feed for the MCP server to consume.
 *
 * Hook ID : stop:evaluate-session
 * Profiles: standard, strict
 *
 * Scoring rubric (0.0–1.0):
 *   Base              0.5
 *   Successes         +0.1 per 10 successful events, capped at +0.3
 *   Failures          -0.1 per failure,               capped at -0.3
 *   Reasonable size   +0.2 if total events < 100 (not stuck in a loop)
 */

const path = require('path');
const {
  appendFile,
  log,
  parseJSON,
  getProjectName,
  getSessionId,
  getToolkitDir,
  readFile,
  dbExec,
} = require('./lib/utils');

const TELEMETRY_FILE = path.join(getToolkitDir(), 'telemetry', 'events.jsonl');

// Keywords in event payloads that suggest a skill is active.
// Ordered by specificity — first match wins.
const SKILL_PATTERNS = [
  { skill: 'tdd-workflow',     pattern: /\b(test|spec|jest|pytest|vitest)\b/i },
  { skill: 'code-review',      pattern: /\b(review|diff|pr|pull.?request)\b/i },
  { skill: 'refactor',         pattern: /\b(refactor|rename|extract|move)\b/i },
  { skill: 'debug',            pattern: /\b(debug|error|stack.?trace|exception|panic)\b/i },
  { skill: 'database-migrate', pattern: /\b(migration|migrate|schema|alter.?table)\b/i },
  { skill: 'ci-cd',            pattern: /\b(ci|cd|pipeline|workflow|deploy|github.?actions)\b/i },
  { skill: 'documentation',    pattern: /\b(docs?|readme|changelog|comment)\b/i },
];

/**
 * Read the last `n` lines from a file without loading the entire file into memory.
 * Falls back gracefully if the file doesn't exist or is empty.
 *
 * @param {string} filePath
 * @param {number} n
 * @returns {string[]}
 */
function readLastNLines(filePath, n) {
  const raw = readFile(filePath);
  if (!raw) return [];
  return raw.split('\n').filter(Boolean).slice(-n);
}

/**
 * Detect the most likely skill from a batch of raw event lines.
 *
 * @param {string[]} lines
 * @returns {string}
 */
function detectSkill(lines) {
  const combined = lines.join(' ');
  for (const { skill, pattern } of SKILL_PATTERNS) {
    if (pattern.test(combined)) return skill;
  }
  return 'general';
}

/**
 * Compute effectiveness score from parsed events.
 *
 * @param {object[]} events
 * @returns {number} clamped to [0.0, 1.0], one decimal place
 */
function computeScore(events) {
  let score = 0.5;

  const failures = events.filter((e) =>
    e.type === 'error' || e.type === 'failure' || String(e.tool || '').toLowerCase().includes('fail')
  ).length;

  const successes = events.filter((e) =>
    e.type !== 'error' && e.type !== 'failure' && e.type !== 'skill_eval'
  ).length;

  // Successes: +0.1 per 10, max +0.3
  score += Math.min(0.3, Math.floor(successes / 10) * 0.1);

  // Failures: -0.1 per failure, max -0.3
  score -= Math.min(0.3, failures * 0.1);

  // Reasonable session size bonus (only if there were actual events)
  if (events.length > 0 && events.length < 100) score += 0.2;

  return Math.round(Math.max(0.0, Math.min(1.0, score)) * 10) / 10;
}

/**
 * @param {string} rawInput - Raw stdin JSON string from Claude Code
 * @returns {string} rawInput unchanged (passthrough)
 */
function run(rawInput) {
  try {
    parseJSON(rawInput); // parse for forward-compat; payload unused at Stop event

    const lines = readLastNLines(TELEMETRY_FILE, 50);
    const events = lines.map((l) => parseJSON(l)).filter((e) => e && e.type);

    const skill = detectSkill(lines);
    const score = computeScore(events);
    const sessionId = getSessionId();
    const project = getProjectName();

    const evalEvent = JSON.stringify({
      timestamp: new Date().toISOString(),
      session_id: sessionId,
      type: 'skill_eval',
      skill,
      score,
      event_count: events.length,
      project,
    });

    appendFile(TELEMETRY_FILE, evalEvent + '\n');

    // Write skill score to DB
    const now = new Date().toISOString();
    const safeSkill = skill.replace(/'/g, "''");
    dbExec(
      `INSERT INTO skill_scores (skill, score, session_id, project, scored_at) VALUES ('${safeSkill}', ${score}, '${sessionId}', '${project}', '${now}');`
    );

    log(`[Toolkit] Session evaluated: ${skill} scored ${score}`);
  } catch (err) {
    log(`[Toolkit] evaluate-session error: ${err.message}`);
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
