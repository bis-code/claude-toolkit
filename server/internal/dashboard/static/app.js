/* Claude Toolkit Dashboard — Node Graph — app.js */

'use strict';

// ── Constants ────────────────────────────────────────────────
const LAYOUT = {
  padding: 40,
  headerH: 48,
  patrolY: 60,
  sessionY: 200,
  skillY: 380,
  patternY: 520,
  sessionW: 180,
  sessionH: 90,
  skillW: 130,
  skillH: 55,
  patternR: 24,
  patrolR: 32,
};

// Colors (Tokyo Night — used directly in SVG attributes)
const C = {
  bg:      '#1a1b26',
  surface: '#24283b',
  border:  '#3b4261',
  text:    '#c0caf5',
  muted:   '#565f89',
  accent:  '#7aa2f7',
  success: '#9ece6a',
  warning: '#e0af68',
  danger:  '#f7768e',
  info:    '#7dcfff',
  purple:  '#bb9af7',
};

// A session is "active" if it has events within this window
const ACTIVE_THRESHOLD_MS = 5 * 60 * 1000;

// ── Utility helpers ──────────────────────────────────────────

function esc(str) {
  if (str == null) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function fmtTime(iso) {
  if (!iso) return '--';
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function fmtDate(iso) {
  if (!iso) return '--';
  const d = new Date(iso);
  return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function fmtDuration(startISO, endISO) {
  if (!startISO) return '--';
  const start = new Date(startISO);
  const end = endISO ? new Date(endISO) : new Date();
  const s = Math.floor((end - start) / 1000);
  if (s < 60) return s + 's';
  const m = Math.floor(s / 60);
  if (m < 60) return m + 'm ' + (s % 60) + 's';
  return Math.floor(m / 60) + 'h ' + (m % 60) + 'm';
}

function truncate(str, len) {
  if (!str) return '';
  return str.length > len ? str.substring(0, len) + '...' : str;
}

function clamp(v, lo, hi) { return Math.max(lo, Math.min(hi, v)); }

// ── SVG namespace helper ─────────────────────────────────────

const SVG_NS = 'http://www.w3.org/2000/svg';

function svgEl(tag, attrs) {
  const el = document.createElementNS(SVG_NS, tag);
  if (attrs) {
    for (const [k, v] of Object.entries(attrs)) {
      el.setAttribute(k, v);
    }
  }
  return el;
}

// ── API fetch helper ─────────────────────────────────────────

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

// ── State ────────────────────────────────────────────────────

const state = {
  sessions: [],
  events: [],
  skills: [],
  patterns: [],
  patrolAlerts: [],
  patrolSessions: [],
  liveConnected: false,
};

// Node position state — survives re-renders so dragged positions persist.
// Keys: "patrol", "session-<id>", "skill-<name>", "pattern-<idx>"
const nodePositions = {};

// Track which nodes have been manually dragged (skip layout recalc for those)
const draggedNodes = new Set();

// Detail panel auto-refresh interval ID
let detailRefreshTimer = null;

// Currently selected node key (for highlighting)
let selectedNodeKey = null;

// Zoom/pan state
const camera = { x: 0, y: 0, scale: 1 };
const ZOOM_MIN = 0.3;
const ZOOM_MAX = 3;
const ZOOM_STEP = 0.1;

// ── DOM refs ─────────────────────────────────────────────────

const $ = (id) => document.getElementById(id);

// ── Layout calculation ───────────────────────────────────────

function calculateLayout() {
  const container = $('graph-container');
  const w = (container ? container.clientWidth : window.innerWidth) || window.innerWidth;

  // Patrol hub — always centered
  if (!draggedNodes.has('patrol')) {
    nodePositions['patrol'] = { x: w / 2, y: LAYOUT.patrolY };
  }

  // Sessions — evenly spaced
  const sessions = state.sessions;
  if (sessions.length > 0) {
    const spacing = Math.min(240, (w - 2 * LAYOUT.padding) / sessions.length);
    const totalW = spacing * (sessions.length - 1);
    const startX = (w - totalW) / 2;

    sessions.forEach((sess, i) => {
      const key = 'session-' + sess.id;
      if (!draggedNodes.has(key)) {
        nodePositions[key] = {
          x: sessions.length === 1 ? w / 2 : startX + i * spacing,
          y: LAYOUT.sessionY,
        };
      }
    });
  }

  // Skills — evenly spaced
  const skills = state.skills;
  if (skills.length > 0) {
    const spacing = Math.min(180, (w - 2 * LAYOUT.padding) / skills.length);
    const totalW = spacing * (skills.length - 1);
    const startX = (w - totalW) / 2;

    skills.forEach((skill, i) => {
      const key = 'skill-' + skill.name;
      if (!draggedNodes.has(key)) {
        nodePositions[key] = {
          x: skills.length === 1 ? w / 2 : startX + i * spacing,
          y: LAYOUT.skillY,
        };
      }
    });
  }

  // Patterns — grouped in a spread
  const patterns = state.patterns;
  if (patterns.length > 0) {
    const maxPerRow = Math.max(1, Math.floor((w - 2 * LAYOUT.padding) / 60));
    patterns.forEach((p, i) => {
      const key = 'pattern-' + i;
      if (!draggedNodes.has(key)) {
        const col = i % maxPerRow;
        const row = Math.floor(i / maxPerRow);
        const spacing = Math.min(60, (w - 2 * LAYOUT.padding) / maxPerRow);
        const totalW = spacing * (Math.min(patterns.length, maxPerRow) - 1);
        const startX = (w - totalW) / 2;
        nodePositions[key] = {
          x: startX + col * spacing,
          y: LAYOUT.patternY + row * 55,
        };
      }
    });
  }
}

// ── Edge building ────────────────────────────────────────────

function buildSessionAlertMap() {
  const map = {};
  (state.patrolSessions || []).forEach((ps) => {
    const maxSev = ps.alerts.some((a) => a.severity === 'critical') ? 'critical' : 'warning';
    map[ps.session_id] = maxSev;
  });
  return map;
}

function buildEdgesGroup() {
  const edgesGroup = svgEl('g', { class: 'edges-layer' });
  const sessions = state.sessions;
  const skills = state.skills;
  const patterns = state.patterns;
  const sessionAlertMap = buildSessionAlertMap();
  const patrolPos = nodePositions['patrol'];

  // Patrol -> Sessions
  sessions.forEach((sess) => {
    const sessPos = nodePositions['session-' + sess.id];
    if (!patrolPos || !sessPos) return;

    const alertSev = sessionAlertMap[sess.id];
    let edgeClass = 'edge';
    if (alertSev === 'critical') edgeClass += ' edge--critical';
    else if (alertSev === 'warning') edgeClass += ' edge--warning';

    edgesGroup.appendChild(svgEl('line', {
      x1: patrolPos.x, y1: patrolPos.y + LAYOUT.patrolR,
      x2: sessPos.x, y2: sessPos.y - LAYOUT.sessionH / 2,
      class: edgeClass,
    }));
  });

  // Sessions -> Skills
  sessions.forEach((sess) => {
    const sessPos = nodePositions['session-' + sess.id];
    if (!sessPos) return;
    skills.forEach((skill) => {
      const skillPos = nodePositions['skill-' + skill.name];
      if (!skillPos) return;
      edgesGroup.appendChild(svgEl('line', {
        x1: sessPos.x, y1: sessPos.y + LAYOUT.sessionH / 2,
        x2: skillPos.x, y2: skillPos.y - LAYOUT.skillH / 2,
        class: 'edge',
      }));
    });
  });

  // Skills -> Patterns (round-robin distribution)
  if (skills.length > 0) {
    patterns.forEach((p, i) => {
      const skillIdx = i % skills.length;
      const skillPos = nodePositions['skill-' + skills[skillIdx].name];
      const patternPos = nodePositions['pattern-' + i];
      if (!skillPos || !patternPos) return;
      edgesGroup.appendChild(svgEl('line', {
        x1: skillPos.x, y1: skillPos.y + LAYOUT.skillH / 2,
        x2: patternPos.x, y2: patternPos.y - LAYOUT.patternR,
        class: 'edge',
      }));
    });
  }

  return edgesGroup;
}

// ── Full graph render ────────────────────────────────────────

function renderGraph() {
  const root = $('graph-root');
  root.innerHTML = '';

  calculateLayout();
  applyCameraTransform();

  const sessions = state.sessions;
  const skills = state.skills;
  const patterns = state.patterns;
  const patrolAlerts = state.patrolAlerts;
  const sessionAlertMap = buildSessionAlertMap();
  const w = ($('graph-container') || {}).clientWidth || window.innerWidth;

  // Row labels
  addRowLabel(root, 16, LAYOUT.patrolY - 30, 'PATROL');
  if (sessions.length > 0) addRowLabel(root, 16, LAYOUT.sessionY - 55, 'SESSIONS');
  if (skills.length > 0) addRowLabel(root, 16, LAYOUT.skillY - 35, 'SKILLS');
  if (patterns.length > 0) addRowLabel(root, 16, LAYOUT.patternY - 30, 'PATTERNS');

  // Edges layer (behind nodes)
  root.appendChild(buildEdgesGroup());

  // Nodes layer
  const nodesGroup = svgEl('g', { class: 'nodes-layer' });
  root.appendChild(nodesGroup);

  // Patrol hub
  renderPatrolNode(nodesGroup, patrolAlerts);

  // Session nodes
  sessions.forEach((sess) => {
    renderSessionNode(nodesGroup, sess, sessionAlertMap[sess.id]);
  });

  // Skill nodes
  skills.forEach((skill) => {
    renderSkillNode(nodesGroup, skill);
  });

  // Pattern nodes
  patterns.forEach((pattern, i) => {
    renderPatternNode(nodesGroup, pattern, i);
  });

  // Empty state
  if (sessions.length === 0 && skills.length === 0) {
    const emptyText = svgEl('text', {
      x: w / 2,
      y: LAYOUT.sessionY,
      class: 'empty-label',
    });
    emptyText.textContent = 'Waiting for data...';
    root.appendChild(emptyText);
  }

  // Re-apply selection highlight
  if (selectedNodeKey) {
    const selected = root.querySelector('[data-node="' + selectedNodeKey + '"]');
    if (selected) selected.classList.add('node-selected');
  }
}

function applyCameraTransform() {
  const root = $('graph-root');
  if (root) {
    root.setAttribute('transform',
      'translate(' + camera.x + ',' + camera.y + ') scale(' + camera.scale + ')');
  }
  const indicator = $('zoom-indicator');
  if (indicator) indicator.textContent = Math.round(camera.scale * 100) + '%';
}

function addRowLabel(svg, x, y, text) {
  const label = svgEl('text', { x: x, y: y, class: 'row-label' });
  label.textContent = text;
  svg.appendChild(label);
}

// ── Session activity detection ───────────────────────────────

function isSessionActive(session) {
  if (session.ended_at) return false;

  const now = Date.now();
  const recentEvent = state.events.find((ev) =>
    ev.session_id === session.id &&
    (now - new Date(ev.timestamp).getTime()) < ACTIVE_THRESHOLD_MS
  );
  if (recentEvent) return true;

  if (session.started_at) {
    return (now - new Date(session.started_at).getTime()) < ACTIVE_THRESHOLD_MS;
  }
  return false;
}

// ── Individual node renderers ────────────────────────────────

function renderPatrolNode(parent, alerts) {
  const pos = nodePositions['patrol'];
  if (!pos) return;

  const hasCritical = alerts.some((a) => a.severity === 'critical');
  const hasWarning = alerts.length > 0;
  let stateClass = '';
  if (hasCritical) stateClass = 'patrol--critical';
  else if (hasWarning) stateClass = 'patrol--warning';

  const g = svgEl('g', {
    class: 'node-patrol ' + stateClass,
    transform: 'translate(' + pos.x + ', ' + pos.y + ')',
    'data-node': 'patrol',
  });

  // Octagon shape
  const r = LAYOUT.patrolR;
  const points = [];
  for (let i = 0; i < 8; i++) {
    const angle = (Math.PI / 8) + (i * Math.PI / 4);
    points.push(Math.cos(angle) * r + ',' + Math.sin(angle) * r);
  }
  const octagon = svgEl('polygon', {
    points: points.join(' '),
    class: 'node-shape',
  });
  g.appendChild(octagon);

  // Label
  const label = svgEl('text', { y: -4, class: 'node-label' });
  label.textContent = 'PATROL';
  g.appendChild(label);

  // Alert count sublabel
  const sublabel = svgEl('text', { y: 12, class: 'node-sublabel' });
  const alertCount = alerts.length;
  sublabel.textContent = alertCount === 0 ? 'All clear' : alertCount + ' alert' + (alertCount !== 1 ? 's' : '');
  g.appendChild(sublabel);

  // Interactions — click selects, dblclick also selects
  g.addEventListener('click', (e) => { e.stopPropagation(); selectNode('patrol'); openDetailPanel('patrol'); });
  g.addEventListener('mouseenter', (e) => showTooltip(e, 'Patrol Hub: ' + alertCount + ' alerts'));
  g.addEventListener('mouseleave', hideTooltip);

  setupDrag(g, 'patrol');
  parent.appendChild(g);
}

function renderSessionNode(parent, session, alertSeverity) {
  const key = 'session-' + session.id;
  const pos = nodePositions[key];
  if (!pos) return;

  const isActive = isSessionActive(session);
  const stateClass = isActive ? 'session--active' : 'session--idle';

  const g = svgEl('g', {
    class: 'node-session ' + stateClass,
    transform: 'translate(' + pos.x + ', ' + pos.y + ')',
    'data-node': key,
  });

  // Rounded rectangle (centered)
  const rect = svgEl('rect', {
    x: -LAYOUT.sessionW / 2,
    y: -LAYOUT.sessionH / 2,
    width: LAYOUT.sessionW,
    height: LAYOUT.sessionH,
    class: 'node-shape',
  });
  g.appendChild(rect);

  // Project name (primary label)
  const label = svgEl('text', { y: -22, class: 'node-label' });
  label.textContent = truncate(session.project || 'Unknown', 18);
  g.appendChild(label);

  // Session ID (sublabel)
  const sublabel = svgEl('text', { y: -6, class: 'node-sublabel' });
  sublabel.textContent = session.id.substring(0, 8) + '...';
  g.appendChild(sublabel);

  // Duration + event count
  const duration = fmtDuration(session.started_at, session.ended_at);
  const eventCount = (session.tasks_completed || 0) + (session.tasks_failed || 0);
  const stat = svgEl('text', { y: 12, class: 'node-stat' });
  stat.textContent = duration + ' | ' + eventCount + ' events';
  g.appendChild(stat);

  // Status indicator
  const statusText = svgEl('text', { y: 28, class: 'node-sublabel' });
  statusText.textContent = isActive ? 'active' : (session.ended_at ? 'ended' : 'idle');
  g.appendChild(statusText);

  // Alert badge (warning triangle in top-right corner)
  if (alertSeverity) {
    const badgeCircle = svgEl('circle', {
      cx: LAYOUT.sessionW / 2 - 10,
      cy: -LAYOUT.sessionH / 2 + 10,
      r: 8,
      fill: alertSeverity === 'critical' ? C.danger : C.warning,
      opacity: 0.9,
    });
    g.appendChild(badgeCircle);

    const badgeText = svgEl('text', {
      x: LAYOUT.sessionW / 2 - 10,
      y: -LAYOUT.sessionH / 2 + 10,
      'text-anchor': 'middle',
      'dominant-baseline': 'central',
      'font-size': '10',
      'font-weight': '600',
      fill: C.bg,
    });
    badgeText.textContent = '!';
    g.appendChild(badgeText);
  }

  // Interactions — click selects and shows detail
  g.addEventListener('click', (e) => { e.stopPropagation(); selectNode(key); openDetailPanel('session', session); });
  g.addEventListener('mouseenter', (e) => {
    const tip = session.project + ' | ' + duration + ' | ' +
      (isActive ? 'Active' : 'Idle');
    showTooltip(e, tip);
  });
  g.addEventListener('mouseleave', hideTooltip);

  setupDrag(g, key);
  parent.appendChild(g);
}

function renderSkillNode(parent, skill) {
  const key = 'skill-' + skill.name;
  const pos = nodePositions[key];
  if (!pos) return;

  const g = svgEl('g', {
    class: 'node-skill',
    transform: 'translate(' + pos.x + ', ' + pos.y + ')',
    'data-node': key,
  });

  // Rounded rectangle (centered)
  const rect = svgEl('rect', {
    x: -LAYOUT.skillW / 2,
    y: -LAYOUT.skillH / 2,
    width: LAYOUT.skillW,
    height: LAYOUT.skillH,
    class: 'node-shape',
  });
  g.appendChild(rect);

  // Skill name
  const label = svgEl('text', { y: -10, class: 'node-label' });
  label.textContent = truncate(skill.name, 14);
  g.appendChild(label);

  // Effectiveness bar inside the node
  const barW = LAYOUT.skillW - 24;
  const barH = 5;
  const barX = -barW / 2;
  const barY = 4;
  const pct = clamp(skill.effectiveness || 0, 0, 1);

  const barBg = svgEl('rect', {
    x: barX, y: barY, width: barW, height: barH,
    class: 'skill-bar-bg',
  });
  g.appendChild(barBg);

  let fillClass = 'skill-bar-fill';
  if (pct >= 0.7) fillClass += ' skill-bar-fill--high';
  else if (pct >= 0.4) fillClass += ' skill-bar-fill--mid';
  else fillClass += ' skill-bar-fill--low';

  const barFill = svgEl('rect', {
    x: barX, y: barY, width: barW * pct, height: barH,
    class: fillClass,
  });
  g.appendChild(barFill);

  // Percentage label below bar
  const pctLabel = svgEl('text', {
    y: 22,
    'text-anchor': 'middle',
    'dominant-baseline': 'central',
    'font-family': "'JetBrains Mono', monospace",
    'font-size': '9',
    fill: pct >= 0.7 ? C.success : pct >= 0.4 ? C.warning : C.danger,
  });
  pctLabel.textContent = Math.round(pct * 100) + '%';
  g.appendChild(pctLabel);

  // Interactions
  g.addEventListener('click', (e) => { e.stopPropagation(); selectNode(key); openDetailPanel('skill', skill); });
  g.addEventListener('mouseenter', (e) => {
    showTooltip(e, skill.name + ': ' + Math.round(pct * 100) + '% effective (' + (skill.total || 0) + ' invocations)');
  });
  g.addEventListener('mouseleave', hideTooltip);

  setupDrag(g, key);
  parent.appendChild(g);
}

function renderPatternNode(parent, pattern, index) {
  const key = 'pattern-' + index;
  const pos = nodePositions[key];
  if (!pos) return;

  const conf = pattern.confidence || 0;

  const g = svgEl('g', {
    class: 'node-pattern',
    transform: 'translate(' + pos.x + ', ' + pos.y + ')',
    'data-node': key,
  });

  // Circle with opacity based on confidence
  const circle = svgEl('circle', {
    r: LAYOUT.patternR,
    class: 'node-shape',
    'fill-opacity': clamp(0.3 + conf * 0.7, 0.3, 1),
  });
  if (conf >= 0.7) circle.setAttribute('stroke', C.purple);
  g.appendChild(circle);

  // Confidence number inside the circle
  const confLabel = svgEl('text', {
    y: 0,
    'text-anchor': 'middle',
    'dominant-baseline': 'central',
    'font-family': "'JetBrains Mono', monospace",
    'font-size': '9',
    fill: C.purple,
  });
  confLabel.textContent = Math.round(conf * 100);
  g.appendChild(confLabel);

  // Label below (visible on hover via CSS)
  const label = svgEl('text', { y: LAYOUT.patternR + 14, class: 'node-label' });
  label.textContent = truncate(pattern.scope || pattern.content, 12);
  g.appendChild(label);

  // Interactions
  g.addEventListener('click', (e) => { e.stopPropagation(); selectNode(key); openDetailPanel('pattern', pattern); });
  g.addEventListener('mouseenter', (e) => {
    const tip = (pattern.scope || 'pattern') + ': ' + truncate(pattern.content, 60) +
      ' (' + Math.round(conf * 100) + '% confidence)';
    showTooltip(e, tip);
  });
  g.addEventListener('mouseleave', hideTooltip);

  parent.appendChild(g);
}

// ── Drag handling ────────────────────────────────────────────

function setupDrag(gElement, nodeKey) {
  let dragging = false;
  let startMouse = { x: 0, y: 0 };
  let startPos = { x: 0, y: 0 };

  gElement.addEventListener('mousedown', (e) => {
    if (e.button !== 0) return;
    e.preventDefault();
    e.stopPropagation();

    dragging = true;
    startMouse = { x: e.clientX, y: e.clientY };
    startPos = { x: nodePositions[nodeKey].x, y: nodePositions[nodeKey].y };

    const onMove = (me) => {
      if (!dragging) return;
      const dx = me.clientX - startMouse.x;
      const dy = me.clientY - startMouse.y;
      const newX = startPos.x + dx;
      const newY = startPos.y + dy;

      nodePositions[nodeKey] = { x: newX, y: newY };
      gElement.setAttribute('transform', 'translate(' + newX + ', ' + newY + ')');

      // Re-render edges (lightweight operation)
      updateEdgesForNode();
    };

    const onUp = () => {
      dragging = false;
      const pos = nodePositions[nodeKey];
      if (pos.x !== startPos.x || pos.y !== startPos.y) {
        draggedNodes.add(nodeKey);
      }
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
    };

    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  });
}

function updateEdgesForNode() {
  const root = $('graph-root');
  const oldEdges = root.querySelector('.edges-layer');
  if (oldEdges) {
    const newEdges = buildEdgesGroup();
    root.replaceChild(newEdges, oldEdges);
  }
}

// ── Tooltip ──────────────────────────────────────────────────

function showTooltip(event, text) {
  const tip = $('tooltip');
  tip.textContent = text;
  tip.classList.add('tooltip--visible');
  positionTooltip(event);
}

function hideTooltip() {
  const tip = $('tooltip');
  tip.classList.remove('tooltip--visible');
}

function positionTooltip(event) {
  const tip = $('tooltip');
  const x = event.clientX + 12;
  const y = event.clientY - 8;
  tip.style.left = x + 'px';
  tip.style.top = y + 'px';
}

document.addEventListener('mousemove', (e) => {
  const tip = $('tooltip');
  if (tip && tip.classList.contains('tooltip--visible')) {
    positionTooltip(e);
  }
});

// ── Node selection ────────────────────────────────────────────

function selectNode(nodeKey) {
  // Remove previous selection
  const root = $('graph-root');
  if (root) {
    const prev = root.querySelector('.node-selected');
    if (prev) prev.classList.remove('node-selected');
  }

  selectedNodeKey = nodeKey;

  // Add selection to new node
  if (root && nodeKey) {
    const node = root.querySelector('[data-node="' + nodeKey + '"]');
    if (node) node.classList.add('node-selected');
  }
}

// ── Zoom / Pan ───────────────────────────────────────────────

function setupZoomPan() {
  const container = $('graph-container');
  if (!container) return;

  // Mouse wheel zoom
  container.addEventListener('wheel', function(e) {
    e.preventDefault();
    const delta = e.deltaY > 0 ? -ZOOM_STEP : ZOOM_STEP;
    const newScale = clamp(camera.scale + delta, ZOOM_MIN, ZOOM_MAX);

    // Zoom toward cursor position
    const rect = container.getBoundingClientRect();
    const mx = e.clientX - rect.left;
    const my = e.clientY - rect.top;

    // Adjust pan so zoom centers on cursor
    const scaleRatio = newScale / camera.scale;
    camera.x = mx - scaleRatio * (mx - camera.x);
    camera.y = my - scaleRatio * (my - camera.y);
    camera.scale = newScale;

    applyCameraTransform();
  }, { passive: false });

  // Middle-click or right-click + drag to pan
  let panning = false;
  let panStart = { x: 0, y: 0 };
  let camStart = { x: 0, y: 0 };

  container.addEventListener('mousedown', function(e) {
    // Middle click (button 1) or right click (button 2) or left click on empty space
    if (e.button === 1 || e.button === 2 || (e.button === 0 && e.target === $('graph'))) {
      e.preventDefault();
      panning = true;
      panStart = { x: e.clientX, y: e.clientY };
      camStart = { x: camera.x, y: camera.y };
    }
  });

  document.addEventListener('mousemove', function(e) {
    if (!panning) return;
    camera.x = camStart.x + (e.clientX - panStart.x);
    camera.y = camStart.y + (e.clientY - panStart.y);
    applyCameraTransform();
  });

  document.addEventListener('mouseup', function() {
    panning = false;
  });

  // Prevent context menu on right-click in graph
  container.addEventListener('contextmenu', function(e) {
    e.preventDefault();
  });

  // Double-click to reset zoom
  container.addEventListener('dblclick', function(e) {
    if (e.target === $('graph')) {
      camera.x = 0;
      camera.y = 0;
      camera.scale = 1;
      applyCameraTransform();
    }
  });
}

// ── Detail Panel ─────────────────────────────────────────────

function openDetailPanel(type, data) {
  const title = $('detail-title');
  const body = $('detail-body');

  // Clear any existing auto-refresh
  clearDetailRefresh();
  clearTranscriptRefresh();

  switch (type) {
    case 'patrol':
      title.textContent = 'Patrol Alerts';
      body.innerHTML = renderPatrolDetail();
      break;
    case 'session':
      title.textContent = 'Session: ' + esc(data.project);
      loadSessionDetail(data);
      break;
    case 'skill':
      title.textContent = 'Skill: ' + esc(data.name);
      body.innerHTML = renderSkillDetail(data);
      break;
    case 'pattern':
      title.textContent = 'Learned Pattern';
      body.innerHTML = renderPatternDetail(data);
      break;
  }
}

function closeDetailPanel() {
  clearDetailRefresh();
  clearTranscriptRefresh();
  selectedNodeKey = null;
  const body = $('detail-body');
  const title = $('detail-title');
  if (title) title.textContent = 'Select a node';
  if (body) body.innerHTML = '<p class="detail-empty">Double-click any node to view details.</p>';
  // Remove selection highlight
  const root = $('graph-root');
  if (root) {
    const prev = root.querySelector('.node-selected');
    if (prev) prev.classList.remove('node-selected');
  }
}

function clearDetailRefresh() {
  if (detailRefreshTimer) {
    clearInterval(detailRefreshTimer);
    detailRefreshTimer = null;
  }
}

// Fetch session events from API and render detail panel
async function loadSessionDetail(session) {
  const body = $('detail-body');
  body.innerHTML = '<p style="color:' + C.muted + ';font-size:11px;">Loading events...</p>';

  const data = await apiFetch('/api/events?session_id=' + encodeURIComponent(session.id));
  const events = (data && data.events) ? data.events : [];

  body.innerHTML = renderSessionDetail(session, events);

  // Auto-refresh every 3 seconds for active sessions
  if (isSessionActive(session)) {
    detailRefreshTimer = setInterval(async () => {
      const freshData = await apiFetch('/api/events?session_id=' + encodeURIComponent(session.id));
      const freshEvents = (freshData && freshData.events) ? freshData.events : [];
      const currentBody = $('detail-body');
      if (currentBody) {
        currentBody.innerHTML = renderSessionDetail(session, freshEvents);
      }
    }, 3000);
  }
}

function renderPatrolDetail() {
  const alerts = state.patrolAlerts;
  const sessions = state.patrolSessions || [];

  if (alerts.length === 0) {
    return '<div class="detail-section"><p style="color:' + C.success + ';font-size:12px;">All clear. No anti-patterns detected.</p></div>';
  }

  let html = '';
  sessions.forEach((sess) => {
    html += '<div class="detail-section">';
    html += '<div class="detail-section__title">' +
      esc(sess.session_id.substring(0, 12)) +
      (sess.project ? ' - ' + esc(sess.project) : '') +
      '</div>';

    sess.alerts.forEach((alert) => {
      html += '<div class="detail-alert detail-alert--' + esc(alert.severity) + '">';
      html += '<div class="detail-alert__severity detail-alert__severity--' + esc(alert.severity) + '">' + esc(alert.severity) + '</div>';
      html += '<div class="detail-alert__pattern">' + esc(alert.pattern) + '</div>';
      html += '<div class="detail-alert__message">' + esc(alert.message) + '</div>';
      if (alert.suggestion) {
        html += '<div class="detail-alert__suggestion">Suggestion: ' + esc(alert.suggestion) + '</div>';
      }
      html += '</div>';
    });

    html += '</div>';
  });

  return html;
}

function renderSessionDetail(session, events) {
  const duration = fmtDuration(session.started_at, session.ended_at);
  const isActive = isSessionActive(session);
  const statusColor = isActive ? 'success' : (session.ended_at ? 'danger' : 'warning');
  const statusLabel = isActive ? 'Active' : (session.ended_at ? 'Ended' : 'Idle');

  let html = '<div class="detail-section">';
  html += '<div class="detail-section__title">Session Info</div>';
  html += detailStat('Project', esc(session.project), 'accent');
  html += detailStat('Session ID', esc(session.id.substring(0, 16)) + '...');
  html += detailStat('Status', statusLabel, statusColor);
  html += detailStat('Duration', duration);
  html += detailStat('Started', fmtDate(session.started_at));
  if (session.ended_at) html += detailStat('Ended', fmtDate(session.ended_at));
  html += detailStat('Tasks Completed', String(session.tasks_completed || 0), 'success');
  html += detailStat('Tasks Failed', String(session.tasks_failed || 0), 'danger');
  html += '</div>';

  // Transcript button
  if (session.transcript_path) {
    html += '<div class="detail-section">';
    html += '<button class="detail-btn" onclick="loadTranscript(\'' + esc(session.id) + '\')">View Live Chat</button>';
    html += '</div>';
  }

  // Event timeline
  html += '<div class="detail-section">';
  html += '<div class="detail-section__title">Event Timeline (' + events.length + ')</div>';

  if (events.length === 0) {
    html += '<p style="color:' + C.muted + ';font-size:11px;">No events recorded for this session.</p>';
  } else {
    events.slice(0, 100).forEach((ev) => {
      const resultColor = getResultColor(ev.result);
      html += '<div class="detail-event">';
      html += '<span class="detail-event__type">' + esc(ev.type) + '</span>';
      html += '<span class="detail-event__time">' + fmtTime(ev.timestamp) + '</span>';
      if (ev.result) {
        html += '<div class="detail-event__result"><span class="badge badge--' + resultColor + '">' + esc(ev.result) + '</span></div>';
      }
      if (ev.details) {
        html += '<div class="detail-event__details">' + esc(truncate(ev.details, 200)) + '</div>';
      }
      html += '</div>';
    });
  }
  html += '</div>';

  return html;
}

function renderSkillDetail(skill) {
  const pct = Math.round((skill.effectiveness || 0) * 100);
  const color = skill.effectiveness >= 0.7 ? 'success' : skill.effectiveness >= 0.4 ? 'warning' : 'danger';

  let html = '<div class="detail-section">';
  html += '<div class="detail-section__title">Skill Info</div>';
  html += detailStat('Name', esc(skill.name), 'accent');
  html += detailStat('Effectiveness', pct + '%', color);
  html += detailStat('Total Invocations', String(skill.total || 0));
  html += detailStat('Successes', String(skill.successes || 0), 'success');
  html += detailStat('Failures', String(skill.failures || 0), 'danger');

  // Effectiveness bar
  html += '<div class="detail-skill-bar">';
  html += '<div class="detail-skill-bar__fill" style="width:' + pct + '%;background:var(--' + color + ')"></div>';
  html += '</div>';

  html += '</div>';

  return html;
}

function renderPatternDetail(pattern) {
  const conf = Math.round((pattern.confidence || 0) * 100);

  let html = '<div class="detail-section">';
  html += '<div class="detail-section__title">Pattern Info</div>';
  html += detailStat('Scope', esc(pattern.scope || 'unknown'), 'purple');
  html += detailStat('Confidence', conf + '%');
  html += '</div>';

  html += '<div class="detail-section">';
  html += '<div class="detail-section__title">Content</div>';
  html += '<p style="color:' + C.text + ';font-size:11px;line-height:1.6;word-break:break-word;">' + esc(pattern.content) + '</p>';
  html += '</div>';

  if (pattern.evidence) {
    html += '<div class="detail-section">';
    html += '<div class="detail-section__title">Evidence</div>';
    html += '<p style="color:' + C.muted + ';font-size:10px;line-height:1.5;word-break:break-word;">' + esc(pattern.evidence) + '</p>';
    html += '</div>';
  }

  return html;
}

function detailStat(label, value, colorModifier) {
  const cls = colorModifier ? ' detail-stat__value--' + colorModifier : '';
  return '<div class="detail-stat">' +
    '<span class="detail-stat__label">' + label + '</span>' +
    '<span class="detail-stat__value' + cls + '">' + value + '</span>' +
    '</div>';
}

function getResultColor(result) {
  if (!result) return 'muted';
  const r = result.toLowerCase();
  if (r === 'success' || r === 'verified') return 'success';
  if (r === 'failure' || r === 'failed' || r === 'error') return 'danger';
  if (r === 'warning') return 'warning';
  return 'info';
}

// ── Transcript viewer ────────────────────────────────────────

let transcriptRefreshTimer = null;

async function loadTranscript(sessionId) {
  const body = $('detail-body');
  const title = $('detail-title');
  title.textContent = 'Live Chat';
  body.innerHTML = '<p style="color:' + C.muted + ';font-size:11px;">Loading transcript...</p>';

  await renderTranscript(sessionId);

  // Auto-refresh every 3 seconds for live chat
  clearTranscriptRefresh();
  transcriptRefreshTimer = setInterval(function() {
    renderTranscript(sessionId);
  }, 3000);
}

function clearTranscriptRefresh() {
  if (transcriptRefreshTimer) {
    clearInterval(transcriptRefreshTimer);
    transcriptRefreshTimer = null;
  }
}

async function renderTranscript(sessionId) {
  const data = await apiFetch('/api/transcript?session_id=' + encodeURIComponent(sessionId));
  const body = $('detail-body');
  if (!body) return;

  if (!data || data.error) {
    body.innerHTML = '<p style="color:' + C.danger + ';font-size:11px;">' + esc(data ? data.error : 'Failed to load') + '</p>';
    return;
  }

  const messages = data.messages || [];
  if (messages.length === 0) {
    body.innerHTML = '<p style="color:' + C.muted + ';font-size:11px;">No messages yet.</p>';
    return;
  }

  // Scroll position preservation
  const wasAtBottom = body.scrollHeight - body.scrollTop - body.clientHeight < 50;

  let html = '<div class="transcript">';
  messages.forEach(function(msg) {
    if (msg.type === 'user') {
      html += '<div class="chat-msg chat-msg--user">';
      html += '<div class="chat-msg__role">You</div>';
      html += '<div class="chat-msg__text">' + escAndFormat(msg.text) + '</div>';
      if (msg.timestamp) html += '<div class="chat-msg__time">' + fmtTime(msg.timestamp) + '</div>';
      html += '</div>';
    } else if (msg.type === 'assistant') {
      html += '<div class="chat-msg chat-msg--assistant">';
      html += '<div class="chat-msg__role">Claude</div>';
      html += '<div class="chat-msg__text">' + escAndFormat(msg.text) + '</div>';
      if (msg.timestamp) html += '<div class="chat-msg__time">' + fmtTime(msg.timestamp) + '</div>';
      html += '</div>';
    } else if (msg.type === 'tool_call') {
      html += '<div class="chat-msg chat-msg--tool">';
      html += '<div class="chat-msg__role">' + esc(msg.tool_name || 'Tool') + '</div>';
      if (msg.tool_input) {
        html += '<div class="chat-msg__code">' + esc(truncate(msg.tool_input, 300)) + '</div>';
      }
      html += '</div>';
    } else if (msg.type === 'tool_result') {
      html += '<div class="chat-msg chat-msg--result">';
      html += '<div class="chat-msg__code">' + esc(truncate(msg.text, 300)) + '</div>';
      html += '</div>';
    }
  });
  html += '</div>';

  body.innerHTML = html;

  // Auto-scroll to bottom if user was at bottom
  if (wasAtBottom) {
    body.scrollTop = body.scrollHeight;
  }
}

function escAndFormat(text) {
  // Escape HTML then add basic formatting
  let s = esc(text);
  // Code blocks
  s = s.replace(/```([^`]*?)```/g, '<pre class="chat-code">$1</pre>');
  // Inline code
  s = s.replace(/`([^`]+)`/g, '<code class="chat-inline-code">$1</code>');
  // Bold
  s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  // Newlines
  s = s.replace(/\n/g, '<br>');
  return s;
}

// ── Live duration timer ──────────────────────────────────────

// Update duration text in session nodes every second without full re-render
setInterval(function() {
  const root = $('graph-root');
  if (!root) return;

  state.sessions.forEach(function(session) {
    const key = 'session-' + session.id;
    const g = root.querySelector('[data-node="' + key + '"]');
    if (!g) return;

    // Find the stat text element (3rd text = duration line)
    const texts = g.querySelectorAll('text.node-stat');
    if (texts.length === 0) return;

    const duration = fmtDuration(session.started_at, session.ended_at);
    const eventCount = state.events.filter(function(e) { return e.session_id === session.id; }).length || 0;
    texts[0].textContent = duration + ' | ' + eventCount + ' events';
  });
}, 1000);

// ── Data loading ─────────────────────────────────────────────

async function loadSessions() {
  const data = await apiFetch('/api/sessions?limit=10');
  if (data && data.sessions) {
    state.sessions = data.sessions;
  }
}

async function loadRecentEvents() {
  const data = await apiFetch('/api/events/recent?limit=50');
  if (data && data.events) {
    state.events = data.events;
  }
}

async function loadSkills() {
  const data = await apiFetch('/api/skills');
  if (data && data.skills) {
    state.skills = data.skills;
  }
}

async function loadPatterns() {
  const data = await apiFetch('/api/workflow');
  if (data && data.patterns) {
    state.patterns = data.patterns;
  }
}

async function loadPatrol() {
  const data = await apiFetch('/api/patrol');
  if (data) {
    state.patrolAlerts = data.alerts || [];
    state.patrolSessions = data.sessions || [];
  }
}

async function loadAll() {
  await Promise.all([
    loadSessions(),
    loadRecentEvents(),
    loadSkills(),
    loadPatterns(),
    loadPatrol(),
  ]);
  renderGraph();
}

// ── SSE connection ───────────────────────────────────────────

let refreshTimer = null;

function debouncedRefresh() {
  clearTimeout(refreshTimer);
  refreshTimer = setTimeout(() => {
    loadAll();
  }, 2000);
}

function prependEvent(ev) {
  state.events.unshift(ev);
  if (state.events.length > 200) state.events.pop();
  debouncedRefresh();
}

function connectSSE() {
  const liveDot = $('live-dot');
  const liveText = $('live-text');

  const es = new EventSource('/sse/events');

  es.onopen = () => {
    state.liveConnected = true;
    if (liveDot) liveDot.className = 'live-dot';
    if (liveText) {
      liveText.textContent = 'Live';
      liveText.className = 'live-text';
    }
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
    if (liveDot) liveDot.className = 'live-dot live-dot--disconnected';
    if (liveText) {
      liveText.textContent = 'Disconnected';
      liveText.className = 'live-text live-text--disconnected';
    }
    // EventSource auto-retries
  };
}

// ── Window resize handler ────────────────────────────────────

let resizeTimer = null;
function onResize() {
  clearTimeout(resizeTimer);
  resizeTimer = setTimeout(() => {
    renderGraph();
  }, 150);
}

// ── Init ─────────────────────────────────────────────────────

async function init() {
  // Set up zoom/pan on the graph canvas
  setupZoomPan();

  // Load all data and render
  await loadAll();

  // Connect SSE for live updates
  connectSSE();

  // Escape clears selection
  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') closeDetailPanel();
  });

  // Resize handler
  window.addEventListener('resize', onResize);

  // Poll every 5 seconds for fresh data
  setInterval(loadAll, 5000);
}

document.addEventListener('DOMContentLoaded', init);
