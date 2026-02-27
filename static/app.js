/* ============================================================
   SENTINEL V2 — Dashboard Application Logic
   "Anti-Gravity" Design Language
   Uses i18n.js for translations, icons.js for SVG icons
   ============================================================ */
const API = '/api/v1';
const CATS_ORDER = { GOV: 1, GOV_SERVICE: 2, BANK: 3, FINTECH: 4, ISP: 5, MEDIA: 6, OTHER: 7, GLOBAL: 8, ANCHOR: 99 };
const CAT_ICON = { GOV: 'landmark', GOV_SERVICE: 'landmark', BANK: 'building', FINTECH: 'activity', ISP: 'globe', MEDIA: 'newspaper', OTHER: 'target', GLOBAL: 'globe', ANCHOR: 'anchor' };

const $ = s => document.querySelector(s);
const esc = s => String(s).replace(/[<>&"]/g, c => ({ '<': '&lt;', '>': '&gt;', '&': '&amp;', '"': '&quot;' }[c]));

function ago(iso) {
    const s = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
    if (s < 60) return s + t('time_s_ago');
    if (s < 3600) return Math.floor(s / 60) + t('time_m_ago');
    return Math.floor(s / 3600) + t('time_h_ago');
}

const statusCls = s => ({ MAJOR_OUTAGE: 'major', PARTIAL_OUTAGE: 'partial', DEGRADED: 'degraded' }[s] || 'healthy');
function statusLabel(s) {
    return ({ MAJOR_OUTAGE: t('status_major'), PARTIAL_OUTAGE: t('status_partial'), DEGRADED: t('status_degraded') }[s] || t('status_healthy'));
}
function statusIcon(s) {
    const cls = statusCls(s);
    if (cls === 'healthy') return ICONS.checkCircle();
    if (cls === 'degraded') return ICONS.minusCircle();
    return ICONS.xCircle();
}
const pillCls = s => 'pill-' + statusCls(s);
const barColor = s => `var(--${({ healthy: 'green', degraded: 'amber', partial: 'orange', major: 'red' })[statusCls(s)]})`;

// AZ/Global perspective — compares local vs external node results
function getAZGlobalLine(s) {
    if (!s.node_breakdown || s.node_breakdown.length === 0) {
        return `<span style="color:var(--text-dim)">${t('az_no_data')}</span>`;
    }
    const azNode = s.node_breakdown.find(n => n.node_id && (n.node_id.startsWith('node-az') || n.node_id.includes('az-baku')));
    const extNodes = s.node_breakdown.filter(n => !n.node_id?.startsWith('node-az') && !n.node_id?.includes('az-baku'));
    const azOk = azNode ? azNode.tcp_success : null;
    const extOkCount = extNodes.filter(n => n.tcp_success).length;
    const extTotal = extNodes.length;

    let azText, globalText, alert = '';
    if (azOk === null) {
        azText = `<span style="color:var(--text-dim)">${t('az_no_data')}</span>`;
    } else if (azOk) {
        azText = `<span style="color:var(--green)">${t('az_reachable')}</span>`;
    } else {
        azText = `<span style="color:var(--red)">${t('az_unreachable')}</span>`;
    }

    if (extTotal === 0) {
        globalText = '';
    } else if (extOkCount === extTotal) {
        globalText = ` · <span style="color:var(--green)">${t('global_ok')} ${extOkCount}/${extTotal}</span>`;
    } else {
        globalText = ` · <span style="color:var(--red)">${t('global_issues')} ${extOkCount}/${extTotal}</span>`;
    }

    if (azOk === false && extOkCount === extTotal && extTotal > 0) {
        alert = ` · <span style="color:var(--red);font-weight:600">${t('local_block')}</span>`;
    } else if (azOk === true && extOkCount < extTotal && extTotal > 0) {
        alert = ` · <span style="color:var(--orange);font-weight:600">${t('routing_issue')}</span>`;
    }

    return azText + globalText + alert;
}

async function fetchJSON(url) {
    try { const r = await fetch(url); return r.ok ? r.json() : null; } catch { return null; }
}

// ── Render: Hero ──
function renderHero(statuses) {
    const real = (statuses || []).filter(s => s.target.category !== 'ANCHOR');
    const h = real.filter(s => s.status === 'HEALTHY').length;
    const d = real.filter(s => s.status === 'DEGRADED').length;
    const o = real.filter(s => s.status === 'PARTIAL_OUTAGE' || s.status === 'MAJOR_OUTAGE').length;
    $('#statTargets').textContent = real.length;
    $('#statHealthy').textContent = h;
    $('#statDegraded').textContent = d;
    $('#statOutages').textContent = o;
    $('#statTargets').style.color = 'var(--cyan)';
    $('#statHealthy').style.color = 'var(--green)';
    $('#statDegraded').style.color = d > 0 ? 'var(--amber)' : 'var(--text-dim)';
    $('#statOutages').style.color = o > 0 ? 'var(--red)' : 'var(--text-dim)';
    $('#lastUpdate').textContent = new Date().toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', second: '2-digit', timeZone: 'UTC' }) + ' UTC';
}

// ── System Group Helper ──
function getSystemStatus(items) {
    if (items.some(s => s.status === 'MAJOR_OUTAGE')) return 'MAJOR_OUTAGE';
    if (items.some(s => s.status === 'PARTIAL_OUTAGE')) return 'PARTIAL_OUTAGE';
    if (items.some(s => s.status === 'DEGRADED')) return 'DEGRADED';
    return 'HEALTHY';
}

function renderTargetCard(s) {
    const cls = statusCls(s.status);
    const conf = (s.confidence * 100).toFixed(1);
    const tm = s.last_check ? ago(s.last_check) : '—';
    const azGlobal = getAZGlobalLine(s);
    const name = s.target.display_name || s.target.url;
    return `<div class="tcard s-${cls}" tabindex="0" role="button" aria-label="${esc(name)} — ${statusLabel(s.status)}" onclick="openDrill('${esc(s.target.url)}')" onkeydown="if(event.key==='Enter')openDrill('${esc(s.target.url)}')">
        <div class="tcard-top"><div><div class="tcard-name">${esc(name)}</div>
        <div class="tcard-url">${esc(s.target.url)}</div></div>
        <span class="tcard-pill ${pillCls(s.status)}"><span class="status-indicator si-${cls}">${statusLabel(s.status)}</span></span></div>
        <div class="tcard-bar" role="progressbar" aria-valuenow="${conf}" aria-valuemin="0" aria-valuemax="100"><div class="tcard-bar-fill" style="width:${conf}%;background:${barColor(s.status)}"></div></div>
        <div class="tcard-meta"><span>${azGlobal}</span><span>${t('confidence_label')} ${conf}%</span><span>${tm}</span></div>
    </div>`;
}

// ── Render: Targets (grouped by system → category) ──
function renderTargets(statuses) {
    const el = $('#targetsContainer');
    if (!statuses || statuses.length === 0) { el.innerHTML = `<div class="empty">${t('empty_no_targets')}</div>`; return; }

    // Separate into system-grouped and standalone
    const systems = {}; // parent_system → array of statuses
    const standalone = {}; // category → array of statuses (no parent_system)

    statuses.forEach(s => {
        if (s.target.category === 'ANCHOR') return;
        const sys = s.target.parent_system;
        if (sys) {
            (systems[sys] = systems[sys] || []).push(s);
        } else {
            (standalone[s.target.category] = standalone[s.target.category] || []).push(s);
        }
    });

    // Group categories: system groups first, then standalone
    const catGroups = {};
    Object.entries(systems).forEach(([sysName, items]) => {
        const mainCat = items[0].target.category;
        (catGroups[mainCat] = catGroups[mainCat] || { systems: [], standalone: [] }).systems.push({ name: sysName, items });
    });
    Object.entries(standalone).forEach(([cat, items]) => {
        (catGroups[cat] = catGroups[cat] || { systems: [], standalone: [] }).standalone.push(...items);
    });

    const sorted = Object.entries(catGroups).sort((a, b) => (CATS_ORDER[a[0]] || 99) - (CATS_ORDER[b[0]] || 99));

    let html = '';
    sorted.forEach(([cat, group]) => {
        const iconFn = CAT_ICON[cat] ? ICONS[CAT_ICON[cat]] : ICONS.target;
        const label = t('cat_' + cat) || cat;
        const totalCount = group.systems.reduce((sum, s) => sum + s.items.length, 0) + group.standalone.length;

        html += `<div class="category-group" role="region" aria-label="${esc(label)}">
            <div class="section-header">${iconFn()} ${esc(label)} <span class="count">${totalCount}</span></div>`;

        // Render system groups (parent_system cards)
        group.systems.sort((a, b) => {
            const aMax = Math.max(...a.items.map(i => i.target.criticality || 5));
            const bMax = Math.max(...b.items.map(i => i.target.criticality || 5));
            return bMax - aMax;
        }).forEach(sys => {
            const sysStatus = getSystemStatus(sys.items);
            const sysCls = statusCls(sysStatus);
            const upCount = sys.items.filter(s => s.status === 'HEALTHY').length;
            const total = sys.items.length;
            const allUp = upCount === total;
            const sysDisplayName = sys.items[0]?.target.display_name?.split(' ')[0] || sys.name;

            html += `<div class="system-group">
                <div class="system-group-header">
                    <div class="system-group-name">
                        <div class="sys-dot s-${sysCls}"></div>
                        ${esc(sysDisplayName)}
                    </div>
                    <div class="system-group-meta">
                        <span class="sys-ratio ${allUp ? 'all-up' : 'some-down'}">${upCount}/${total} ${t('status_healthy')}</span>
                    </div>
                </div>
                <div class="system-children">`;
            sys.items.forEach(s => { html += renderTargetCard(s); });
            html += '</div></div>';
        });

        // Render standalone targets
        if (group.standalone.length > 0) {
            html += '<div class="targets-grid">';
            group.standalone.forEach(s => { html += renderTargetCard(s); });
            html += '</div>';
        }

        html += '</div>';
    });
    el.innerHTML = html;
}

// ── Render: Nodes ──
function renderNodes(nodes) {
    const el = $('#nodesGrid');
    if (!nodes || nodes.length === 0) { el.innerHTML = `<div class="empty">${t('empty_nodes')}</div>`; return; }
    let html = '';
    nodes.forEach(n => {
        const alive = n.is_alive;
        const latPct = Math.min(n.avg_latency_ms / 500 * 100, 100);
        html += `<div class="ncard" role="listitem">
            <div class="ncard-top">
                <div class="ncard-dot ${alive ? 'alive' : 'dead'}" aria-label="${alive ? 'Online' : 'Offline'}"></div>
                <span class="ncard-name">${esc(n.node_id)}</span>
            </div>
            <div class="ncard-stat">${t('node_latency')} ${n.avg_latency_ms}ms · ${t('node_buffer')} ${n.buffer_depth} · v${n.version}</div>
            <div class="ncard-bar" role="progressbar" aria-label="Latency" aria-valuenow="${n.avg_latency_ms}" aria-valuemax="500"><div class="ncard-bar-fill" style="width:${latPct}%"></div></div>
        </div>`;
    });
    el.innerHTML = html;
    renderMapNodes(nodes);
}

// ── Render: BGP ──
function renderBGP(events) {
    const el = $('#bgpFeed');
    if (!events || events.length === 0) { el.innerHTML = `<div class="empty">${t('empty_no_bgp')}</div>`; return; }
    let html = '';
    events.slice(0, 15).forEach(e => {
        const isW = e.event_type === 'WITHDRAW';
        html += `<div class="bgp-item" role="listitem">
            <span class="bgp-type ${isW ? 'bgp-withdraw' : 'bgp-announce'}">${isW ? t('bgp_withdraw') : t('bgp_announce')}</span>
            <span class="bgp-detail"><strong>AS${e.asn}</strong> ${esc(e.provider)} · ${esc(e.prefix)}</span>
            <span class="bgp-time">${ago(e.time)}</span>
        </div>`;
    });
    el.innerHTML = html;
}

// ── Render: Incidents ──
function renderIncidents(incidents) {
    const el = $('#incidentsFeed');
    if (!incidents || incidents.length === 0) { el.innerHTML = `<div class="empty">${t('empty_no_incidents')}</div>`; return; }
    let html = '';
    incidents.slice(0, 10).forEach(inc => {
        const cls = statusCls(inc.peak_status);
        const color = `var(--${({ major: 'red', partial: 'orange', degraded: 'amber' })[cls] || 'green'})`;
        const resolved = inc.resolved_at ? `${t('incident_resolved')} ${ago(inc.resolved_at)}` : t('incident_ongoing');
        html += `<div class="incident-item" role="listitem">
            <div class="incident-dot" style="background:${color};box-shadow:0 0 6px ${color}" aria-hidden="true"></div>
            <div class="incident-body"><div class="incident-title">${esc(inc.target_url)}</div>
            <div class="incident-meta">${inc.peak_status} · ${t('drill_peak')} ${(inc.peak_confidence * 100).toFixed(0)}% · ${ago(inc.started_at)} · ${resolved}</div></div>
        </div>`;
    });
    el.innerHTML = html;
}

// ── Render: Methodology ──
function renderMethodology(m) {
    if (!m) return;
    const el = $('#methWeights');
    if (!el) return;
    el.innerHTML = `
        <div class="meth-w"><div class="meth-w-val" style="color:var(--cyan)">${((m.weights?.node || 0.5) * 100).toFixed(0)}%</div><div class="meth-w-label">Node Signal</div></div>
        <div class="meth-w"><div class="meth-w-val" style="color:var(--amber)">${((m.weights?.bgp || 0.3) * 100).toFixed(0)}%</div><div class="meth-w-label">BGP Signal</div></div>
        <div class="meth-w"><div class="meth-w-val" style="color:var(--orange)">${((m.weights?.social || 0.2) * 100).toFixed(0)}%</div><div class="meth-w-label">Social Signal</div></div>`;
}

// ── Dot-Matrix World Map ──
const LAND_HEX = [
    '0000000000000000000000000000000000000000000000000000',
    '0000000000000030000000000000000000000000000000000000',
    '00000000000c007c000780000000000000000000000000000000',
    '000000000e1e03fe003ff0000c0000000f800000000000000000',
    '000c00001e1f87ff007ff8001e00000e1fc00000000000000000',
    '001e00003e3fc7ff80fffc003f0000ff3fe00060000000000000',
    '003f80007e7fe7ffc1fffe003f8001ff7ff000f8000000000000',
    '007fc000fe7ff7ffc1ffff007fc003fffff800fc000000000000',
    '007fe001ff7ff7ffe3ffff807fe007fffffc01fe000000000000',
    '00fff003ff7fffffffffff80ffe00ffffffe03ff000000000000',
    '01fff807ffffffffffffffffffe01fffffff07ff800000000000',
    '03fffc0fffffffffffffffffffffffffff8fff800000000000',
    '03fffe0fffffffffffffffffffffffffffffff000000000000',
    '07ffff1fffffffffffffffffffffffffffffff000000000000',
    '0fffffffffffffffffffffffffffffffffff8000000000000',
    '0fffffffffffffffffffffffffffffffffff0000000000000',
    '1fffffffffffffffffffffffffffffffffff0000000000000',
    '1ffffffffffffffffffffffffffffffffffe0000000000000',
    '1ffffffffffffffffffffffffffffffffffe0000000000000',
    '0ffffffffffffffffffffffffffffffffffe0000000000000',
    '0ffffffffffffffffffffffffffffffffffc0000000000000',
    '07fffffffffffffffffffffffffffffffef80000000000000',
    '07ffffffffffffffffffffffffffffffe0f00000000000000',
    '03fffffffffffffffffffffffffffffe00600000000000000',
    '01ffffffffffffffffffc7fffffffff800000000000000000',
    '00ffffffffffffffffffc3fffffffe00000000000000000',
    '007fffffffffffffffffc1fffffffc00000000000000000',
    '003fffffffffffffffe001fffff80000000000000000000',
    '001fffffffffffffffc000fffff00000000000000000000',
    '000ffffffffffffff80000ffffe00000000000000000000',
    '0007fffffffffff000007fff800000000000000000000',
    '0003ffffffffe0000003ffe000000000000000000000',
    '0001ffffffff000000007f0000000000000000000000',
    '0000fffffffe000000003e0000000000000000000000',
    '00007fffff800000000000000000000c000000000000',
    '00001ffffc0000000000000000000007e00000000000',
    '000007ffe00000000000000000000003fc0000000000',
    '000001ff000000000000000000000001ff8000000000',
    '0000003c000000000000000000000000ffc000000000',
    '0000000000000000000000000000000007fe000000000',
    '00000000000000000000000000000000007f000000000',
    '000000000000000000000000000000000018000000000',
    '000000000000000000000000000000000018000000000',
    '00000000000000000000000000000001fffe000000000',
    '00000000000000000000000000000003ffff000000000',
];
const MAP_COLS = 100, MAP_ROWS = 45;
const DOT_SPACING = 8;

const NODE_GEO = {
    'node-us': { col: 22, row: 15, label: 'Ashburn' },
    'node-eu': { col: 49, row: 12, label: 'Frankfurt' },
    'node-eu-central': { col: 49, row: 12, label: 'Frankfurt' },
    'node-az': { col: 59, row: 14, label: 'Baku' },
    'node-asia': { col: 70, row: 20, label: 'Singapore' },
};
const AZ_POS = { col: 59, row: 14 };

function buildDotMap(nodes) {
    const container = document.getElementById('dotMap');
    if (!container) return;
    const w = MAP_COLS * DOT_SPACING;
    const h = MAP_ROWS * DOT_SPACING;
    let svg = `<svg viewBox="0 0 ${w} ${h}" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Dot matrix world map showing probe node locations">`;

    // Land dots
    for (let r = 0; r < MAP_ROWS; r++) {
        const hex = LAND_HEX[r] || '';
        for (let c = 0; c < MAP_COLS; c++) {
            const charIdx = Math.floor(c / 4);
            const bitIdx = 3 - (c % 4);
            const nibble = parseInt(hex.charAt(charIdx) || '0', 16);
            if ((nibble >> bitIdx) & 1) {
                svg += `<circle cx="${c * DOT_SPACING + 4}" cy="${r * DOT_SPACING + 4}" r="1.1" fill="rgba(255,255,255,0.06)"/>`;
            }
        }
    }

    // Azerbaijan highlight glow
    const azX = AZ_POS.col * DOT_SPACING + 4;
    const azY = AZ_POS.row * DOT_SPACING + 4;
    svg += `<circle cx="${azX}" cy="${azY}" r="16" fill="rgba(0,229,255,0.03)" stroke="none"/>`;
    svg += `<circle cx="${azX}" cy="${azY}" r="10" fill="rgba(0,229,255,0.05)" stroke="var(--cyan)" stroke-width="0.5" opacity="0.3"/>`;

    // Nodes + arcs
    if (nodes && nodes.length > 0) {
        nodes.forEach(n => {
            const geo = NODE_GEO[n.node_id];
            if (!geo) return;
            const nx = geo.col * DOT_SPACING + 4;
            const ny = geo.row * DOT_SPACING + 4;
            const color = n.is_alive ? 'var(--green)' : 'var(--red)';
            const glowColor = n.is_alive ? 'rgba(0,230,118,0.15)' : 'rgba(255,23,68,0.15)';

            // Arc to Azerbaijan
            if (n.node_id !== 'node-az') {
                const mx = (nx + azX) / 2;
                const my = Math.min(ny, azY) - 30;
                svg += `<path d="M${nx},${ny} Q${mx},${my} ${azX},${azY}" fill="none" stroke="${color}" stroke-width="0.7" stroke-dasharray="3 4" opacity="0.2" class="map-arc"/>`;
            }

            // Node glow + dot + ring
            svg += `<circle cx="${nx}" cy="${ny}" r="8" fill="${glowColor}"/>`;
            svg += `<circle cx="${nx}" cy="${ny}" r="2.5" fill="${color}"/>`;
            svg += `<circle cx="${nx}" cy="${ny}" r="6" fill="none" stroke="${color}" stroke-width="0.7" opacity="0.3" class="node-ring"/>`;

            // Label
            const labelY = ny < h / 2 ? ny - 14 : ny + 16;
            svg += `<text x="${nx}" y="${labelY}" text-anchor="middle" fill="var(--text-dim)" font-size="7.5" font-family="var(--font-sans)" font-weight="500">${geo.label}</text>`;
            svg += `<text x="${nx}" y="${labelY + 9}" text-anchor="middle" fill="var(--text-muted)" font-size="6" font-family="var(--font-mono)">${n.avg_latency_ms}ms</text>`;
        });
    }

    svg += '</svg>';
    container.innerHTML = svg;
}

function renderMapNodes(nodes) { buildDotMap(nodes); }

// ── Drill-Down ──
async function openDrill(targetUrl) {
    const overlay = $('#drillOverlay');
    const panel = $('#drillContent');
    const data = await fetchJSON(`${API}/status/${encodeURIComponent(targetUrl)}`);
    if (!data) { panel.innerHTML = `<div class="empty">${t('drill_failed')}</div>`; overlay.classList.add('open'); return; }

    let html = `<button class="overlay-close" onclick="closeDrill()" aria-label="Close">&times;</button>
        <div class="drill-title">${esc(data.target.display_name || data.target.url)}</div>
        <div class="drill-url">${esc(data.target.url)}</div>
        <div style="margin-bottom:1rem"><span class="tcard-pill ${pillCls(data.status)}"><span class="status-indicator si-${statusCls(data.status)}">${statusLabel(data.status)}</span></span>
        <span style="margin-left:1rem;font-family:var(--font-mono);font-size:0.78rem;color:var(--text-dim)">${t('confidence_label')}: ${(data.confidence * 100).toFixed(1)}%</span></div>`;

    if (data.node_breakdown && data.node_breakdown.length > 0) {
        html += `<div class="section-header" style="margin-top:1.5rem">${ICONS.server()} ${t('drill_per_node')}</div>`;
        const maxMs = Math.max(...data.node_breakdown.map(n => n.total_ms || 1), 1);
        data.node_breakdown.forEach(nb => {
            const w = Math.max((nb.total_ms / maxMs) * 100, 4);
            const ok = nb.tcp_success;
            const sIcon = ok ? ICONS.checkCircle() : ICONS.xCircle();
            html += `<div class="node-row">
                <div class="node-row-id">${sIcon} ${esc(nb.node_id.split('-').slice(0, 2).join('-'))}</div>
                <div class="timing-bars"><div class="timing-bar bar-tcp" style="width:${w}%;height:14px" title="Total: ${nb.total_ms}ms"></div></div>
                <div class="node-row-ms">${nb.total_ms}ms${nb.http_status ? ' · ' + nb.http_status : ''}</div></div>`;
        });
        html += `<div class="timing-legend"><span class="tl-tcp">${t('drill_timing_legend')}</span></div>`;
    }

    if (data.active_incident) {
        const inc = data.active_incident;
        html += `<div class="section-header" style="margin-top:1.5rem;color:var(--red)">${ICONS.alertTri()} ${t('drill_active_incident')}</div>
            <div style="font-size:0.82rem">${t('drill_started')} ${ago(inc.started_at)} · ${t('drill_peak')}: ${inc.peak_status} (${(inc.peak_confidence * 100).toFixed(0)}%)</div>
            <div style="font-size:0.72rem;color:var(--text-dim);margin-top:4px">${t('drill_signals')}: ${(inc.signals_fired || []).join(', ') || '—'}</div>`;
    }

    panel.innerHTML = html;
    overlay.classList.add('open');
    panel.querySelector('.overlay-close')?.focus();
}
function closeDrill() { $('#drillOverlay').classList.remove('open'); }

// ── Main Loop ──
async function refresh() {
    const [statuses, nodes, bgp, incidents, meth] = await Promise.all([
        fetchJSON(`${API}/status`),
        fetchJSON(`${API}/nodes`),
        fetchJSON(`${API}/bgp/events?hours=4`),
        fetchJSON(`${API}/incidents?limit=10`),
        fetchJSON(`${API}/methodology`),
    ]);
    renderHero(statuses);
    renderTargets(statuses);
    renderNodes(nodes);
    renderBGP(bgp);
    renderIncidents(incidents);
    renderMethodology(meth);
}

buildDotMap(null);
refresh();
setInterval(refresh, 30000);
document.addEventListener('keydown', e => { if (e.key === 'Escape') closeDrill(); });
