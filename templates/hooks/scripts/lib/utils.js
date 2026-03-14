#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const os = require('os');

const MAX_STDIN = 1024 * 1024;

/**
 * Get the Claude config directory (~/.claude).
 * @returns {string}
 */
function getClaudeDir() {
  return path.join(os.homedir(), '.claude');
}

/**
 * Get the toolkit data directory (~/.claude-toolkit).
 * @returns {string}
 */
function getToolkitDir() {
  return path.join(os.homedir(), '.claude-toolkit');
}

/**
 * Get the metrics directory (~/.claude/metrics).
 * @returns {string}
 */
function getMetricsDir() {
  return path.join(getClaudeDir(), 'metrics');
}

/**
 * Ensure a directory exists, creating it recursively if needed.
 * @param {string} dirPath
 */
function ensureDir(dirPath) {
  if (!fs.existsSync(dirPath)) {
    fs.mkdirSync(dirPath, { recursive: true });
  }
}

/**
 * Read a file's contents, returning null if it doesn't exist.
 * @param {string} filePath
 * @returns {string|null}
 */
function readFile(filePath) {
  try {
    return fs.readFileSync(filePath, 'utf8');
  } catch {
    return null;
  }
}

/**
 * Append content to a file, creating it if needed.
 * @param {string} filePath
 * @param {string} content
 */
function appendFile(filePath, content) {
  ensureDir(path.dirname(filePath));
  fs.appendFileSync(filePath, content, 'utf8');
}

/**
 * Log a message to stderr (visible in Claude Code hook output).
 * @param {string} msg
 */
function log(msg) {
  process.stderr.write(`${msg}\n`);
}

/**
 * Output content to stdout (injected into Claude's context).
 * @param {string} content
 */
function output(content) {
  process.stdout.write(content);
}

/**
 * Read all stdin up to MAX_STDIN bytes.
 * @returns {Promise<string>}
 */
function readStdin() {
  return new Promise((resolve) => {
    let raw = '';
    process.stdin.setEncoding('utf8');
    process.stdin.on('data', (chunk) => {
      if (raw.length < MAX_STDIN) {
        const remaining = MAX_STDIN - raw.length;
        raw += chunk.substring(0, remaining);
      }
    });
    process.stdin.on('end', () => resolve(raw));
    process.stdin.on('error', () => resolve(raw));
  });
}

/**
 * Parse JSON safely, returning empty object on failure.
 * @param {string} raw
 * @returns {object}
 */
function parseJSON(raw) {
  try {
    return raw.trim() ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
}

/**
 * Get the current project name from git or cwd.
 * @returns {string}
 */
function getProjectName() {
  try {
    const { execSync } = require('child_process');
    const root = execSync('git rev-parse --show-toplevel 2>/dev/null', { encoding: 'utf8' }).trim();
    return path.basename(root);
  } catch {
    return path.basename(process.cwd());
  }
}

/**
 * Get session ID from environment.
 * @returns {string}
 */
function getSessionId() {
  return process.env.CLAUDE_SESSION_ID || `session-${Date.now()}`;
}

module.exports = {
  MAX_STDIN,
  getClaudeDir,
  getToolkitDir,
  getMetricsDir,
  ensureDir,
  readFile,
  appendFile,
  log,
  output,
  readStdin,
  parseJSON,
  getProjectName,
  getSessionId,
};
