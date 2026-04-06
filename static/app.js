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
        <div class="sys-endpoint">
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
        <div class="sys-standalone">
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

buildMap(null);
refresh();
setInterval(refresh, 5000);
