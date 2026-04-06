/* ============================================================
   NETWATCH.AZ — Dashboard Logic
   ============================================================ */

const API = '/api/v1';
const $  = s => document.querySelector(s);
const esc = s => String(s).replace(/[<>&"]/g, c =>
    ({ '<':'&lt;', '>':'&gt;', '&':'&amp;', '"':'&quot;' }[c]));

function timeStr() {
    return new Date().toLocaleTimeString('en-GB', { hour12: false });
}

async function fetchJSON(url) {
    try {
        const r = await fetch(url);
        if (!r.ok) return null;
        return await r.json();
    } catch { return null; }
}

// ── MAP ────────────────────────────────────────────────────

function buildMap(nodes) {
    const el = $('#azMapContainer');
    if (!el) return;

    const W = 900, H = 300;
    const BX = 450, BY = 150;

    const peers = {
        'node-eu-central': { x: 150, y:  80, label: 'FRANKFURT' },
        'node-us':         { x:  80, y: 220, label: 'ASHBURN'   },
        'node-asia':       { x: 780, y: 240, label: 'MUMBAI'    },
    };

    let svg = `<svg viewBox="0 0 ${W} ${H}" xmlns="http://www.w3.org/2000/svg">`;
    svg += `<path class="az-silhouette" d="M340 110 Q380 80 430 110 T560 125 Q650 100 610 165 T480 195 Q390 195 340 160 Z"/>`;

    // Arcs to peer nodes
    if (nodes && nodes.length > 0) {
        nodes.forEach(n => {
            const p = peers[n.node_id];
            if (!p) return;
            const mx = (p.x + BX) / 2;
            const my = Math.min(p.y, BY) - 60;
            const color = n.is_alive ? 'var(--cyan)' : 'var(--purple)';
            svg += `<path d="M${p.x} ${p.y} Q${mx} ${my} ${BX} ${BY}"
                         fill="none" stroke="${color}" stroke-width="1.5"
                         stroke-dasharray="4 5" opacity="0.5"/>`;
            svg += `<circle cx="${p.x}" cy="${p.y}" r="18"
                            fill="${n.is_alive ? 'rgba(34,211,238,0.1)' : 'rgba(167,139,250,0.1)'}"/>`;
            svg += `<circle cx="${p.x}" cy="${p.y}" r="4"
                            fill="${n.is_alive ? 'var(--cyan)' : 'var(--purple)'}"/>`;
            svg += `<text x="${p.x}" y="${p.y + 18}" text-anchor="middle" class="node-label">${p.label}</text>`;
        });
    }

    // Baku hub
    svg += `<circle cx="${BX}" cy="${BY}" r="22" class="map-node-glow"/>`;
    svg += `<circle cx="${BX}" cy="${BY}" r="10" class="map-pulse"/>`;
    svg += `<circle cx="${BX}" cy="${BY}" r="5"  class="map-node"/>`;
    svg += `<text x="${BX}" y="${BY - 20}" text-anchor="middle" class="node-label"
                  style="fill:var(--cyan);font-weight:700">BAKU</text>`;
    svg += `</svg>`;

    el.innerHTML = svg;
}

// ── SYSTEM HEALTH ──────────────────────────────────────────

const knownSystems = {
    'mygov':          'myGov',
    'egov':           'E-Government',
    'asan':           'ASAN',
    'sima':           'SIMA',
    'esosial':        'E-Sosial',
    'abb':            'ABB Bank',
    'kapitalbank':    'Kapital Bank',
    'm10':            'm10 / PashaPay',
    'leo':            'Leobank',
    'leobank':        'LeoBank',
    'azercell':       'Azercell',
    'bakcell':        'Bakcell',
    'nar':            'Nar Mobile',
    'agtelecom':      'Aztelekom',
    'baktelecom':     'Baktelecom',
    'katv':           'KATV1',
    'citynet':        'CityNet',
    'cbar':           'Central Bank',
    'ibar':           'International Bank',
    'bakumetro':      'Baku Metro',
    'bakubus':        'BakuBus',
    'bankrespublika': 'Bank Respublika',
    'xalqbank':       'Xalq Bank',
    'accessbank':     'AccessBank',
    'unibank':        'Unibank',
    'expressbank':    'Expressbank',
    'nikoil':         'Nikoil Bank',
    'yelo':           'Yelo Bank',
    'azerturkbank':   'AzerTurkBank',
    'pashabank':      'PashaBank',
    'socar':          'SOCAR',
    'azal':           'AZAL Airlines',
    'epoint':         'ePoint',
    'azerishiq':      'Azerishiq',
    'azeriqaz':       'Azeriqaz',
    'azersu':         'Azersu',
    'ady':            'ADY Railways',
    'bina':           'Bina.az',
    'turbo':          'Turbo.az',
    'tap':            'Tap.az',
    'umico':          'Umico',
    'iticket':        'iTicket.az',
    'lalafo':         'Lalafo.az',
    'azparking':      'AzParking',
    'smsradar':       'SMS Radar',
};

function dotClass(status) {
    if (status === 'HEALTHY') return 'healthy';
    if (status === 'DEGRADED') return 'degraded';
    return 'outage';
}

function statusLabel(s) {
    if (s === 'HEALTHY') return `<span class="sys-ep-status" style="color:var(--green)">OK</span>`;
    const conf = s.confidence ? ` ${(s.confidence*100).toFixed(0)}%` : '';
    return `<span class="sys-ep-status" style="color:var(--purple)">${s.status}${conf}</span>`;
}

function renderSystemHealth(statuses) {
    const list = $('#systemList');
    if (!list) return;

    const real = (statuses || []).filter(s => s.target.category !== 'ANCHOR');

    if (real.length === 0) {
        list.innerHTML = `<div style="padding:24px;text-align:center;color:var(--muted)">No data.</div>`;
        return;
    }

    // Group by parent_system
    const groups = {};
    const standalone = [];

    real.forEach(s => {
        const sys = s.target.parent_system;
        if (sys) {
            if (!groups[sys]) groups[sys] = [];
            groups[sys].push(s);
        } else {
            standalone.push(s);
        }
    });

    // Update KPI
    const kpiSys = $('#kpiSystems');
    if (kpiSys) kpiSys.textContent = Object.keys(groups).length + standalone.length;

    // Update health badge
    const allHealthy = real.every(s => s.status === 'HEALTHY');
    const badge = $('#healthBadge');
    if (badge) {
        badge.textContent = allHealthy ? `${real.length} OK` : 'Issues detected';
        badge.className = `badge ${allHealthy ? 'ok' : 'warn'}`;
    }

    let html = '';

    // Grouped systems
    Object.keys(groups).sort().forEach(sys => {
        const items  = groups[sys];
        const label  = knownSystems[sys] || items[0].target.display_name_en?.split(' ')[0] || sys.toUpperCase();
        const healthy = items.filter(i => i.status === 'HEALTHY').length;
        const worst  = items.some(i => i.status !== 'HEALTHY');
        const dc     = worst ? (items.some(i => i.status.includes('OUTAGE')) ? 'outage' : 'degraded') : 'healthy';

        html += `
        <div class="sys-group-header">
            <span class="dot ${dc}"></span>
            <span class="sys-group-name">${esc(label)}</span>
            <span class="sys-group-ratio">${healthy}/${items.length}</span>
        </div>`;

        items.forEach(c => {
            const name = c.target.display_name_en || c.target.display_name
                         || c.target.url.replace(/^https?:\/\//, '');
            html += `
        <div class="sys-endpoint" style="cursor:pointer"
             onclick="openDetail('${esc(c.target.url)}','${esc(name)}')">
            <span class="dot ${dotClass(c.status)}"></span>
            <span class="sys-ep-name">${esc(name)}</span>
            ${statusLabel(c)}
        </div>`;
        });
    });

    // Standalone
    if (standalone.length > 0) {
        standalone.forEach(c => {
            const name = c.target.display_name_en || c.target.display_name
                         || c.target.url.replace(/^https?:\/\//, '');
            html += `
        <div class="sys-standalone" style="cursor:pointer"
             onclick="openDetail('${esc(c.target.url)}','${esc(name)}')">
            <span class="dot ${dotClass(c.status)}"></span>
            <span class="sys-ep-name">${esc(name)}</span>
            ${statusLabel(c)}
        </div>`;
        });
    }

    list.innerHTML = html;
}

// ── INCIDENTS ──────────────────────────────────────────────

function renderIncidents(incidents) {
    const grid = $('#incidentGrid');
    if (!grid) return;

    const active = (incidents || []).filter(i => !i.resolved_at).slice(0, 4);

    if (active.length === 0) {
        grid.innerHTML = `
        <div class="incident-empty" style="grid-column:1/-1;background:var(--surface);border:1px solid var(--border);border-radius:10px;">
            No active incidents — all systems nominal
        </div>`;
        return;
    }

    grid.innerHTML = active.map(inc => `
    <div class="incident-card active">
        <div class="inc-target">${esc(inc.target_url.replace(/^https?:\/\//, ''))}</div>
        <span class="inc-status">${inc.peak_status.replace('_', ' ')}</span>
        <div class="inc-meta">
            Confidence ${Math.round(inc.peak_confidence * 100)}%
            &nbsp;·&nbsp; ${inc.started_at ? new Date(inc.started_at).toLocaleTimeString('en-GB') + ' UTC' : ''}
        </div>
    </div>`).join('');
}

// ── KPIS ───────────────────────────────────────────────────

function renderKPIs(statuses, bgpEvents, incidents) {
    const real = (statuses || []).filter(s => s.target.category !== 'ANCHOR');
    const healthyCount = real.filter(s => s.status === 'HEALTHY').length;
    const uptimePct = real.length > 0 ? (healthyCount / real.length * 100) : 0;
    const openIncs  = (incidents || []).filter(i => !i.resolved_at).length;
    const bgpCount  = (bgpEvents  || []).filter(e => e.event_type === 'WITHDRAW').length;

    const isOK = openIncs === 0;

    const conf = $('#kpiConfidence');
    if (conf) {
        conf.textContent = `${uptimePct.toFixed(1)}%`;
        conf.className = `kpi-value ${uptimePct === 100 ? 'ok' : 'warn'}`;
    }

    const te = $('#topEndpoints');   if (te) te.textContent = real.length;
    const ti = $('#topIncidents');   if (ti) ti.textContent = openIncs;
    const ts = $('#topLastScan');    if (ts) ts.textContent = timeStr();
    const ke = $('#kpiEndpoints');   if (ke) ke.textContent = real.length;

    const tStat = $('#topStatus');
    if (tStat) {
        tStat.textContent = isOK ? 'STABLE' : 'DEGRADED';
        tStat.style.color  = isOK ? 'var(--green)' : 'var(--purple)';
    }

    const dot = $('#statDot');
    if (dot) dot.style.background = isOK ? 'var(--green)' : 'var(--purple)';

    const kb = $('#kpiBGP');
    if (kb) {
        kb.textContent = bgpCount;
        kb.className = `kpi-value ${bgpCount > 0 ? 'warn' : 'ok'}`;
    }
}

// ── MAIN LOOP ──────────────────────────────────────────────

async function refresh() {
    const [statuses, nodes, bgp, incidents] = await Promise.all([
        fetchJSON(`${API}/status`),
        fetchJSON(`${API}/nodes`),
        fetchJSON(`${API}/bgp/events?hours=4`),
        fetchJSON(`${API}/incidents?limit=10`),
    ]);

    renderKPIs(statuses, bgp, incidents);
    renderSystemHealth(statuses);
    buildMap(nodes);
    renderIncidents(incidents);
}

// ── DETAIL PANEL ──────────────────────────────────────────

function targetParam(url) {
    // Strip protocol so the Go handler can reconstruct it
    return encodeURIComponent(url.replace(/^https?:\/\//, ''));
}

function fmtTime(iso) {
    if (!iso) return '—';
    return new Date(iso).toLocaleTimeString('en-GB', { hour12: false });
}

function fmtDate(iso) {
    if (!iso) return '—';
    const d = new Date(iso);
    return d.toLocaleDateString('en-GB', { day:'2-digit', month:'short' })
           + ' ' + d.toLocaleTimeString('en-GB', { hour:'2-digit', minute:'2-digit', hour12:false });
}

// Compute stats from history array
function analyzeHistory(history) {
    const valid = history.filter(h => h.tcp_success && h.total_ms > 0);
    const total = history.length;
    const failed = history.filter(h => !h.tcp_success).length;

    if (valid.length === 0) return { avg:0, p50:0, p95:0, p99:0, uptime:'0.0', total, failed };

    const sorted = [...valid.map(h => h.total_ms)].sort((a,b) => a-b);
    const avg  = Math.round(sorted.reduce((a,b) => a+b, 0) / sorted.length);
    const p50  = sorted[Math.floor(sorted.length * 0.50)] || 0;
    const p95  = sorted[Math.floor(sorted.length * 0.95)] || 0;
    const p99  = sorted[Math.floor(sorted.length * 0.99)] || 0;
    const uptime = ((total - failed) / total * 100).toFixed(1);

    return { avg, p50, p95, p99, uptime, total, failed };
}

// Build SVG sparkline from history
function sparkline(history) {
    const pts = history
        .slice(-120)
        .map(h => ({ ms: h.tcp_success && h.total_ms > 0 ? h.total_ms : null, ok: h.tcp_success, t: h.time }));

    const values = pts.filter(p => p.ms !== null).map(p => p.ms);
    if (values.length < 2) return '<div style="color:var(--muted);font-size:12px;padding:8px 0">Not enough data</div>';

    const W = 440, H = 64, pad = 4;
    const maxMs = Math.max(...values);
    const minMs = Math.min(...values);
    const range = (maxMs - minMs) || 1;

    const xOf = i => pad + (i / (pts.length - 1)) * (W - pad * 2);
    const yOf = ms => pad + (H - pad * 2) - ((ms - minMs) / range) * (H - pad * 2);

    // Build the line through non-null points only
    let path = '';
    let area = '';
    let first = true;
    pts.forEach((p, i) => {
        if (p.ms === null) { first = true; return; }
        const x = xOf(i).toFixed(1), y = yOf(p.ms).toFixed(1);
        path += first ? `M${x} ${y}` : ` L${x} ${y}`;
        first = false;
    });

    // Closed area path for gradient fill
    const validIdx = pts.map((p,i) => p.ms !== null ? i : null).filter(i => i !== null);
    if (validIdx.length > 1) {
        const fx = xOf(validIdx[0]).toFixed(1), fy = yOf(pts[validIdx[0]].ms).toFixed(1);
        const lx = xOf(validIdx[validIdx.length-1]).toFixed(1);
        area = path + ` L${lx} ${(pad+H-pad*2).toFixed(1)} L${fx} ${(pad+H-pad*2).toFixed(1)} Z`;
    }

    // Failed probe markers
    const failDots = pts
        .map((p, i) => (!p.ok ? `<circle cx="${xOf(i).toFixed(1)}" cy="${(H-3).toFixed(1)}" r="2.5" fill="var(--red)" opacity="0.8"/>` : ''))
        .join('');

    // Y-axis labels
    const yLabels = `
        <text x="${W-2}" y="${pad+8}" text-anchor="end" font-size="9" fill="var(--muted)" font-family="var(--font)">${maxMs}ms</text>
        <text x="${W-2}" y="${H-2}" text-anchor="end" font-size="9" fill="var(--muted)" font-family="var(--font)">${minMs}ms</text>`;

    return `
    <svg viewBox="0 0 ${W} ${H}" style="width:100%;height:${H}px;display:block">
        <defs>
            <linearGradient id="sg" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%"   stop-color="var(--cyan)" stop-opacity="0.18"/>
                <stop offset="100%" stop-color="var(--cyan)" stop-opacity="0"/>
            </linearGradient>
        </defs>
        ${area ? `<path d="${area}" fill="url(#sg)"/>` : ''}
        <path d="${path}" fill="none" stroke="var(--cyan)" stroke-width="1.5" stroke-linejoin="round" stroke-linecap="round"/>
        ${failDots}
        ${yLabels}
    </svg>`;
}

async function openDetail(targetUrl, displayName) {
    const panel  = document.getElementById('detailPanel');
    const overlay = document.getElementById('overlay');
    const body   = document.getElementById('dpBody');

    // Set header immediately
    document.getElementById('dpName').textContent = displayName;
    document.getElementById('dpUrl').textContent  = targetUrl;
    document.getElementById('dpStatusRow').innerHTML = '';
    body.innerHTML = '<div class="dp-loading">Loading…</div>';

    panel.classList.add('open');
    overlay.classList.add('open');
    document.body.style.overflow = 'hidden';

    // Fetch status detail + 24h history in parallel
    const param = targetParam(targetUrl);
    const [detail, history] = await Promise.all([
        fetchJSON(`${API}/status/${param}`),
        fetchJSON(`${API}/history/${param}?hours=24`),
    ]);

    // Status row in header
    const statusRow = document.getElementById('dpStatusRow');
    if (detail) {
        const s = detail.status || 'UNKNOWN';
        const color = s === 'HEALTHY' ? 'var(--green)' : 'var(--purple)';
        const lastChk = detail.last_check ? fmtDate(detail.last_check) : '';
        statusRow.innerHTML = `
            <span class="dot ${dotClass(s)}"></span>
            <span style="font-size:12px;font-weight:600;color:${color}">${s}</span>
            ${lastChk ? `<span style="font-size:11px;color:var(--muted)">· last checked ${lastChk}</span>` : ''}`;
    }

    // Build body
    const hist = Array.isArray(history) ? history : [];
    const stats = analyzeHistory(hist);
    let html = '';

    // ── Active incident alert ──
    if (detail?.active_incident) {
        const inc = detail.active_incident;
        html += `
        <div class="dp-incident">
            ⚠ Active incident since ${fmtDate(inc.started_at)} &nbsp;·&nbsp;
            Peak: ${inc.peak_status} (${Math.round(inc.peak_confidence*100)}% confidence)
        </div>`;
    }

    // ── 24h Stats ──
    html += `
    <div>
        <div class="dp-section-title">Last 24 Hours — ${stats.total} probes</div>
        <div class="dp-stats">
            <div class="dp-stat">
                <span class="dp-stat-label">Uptime</span>
                <span class="dp-stat-value ${parseFloat(stats.uptime) === 100 ? 'ok' : 'warn'}">${stats.uptime}%</span>
            </div>
            <div class="dp-stat">
                <span class="dp-stat-label">Avg</span>
                <span class="dp-stat-value cyan">${stats.avg ? stats.avg+'ms' : '—'}</span>
            </div>
            <div class="dp-stat">
                <span class="dp-stat-label">P95</span>
                <span class="dp-stat-value cyan">${stats.p95 ? stats.p95+'ms' : '—'}</span>
            </div>
            <div class="dp-stat">
                <span class="dp-stat-label">Failures</span>
                <span class="dp-stat-value ${stats.failed > 0 ? 'warn' : 'ok'}">${stats.failed}</span>
            </div>
        </div>
    </div>`;

    // ── Response time chart ──
    if (hist.length > 0) {
        const oldest = hist[0]?.time ? fmtDate(hist[0].time) : '';
        const newest = hist[hist.length-1]?.time ? fmtDate(hist[hist.length-1].time) : '';
        html += `
        <div>
            <div class="dp-section-title">Response Time (ms)</div>
            <div class="dp-chart">
                ${sparkline(hist)}
                <div class="dp-chart-labels">
                    <span>${oldest}</span>
                    <span style="color:var(--red);font-size:9px">● failed probe</span>
                    <span>${newest}</span>
                </div>
            </div>
        </div>`;
    }

    // ── Per-node breakdown ──
    if (detail?.node_breakdown?.length > 0) {
        html += `<div><div class="dp-section-title">Per-Node Probes</div><div class="dp-nodes">`;
        detail.node_breakdown.forEach(n => {
            const ok = n.tcp_success;
            const ms = n.total_ms > 0 ? `${n.total_ms}ms` : '—';
            const http = n.http_status > 0 ? `HTTP ${n.http_status}` : (n.error || 'TCP fail');
            html += `
            <div class="dp-node-row">
                <span class="dot ${ok ? 'healthy' : 'outage'}"></span>
                <span class="dp-node-name">${esc(n.node_id)}</span>
                <span class="dp-node-region">${esc(n.region || '')}</span>
                <span class="dp-node-ms">${ms}</span>
                <span class="dp-node-http" style="color:${n.http_status===200?'var(--green)':'var(--muted)'}">${esc(http)}</span>
            </div>`;
        });
        html += `</div></div>`;
    }

    // ── Recent probe log ──
    const recent = hist.slice(-20).reverse();
    if (recent.length > 0) {
        html += `<div><div class="dp-section-title">Recent Probes</div><div class="dp-probes">`;
        recent.forEach(p => {
            const ok = p.tcp_success;
            const color = ok ? 'var(--green)' : 'var(--red)';
            const label = ok ? (p.http_status ? `HTTP ${p.http_status}` : 'TCP OK') : (p.error_type || 'FAIL');
            const ms = p.total_ms > 0 ? `${p.total_ms}ms` : '—';
            html += `
            <div class="dp-probe-row">
                <span class="dp-probe-time">${fmtTime(p.time)}</span>
                <span class="dp-probe-node">${esc(p.node_id || '')}</span>
                <span class="dp-probe-ms">${ms}</span>
                <span class="dp-probe-status" style="color:${color}">${esc(label)}</span>
            </div>`;
        });
        html += `</div></div>`;
    }

    if (!html) {
        html = '<div class="dp-loading">No data available for this endpoint.</div>';
    }

    body.innerHTML = html;
}

function closeDetail() {
    document.getElementById('detailPanel').classList.remove('open');
    document.getElementById('overlay').classList.remove('open');
    document.body.style.overflow = '';
}

// Close on Escape key
document.addEventListener('keydown', e => { if (e.key === 'Escape') closeDetail(); });

// ── MAIN LOOP ──────────────────────────────────────────────

buildMap(null);
refresh();
setInterval(refresh, 5000);
