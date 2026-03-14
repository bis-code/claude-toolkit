/* Claude Toolkit Dashboard — app.js */

'use strict';

// ── Utility helpers ───────────────────────────────────────────

function fmtTime(iso) {
  if (!iso) return '—';
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function fmtDate(iso) {
  if (!iso) return '—';
  const d = new Date(iso);
  return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function fmtDuration(startISO, endISO) {
  if (!startISO) return '—';
  const start = new Date(startISO);
  const end = endISO ? new Date(endISO) : new Date();
  const s = Math.floor((end - start) / 1000);
  if (s < 60) return s + 's';
  const m = Math.floor(s / 60);
  if (m < 60) return m + 'm ' + (s % 60) + 's';
  return Math.floor(m / 60) + 'h ' + (m % 60) + 'm';
}

function resultBadgeClass(result) {
  if (!result) return 'badge--muted';
  const r = result.toLowerCase();
  if (r === 'success' || r === 'verified') return 'badge--success';
  if (r === 'failure' || r === 'failed' || r === 'error') return 'badge--danger';
  if (r === 'warning') return 'badge--warning';
  return 'badge--info';
}

function severityBadgeClass(severity) {
  if (severity === 'critical') return 'badge--danger';
  if (severity === 'warning')  return 'badge--warning';
  return 'badge--muted';
}

function skillBarClass(eff) {
  if (eff >= 0.7) return 'skill-bar__fill--high';
  if (eff >= 0.4) return 'skill-bar__fill--mid';
  return 'skill-bar__fill--low';
}

function clamp(v, lo, hi) { return Math.max(lo, Math.min(hi, v)); }

// ── API fetch helpers ─────────────────────────────────────────

async function apiFetch(path) {
  try {
    const res = await fetch(path);
    if (!res.ok) throw new Error(res.status);
    return await res.json();
  } catch (err) {
    console.error('[api]', path, err);
    return null;
  }
}

// ── State ─────────────────────────────────────────────────────

const state = {
  sessions: [],
  events: [],      // live feed (prepended)
  skills: [],
  patterns: [],
  patrolAlerts: [],
  activeSession: null,
  patrolSessionID: '',
  liveConnected: false,
};

// ── DOM refs ──────────────────────────────────────────────────

const $ = id => document.getElementById(id);

// ── Rendering ─────────────────────────────────────────────────

function renderActiveSession() {
  const sess = state.sessions[0] || null;
  const el = $('active-session-content');
  if (!el) return;

  if (!sess) {
    el.innerHTML = '<span class="text-muted">No active session found.</span>';
    return;
  }

  const duration = fmtDuration(sess.started_at, sess.ended_at);
  const status = sess.ended_at ? 'ended' : 'running';
  const statusClass = sess.ended_at ? 'stat__value--muted' : 'stat__value--success';

  el.innerHTML = `
    <div class="stat">
      <span class="stat__label">Project</span>
      <span class="stat__value stat__value--accent">${esc(sess.project)}</span>
    </div>
    <div class="stat">
      <span class="stat__label">Session ID</span>
      <span class="stat__value font-mono text-muted" style="font-size:11px">${esc(sess.id.substring(0,12))}…</span>
    </div>
    <div class="stat">
      <span class="stat__label">Duration</span>
      <span class="stat__value">${esc(duration)}</span>
    </div>
    <div class="stat">
      <span class="stat__label">Status</span>
      <span class="stat__value ${statusClass}">${esc(status)}</span>
    </div>
    <div class="stat">
      <span class="stat__label">Events</span>
      <span class="stat__value">${state.events.length}</span>
    </div>
  `;
}

function renderEventFeed() {
  const el = $('events-feed');
  if (!el) return;

  if (state.events.length === 0) {
    el.innerHTML = '<div class="empty-feed">Waiting for events…</div>';
    return;
  }

  el.innerHTML = state.events.slice(0, 100).map(ev => `
    <div class="event-row">
      <span class="event-row__type">${esc(ev.type)}</span>
      <span class="event-row__target text-muted">${esc(ev.session_id ? ev.session_id.substring(0,8) : '—')}</span>
      <span class="event-row__result">
        ${ev.result ? `<span class="badge ${resultBadgeClass(ev.result)}">${esc(ev.result)}</span>` : '<span class="text-muted">—</span>'}
        ${ev.details ? `<span class="text-muted" style="font-size:10px;margin-left:4px">${esc(ev.details.substring(0, 60))}</span>` : ''}
      </span>
      <span class="event-row__ts">${fmtTime(ev.timestamp)}</span>
    </div>
  `).join('');
}

function renderSessions() {
  const el = $('sessions-tbody');
  if (!el) return;

  if (state.sessions.length === 0) {
    el.innerHTML = '<tr><td colspan="5" class="empty-state">No sessions yet.</td></tr>';
    return;
  }

  el.innerHTML = state.sessions.slice(0, 10).map(sess => `
    <tr>
      <td class="font-mono text-muted" style="font-size:11px">${esc(sess.id.substring(0, 12))}…</td>
      <td>${esc(sess.project)}</td>
      <td class="text-muted">${fmtDuration(sess.started_at, sess.ended_at)}</td>
      <td style="text-align:center">
        <span class="badge badge--success">${sess.tasks_completed}</span>
        <span class="badge badge--danger">${sess.tasks_failed}</span>
      </td>
      <td class="text-muted" style="font-size:11px">${fmtDate(sess.started_at)}</td>
    </tr>
  `).join('');
}

function renderSkills() {
  const el = $('skills-list');
  if (!el) return;

  if (state.skills.length === 0) {
    el.innerHTML = '<div class="empty-state">No skill data yet.</div>';
    return;
  }

  el.innerHTML = state.skills.map(skill => {
    const pct = Math.round(skill.effectiveness * 100);
    const barClass = skillBarClass(skill.effectiveness);
    const warn = skill.effectiveness < 0.7 ? '<span class="warn-icon" title="Below threshold">⚠</span>' : '';
    return `
      <div class="skill-item">
        <div class="skill-item__header">
          <span class="skill-item__name">${warn}${esc(skill.name)}</span>
          <span class="skill-item__score" style="color:${skill.effectiveness >= 0.7 ? 'var(--success)' : skill.effectiveness >= 0.4 ? 'var(--warning)' : 'var(--danger)'}">${pct}%</span>
        </div>
        <div class="skill-bar">
          <div class="skill-bar__fill ${barClass}" style="width:${pct}%"></div>
        </div>
      </div>
    `;
  }).join('');
}

function renderPatterns() {
  const el = $('patterns-list');
  if (!el) return;

  if (state.patterns.length === 0) {
    el.innerHTML = '<div class="empty-state">No learned patterns yet.</div>';
    return;
  }

  el.innerHTML = state.patterns.map(p => {
    const conf = Math.round((p.confidence || 0) * 100);
    return `
      <div class="pattern-item">
        <div class="pattern-item__content">${esc(p.content)}</div>
        <div class="pattern-item__meta">
          <span class="badge badge--purple">${esc(p.scope)}</span>
          <span class="text-muted" style="font-size:10px">confidence: ${conf}%</span>
        </div>
      </div>
    `;
  }).join('');
}

function renderPatrol() {
  const statusEl = $('patrol-status-label');
  const lightEl  = $('patrol-light');
  const listEl   = $('patrol-alerts');
  if (!statusEl || !lightEl || !listEl) return;

  const alerts = state.patrolAlerts;

  if (alerts.length === 0) {
    lightEl.className = 'patrol-light patrol-light--green';
    statusEl.textContent = 'No anti-patterns detected';
    statusEl.style.color = 'var(--success)';
    listEl.innerHTML = '<div class="no-alerts">All clear.</div>';
    return;
  }

  const hasCritical = alerts.some(a => a.severity === 'critical');
  if (hasCritical) {
    lightEl.className = 'patrol-light patrol-light--red';
    statusEl.textContent = `${alerts.length} alert${alerts.length > 1 ? 's' : ''} — critical`;
    statusEl.style.color = 'var(--danger)';
  } else {
    lightEl.className = 'patrol-light patrol-light--yellow';
    statusEl.textContent = `${alerts.length} alert${alerts.length > 1 ? 's' : ''} — warnings`;
    statusEl.style.color = 'var(--warning)';
  }

  listEl.innerHTML = alerts.map(alert => `
    <div class="alert-item alert-item--${alert.severity}">
      <div class="alert-item__header">
        <span class="badge ${severityBadgeClass(alert.severity)}">${esc(alert.severity)}</span>
        <span class="alert-item__pattern">${esc(alert.pattern)}</span>
        <span class="text-muted" style="font-size:10px">${alert.event_count} events</span>
      </div>
      <div class="alert-item__message">${esc(alert.message)}</div>
      <div class="alert-item__suggestion">Suggestion: ${esc(alert.suggestion)}</div>
    </div>
  `).join('');
}

// ── Escape HTML ───────────────────────────────────────────────

function esc(str) {
  if (str == null) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

// ── Data loading ──────────────────────────────────────────────

async function loadSessions() {
  const data = await apiFetch('/api/sessions?limit=10');
  if (data && data.sessions) {
    state.sessions = data.sessions;
    renderActiveSession();
    renderSessions();
    populatePatrolSelect();
  }
}

async function loadRecentEvents() {
  const data = await apiFetch('/api/events/recent?limit=50');
  if (data && data.events) {
    state.events = data.events;
    renderEventFeed();
  }
}

async function loadSkills() {
  const data = await apiFetch('/api/skills');
  if (data && data.skills) {
    state.skills = data.skills;
    renderSkills();
  }
}

async function loadPatterns() {
  const data = await apiFetch('/api/workflow');
  if (data && data.patterns) {
    state.patterns = data.patterns;
    renderPatterns();
  }
}

async function loadPatrol(sessionID) {
  if (!sessionID) return;
  const data = await apiFetch('/api/patrol?session_id=' + encodeURIComponent(sessionID));
  if (data && data.alerts) {
    state.patrolAlerts = data.alerts;
    renderPatrol();
  }
}

function populatePatrolSelect() {
  const sel = $('patrol-session-select');
  if (!sel) return;
  const current = sel.value;
  sel.innerHTML = '<option value="">Select a session…</option>' +
    state.sessions.map(s => `<option value="${esc(s.id)}">${esc(s.id.substring(0,12))} — ${esc(s.project)}</option>`).join('');
  if (current) sel.value = current;
}

// ── SSE connection ────────────────────────────────────────────

function prependEvent(ev) {
  state.events.unshift(ev);
  if (state.events.length > 200) state.events.pop();
  renderEventFeed();

  // Also refresh sessions/skills when new events arrive (throttled).
  debouncedRefresh();
}

let refreshTimer = null;
function debouncedRefresh() {
  clearTimeout(refreshTimer);
  refreshTimer = setTimeout(() => {
    loadSessions();
    loadSkills();
    loadPatterns();
    // Auto-refresh patrol for the active session
    const sel = $('patrol-session-select');
    if (sel && sel.value) loadPatrol(sel.value);
    else if (state.sessions.length > 0) loadPatrol(state.sessions[0].id);
  }, 2000);
}

function connectSSE() {
  const liveText = $('live-text');
  const liveDot  = $('live-dot');

  const es = new EventSource('/sse/events');

  es.onopen = () => {
    state.liveConnected = true;
    if (liveDot)  liveDot.className  = 'live-dot';
    if (liveText) liveText.textContent = 'Live';
  };

  es.onmessage = (e) => {
    try {
      const ev = JSON.parse(e.data);
      prependEvent(ev);
    } catch (err) {
      console.warn('[sse] parse error', err);
    }
  };

  es.onerror = () => {
    state.liveConnected = false;
    if (liveDot)  liveDot.className  = 'live-dot live-dot--disconnected';
    if (liveText) liveText.textContent = 'Disconnected — retrying…';
    // EventSource will auto-retry; don't close it.
  };
}

// ── Patrol session select handler ─────────────────────────────

function onPatrolSelectChange() {
  const sel = $('patrol-session-select');
  if (sel && sel.value) {
    loadPatrol(sel.value);
  }
}

// ── Init ──────────────────────────────────────────────────────

async function init() {
  // Inject port into header from location
  const portEl = $('header-port');
  if (portEl) portEl.textContent = window.location.port || '80';

  await Promise.all([
    loadSessions(),
    loadRecentEvents(),
    loadSkills(),
    loadPatterns(),
  ]);

  // Load patrol for most recent session by default if any
  if (state.sessions.length > 0) {
    const sel = $('patrol-session-select');
    if (sel) {
      sel.value = state.sessions[0].id;
      loadPatrol(state.sessions[0].id);
    }
  } else {
    renderPatrol();
  }

  connectSSE();

  // Periodic refresh every 30s for stale tabs with no SSE
  setInterval(() => {
    loadSessions();
    loadSkills();
    loadPatterns();
    if (state.sessions.length > 0 && !$('patrol-session-select')?.value) {
      loadPatrol(state.sessions[0].id);
    }
  }, 30000);
}

document.addEventListener('DOMContentLoaded', init);
