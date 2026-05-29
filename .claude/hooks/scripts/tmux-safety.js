#!/usr/bin/env node
'use strict';

/**
 * tmux-safety.js — PreToolUse(Bash) hook
 *
 * Blocks dev server commands when not running inside a tmux session.
 * Dev servers are long-running processes; if Claude Code exits, any server
 * it launched dies with it. Requiring tmux protects the user's workflow.
 *
 * Hook ID : pre:bash:tmux-safety
 * Profiles: standard, strict
 *
 * Blocking behaviour:
 *   - In the run() fast-path (via run-with-flags.js): returns a JSON string
 *     containing { "error": "..." }. run-with-flags.js writes this to stdout
 *     and exits 0, but Claude Code interprets the error field as a block signal.
 *   - In the CLI entrypoint path (legacy spawn): writes the error JSON to stdout
 *     then calls process.exit(2) so the hook runner knows to abort the tool call.
 *
 * If $TMUX is set, or the command is not a recognised dev server, the hook
 * passes through silently.
 */

const { log, parseJSON } = require('./lib/utils');

/**
 * Patterns that identify long-running dev server invocations.
 * Deliberately conservative — only match commands that are very unlikely to
 * be short-lived. `go run .` and `cargo run` are excluded unless they look
 * like server entry-points because they're commonly used for one-shot scripts.
 */
const DEV_SERVER_PATTERNS = [
  /\bnpm\s+(run\s+)?dev\b/,
  /\bnpm\s+start\b/,
  /\byarn\s+dev\b/,
  /\bpnpm\s+dev\b/,
  /\bbun\s+dev\b/,
  /\bpython\b.*manage\.py\s+runserver\b/,
  /\bpython\s+-m\s+http\.server\b/,
  /\bnext\s+dev\b/,
  /\bvite\b(?!\s+build)/,           // vite alone or vite --port etc., not vite build
  /\bwebpack-dev-server\b/,
  /\bwebpack\s+serve\b/,
];

const BLOCK_REASON =
  'Dev server commands should run inside tmux. Start tmux first.';

/**
 * Check whether a shell command string looks like a dev server invocation.
 *
 * @param {string} command
 * @returns {boolean}
 */
function isDevServer(command) {
  if (!command || typeof command !== 'string') return false;
  return DEV_SERVER_PATTERNS.some((re) => re.test(command));
}

/**
 * @param {string} rawInput - Raw stdin JSON string from Claude Code
 * @returns {string} rawInput unchanged, or a JSON error string to signal blocking
 */
function run(rawInput) {
  try {
    const payload = parseJSON(rawInput);
    const command = (payload.tool_input || payload.input || {}).command || '';

    if (isDevServer(command) && !process.env.TMUX) {
      log('[Toolkit] WARNING: Dev server blocked — run inside tmux for session safety');
      // Return a JSON error object. Claude Code treats this as a hook block signal.
      return JSON.stringify({ error: BLOCK_REASON });
    }
  } catch (err) {
    log(`[Toolkit] tmux-safety error: ${err.message}`);
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

    // Exit 2 when the hook decided to block so the spawn-based runner
    // can propagate the non-zero exit code to Claude Code.
    const blocked = result !== raw;
    process.exit(blocked ? 2 : 0);
  });
}
