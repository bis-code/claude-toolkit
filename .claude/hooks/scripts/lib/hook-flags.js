#!/usr/bin/env node
'use strict';

/**
 * Profile-based hook enabling system.
 *
 * Profiles: minimal, standard (default), strict
 * Each hook declares which profiles it runs in.
 * Users control via TOOLKIT_HOOK_PROFILE and TOOLKIT_DISABLED_HOOKS env vars.
 */

const VALID_PROFILES = ['minimal', 'standard', 'strict'];

/**
 * Check if a hook is enabled based on profile and disabled list.
 *
 * @param {string} hookId - Unique hook identifier (e.g., "pre:bash:tmux-reminder")
 * @param {object} options
 * @param {string} [options.profiles] - Comma-separated profiles this hook runs in (e.g., "standard,strict")
 * @returns {boolean}
 */
function isHookEnabled(hookId, options = {}) {
  const currentProfile = (process.env.TOOLKIT_HOOK_PROFILE || 'standard').toLowerCase().trim();
  const disabledHooks = (process.env.TOOLKIT_DISABLED_HOOKS || '')
    .split(',')
    .map((h) => h.trim().toLowerCase())
    .filter(Boolean);

  // Check if explicitly disabled
  if (disabledHooks.includes(hookId.toLowerCase())) {
    return false;
  }

  // Check if current profile matches allowed profiles
  const allowedProfiles = (options.profiles || 'standard,strict')
    .split(',')
    .map((p) => p.trim().toLowerCase())
    .filter(Boolean);

  // "strict" includes everything in "standard" which includes everything in "minimal"
  const profileHierarchy = {
    minimal: ['minimal'],
    standard: ['minimal', 'standard'],
    strict: ['minimal', 'standard', 'strict'],
  };

  const activeProfiles = profileHierarchy[currentProfile] || profileHierarchy.standard;

  return allowedProfiles.some((p) => activeProfiles.includes(p));
}

module.exports = { isHookEnabled, VALID_PROFILES };
