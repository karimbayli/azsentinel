/* ============================================================
   ANTI-GRAVITY UI / UX — Dashboard Application Logic
   ============================================================ */
const API = '/api/v1';

// We want to group by system, but display flat dense rows in the right sidebar.
const $ = s => document.querySelector(s);
const esc = s => String(s).replace(/[<>&"]/g, c => ({ '<': '&lt;', '>': '&gt;', '&': '&amp;', '"': '&quot;' }[c]));

// Utility: time ago
function ago(iso) {
    if (!iso) return '—';
    const s = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
    if (s < 60) return s + 's ago';
    if (s < 3600) return Math.floor(s / 60) + 'm ago';
    return Math.floor(s / 3600) + 'h ago';
}

function statusCls(s) {
    if (s === 'HEALTHY') return 'healthy';
    if (s === 'DEGRADED') return 'degraded';
    return 'outage'; // PARTIAL_OUTAGE or MAJOR_OUTAGE
}

// Sparkline Generator (Simulated latency history for visual density)
// Generates a smooth SVG path. baseMs determines the y-axis center.
function generateSparkline(latencyMs, status, width = 60, height = 24) {
    const pts = 20;
    const path = [];
    const cls = statusCls(status);

    // baseline volatility
    let vol = 3;
    let base = latencyMs || 80;
    if (cls === 'degraded') { vol = 15; base += 50; }
    if (cls === 'outage') { vol = 5; base = height * 2; } // off chart

    // Generate random points moving right
    for (let i = 0; i < pts; i++) {
        const x = (i / (pts - 1)) * width;
        // Generate y bouncing around base, scaled to height
        // normal latency maps to ~50% height
        let val = base + (Math.random() * vol * 2 - vol);
        let y = height - (val / 200 * height);
        // clamp
        if (y < 2) y = 2;
        if (y > height - 2) y = height - 2;

        path.push(`${i === 0 ? 'M' : 'L'} ${x.toFixed(1)} ${y.toFixed(1)}`);
    }

    const d = path.join(' ');
    // area path needs to close at bottom
    const areaD = `${d} L ${width} ${height} L 0 ${height} Z`;

    return `
        <svg class="spark-svg" viewBox="0 0 ${width} ${height}">
            <path class="spark-area" d="${areaD}" fill="var(--${cls}-glow)"></path>
            <path class="spark-line ${cls}" d="${d}"></path>
        </svg>
    `;
}

// Mini SLA Ring
function renderMiniSLA(conf, status) {
    const pct = Math.round(conf * 100);
    const cls = statusCls(status);
    const c = 283; // 2 * pi * 45
    // dashoffset = 283 - (pct / 100) * 283
    const offset = c - (pct / 100 * c);
    return `
        <div class="t-sla">
            <svg viewBox="0 0 100 100" style="transform:rotate(-90deg);width:100%;height:100%">
                <circle cx="50" cy="50" r="45" class="ring-mini-bg"></circle>
                <circle cx="50" cy="50" r="45" class="ring-mini-fill ${cls}" stroke-dashoffset="${offset}"></circle>
            </svg>
            <div class="t-sla-val">${pct}</div>
        </div>
    `;
}

// ── RENDER AZERBAIJAN MAP ──
// High-tech abstract node map showing fiber paths to AZ
const MAP_W = 800;
const MAP_H = 400;

function buildMap(nodes) {
    const container = $('#azMapContainer');
    if (!container) return;

    // AZ Center
    const AZ_X = 550;
    const AZ_Y = 200;

    let svg = `<svg class="map-svg" viewBox="0 0 ${MAP_W} ${MAP_H}" preserveAspectRatio="xMidYMid slice">`;

    // Abstract dot grid background pattern
    svg += `<defs><pattern id="dotGrid" width="20" height="20" patternUnits="userSpaceOnUse">
            <circle cx="2" cy="2" r="1" fill="rgba(255,255,255,0.05)" />
            </pattern></defs>
            <rect width="${MAP_W}" height="${MAP_H}" fill="url(#dotGrid)" />`;

    // Map the node APIs geographically-ish
    const pos = {
        'node-us': { x: 100, y: 150, lbl: 'ASHBURN (US)' },
        'node-eu-central': { x: 250, y: 100, lbl: 'FRANKFURT (EU)' },
        'node-eu': { x: 250, y: 100, lbl: 'FRANKFURT (EU)' },
        'node-asia': { x: 700, y: 300, lbl: 'SINGAPORE (SG)' },
        'node-az': { x: AZ_X, y: AZ_Y, lbl: 'BAKU (AZ)' }
    };

    // AZ Hub glow
    svg += `<circle cx="${AZ_X}" cy="${AZ_Y}" r="40" class="map-node-glow" opacity="0.5" />`;
    svg += `<circle cx="${AZ_X}" cy="${AZ_Y}" r="6" fill="var(--cyan)" />`;
    svg += `<circle cx="${AZ_X}" cy="${AZ_Y}" r="14" fill="none" stroke="var(--cyan)" stroke-width="1" opacity="0.4" stroke-dasharray="2 4" />`;
    svg += `<text x="${AZ_X}" y="${AZ_Y - 20}" text-anchor="middle" class="map-node-value text-cyan" style="font-size:12px;">BAKU HUB</text>`;

    if (nodes && nodes.length > 0) {
        nodes.forEach(n => {
            const p = pos[n.node_id];
            if (!p) return;
            const isLocal = n.node_id === 'node-az';
            const color = n.is_alive ? 'var(--cyan)' : 'var(--crimson)';
            const glow = n.is_alive ? 'var(--cyan-glow)' : 'var(--crimson-glow)';

            if (!isLocal) {
                // Fiber path arc to AZ
                const mx = (p.x + AZ_X) / 2;
                const my = Math.min(p.y, AZ_Y) - 50;
                const pathD = `M ${p.x} ${p.y} Q ${mx} ${my} ${AZ_X} ${AZ_Y}`;

                // base line
                svg += `<path d="${pathD}" class="fiber-glow" stroke="${color}" />`;
                // animated dash
                svg += `<path d="${pathD}" class="fiber-path" stroke="${color}" />`;
                // pulse
                if (n.is_alive) {
                    svg += `<path d="${pathD}" class="fiber-pulse" stroke="var(--cyan)" />`;
                }
            }

            // Node Dot
            svg += `<circle cx="${p.x}" cy="${p.y}" r="4" fill="${color}" />`;
            svg += `<circle cx="${p.x}" cy="${p.y}" r="12" fill="none" stroke="${color}" stroke-width="1" opacity="0.3" />`;

            // Labels
            svg += `<text x="${p.x}" y="${p.y + 16}" text-anchor="middle" class="map-node-label">${p.lbl}</text>`;
            if (n.avg_latency_ms) {
                svg += `<text x="${p.x}" y="${p.y + 26}" text-anchor="middle" class="map-node-value">${n.avg_latency_ms}ms</text>`;
            }
        });
    }

    svg += `</svg>`;
    container.innerHTML = svg;
}

// ── RENDER TELEMETRY RIGHT SIDEBAR ──
function renderTelemetry(statuses) {
    const list = $('#targetList');
    if (!statuses || statuses.length === 0) {
        list.innerHTML = `<div class="empty-state" style="padding: 24px; text-align: center;">No endpoints found.</div>`;
        return;
    }

    const sysGroups = {};
    const standalone = [];

    statuses.forEach(s => {
        if (s.target.category === 'ANCHOR') return;
        const sys = s.target.parent_system;
        if (sys) {
            if (!sysGroups[sys]) sysGroups[sys] = [];
            sysGroups[sys].push(s);
        } else {
            standalone.push(s);
        }
    });

    let html = '';

    // Render grouped systems
    Object.keys(sysGroups).sort().forEach(sys => {
        const items = sysGroups[sys];
        const sysDisplayName = items[0].target.display_name?.split(' ')[0] || sys;

        html += `<div class="sys-header">
            <span>// ${sysDisplayName}</span>
            <div class="sys-line"></div>
        </div>`;

        items.forEach(s => { html += renderTargetRow(s); });
    });

    // Render standalone
    if (standalone.length > 0) {
        html += `<div class="sys-header"><span>// STANDALONE</span><div class="sys-line"></div></div>`;
        standalone.forEach(s => { html += renderTargetRow(s); });
    }

    list.innerHTML = html;
}

function renderTargetRow(s) {
    const name = s.target.display_name || s.target.url;
    const cls = statusCls(s.status);

    // Try to find a latency ms from node_breakdown if possible, else random baseline
    let ms = 45;
    if (s.node_breakdown && s.node_breakdown.length > 0) {
        const azNode = s.node_breakdown.find(n => n.node_id.includes('az'));
        if (azNode && azNode.total_ms) ms = azNode.total_ms;
    }

    return `
        <div class="t-row" onclick="openDrill('${esc(s.target.url)}')">
            <div class="t-dot ${cls}"></div>
            <div class="t-info">
                <div class="t-name">${esc(name)}</div>
                <div class="t-url">${esc(s.target.url.replace(/^https?:\/\//, ''))}</div>
            </div>
            <div class="t-spark">
                ${generateSparkline(ms, s.status, 60, 24)}
            </div>
            ${renderMiniSLA(s.confidence, s.status)}
        </div>
    `;
}

// ── GLOBAL STATS & HERO ──
function renderStats(statuses) {
    const real = (statuses || []).filter(s => s.target.category !== 'ANCHOR');
    const h = real.filter(s => s.status === 'HEALTHY').length;
    const d = real.filter(s => s.status === 'DEGRADED').length;
    const o = real.filter(s => s.status === 'PARTIAL_OUTAGE' || s.status === 'MAJOR_OUTAGE').length;

    $('#statMonitored').textContent = real.length;
    $('#statHealthy').textContent = h;
    $('#statDegraded').textContent = d;
    $('#statOutage').textContent = o;

    // Overall SLA = average confidence of all targets
    let totalConf = 0;
    real.forEach(s => totalConf += s.confidence);
    const avgConf = real.length > 0 ? (totalConf / real.length) : 0;

    const slider = $('#globalSlaRing');
    if (slider) {
        const c = 283;
        const off = c - (avgConf * c);
        slider.style.strokeDashoffset = off;
        // color change if dropping
        if (avgConf < 0.9) slider.style.stroke = 'var(--amber)';
        if (avgConf < 0.5) slider.style.stroke = 'var(--crimson)';
    }

    const val = $('#globalSlaValue');
    if (val) val.innerHTML = `${(avgConf * 100).toFixed(2)}<span class="pct">%</span>`;

    const lu = $('#lastUpdate');
    if (lu) lu.textContent = new Date().toLocaleTimeString('en-GB') + ' UTC';
}

// ── INCIDENTS ──
function renderIncidents(incidents) {
    const list = $('#incidentList');
    if (!incidents || incidents.length === 0) {
        list.innerHTML = `<div class="empty-state">No active incidents detected.</div>`;
        return;
    }
    let html = '';
    incidents.slice(0, 15).forEach(inc => {
        html += `
            <div class="incident-item">
                <div class="inc-target">${esc(inc.target_url.replace(/^https?:\/\//, ''))}</div>
                <div class="inc-meta">
                    <span class="text-crimson">${inc.peak_status}</span>
                    <span>${ago(inc.started_at)}</span>
                </div>
            </div>
        `;
    });
    list.innerHTML = html;
}

// ── BGP HEATMAP ──
function renderBGP(events) {
    const feed = $('#bgpFeed');
    if (!events || events.length === 0) {
        feed.innerHTML = `<div class="empty-state">Monitoring BGP announcements...</div>`;
        return;
    }
    let html = '';
    events.slice(0, 30).forEach(e => {
        const isW = e.event_type === 'WITHDRAW';
        html += `
            <div class="bgp-item ${isW ? 'withdraw' : 'announce'}">
                <div class="bgp-item-type">${isW ? 'W/DRAW' : 'ANNOUN'}</div>
                <div>
                    <div class="bgp-item-as">AS${e.asn}</div>
                    <div class="bgp-item-prefix">${esc(e.prefix)}</div>
                </div>
                <div class="bgp-item-time">${ago(e.time)}</div>
            </div>
        `;
    });
    feed.innerHTML = html;
}

// ── DRILLDOWN MODAL ──
async function openDrill(targetUrl) {
    const overlay = $('#drillModal');
    const panel = $('#drillContent');
    const data = await fetchJSON(`${API}/status/${encodeURIComponent(targetUrl)}`);

    if (!data) {
        panel.innerHTML = `<div class="drill-header"><div class="drill-title">Data Unavailable</div><button class="drill-close" onclick="closeDrill()">&times;</button></div>`;
        overlay.classList.add('open');
        return;
    }

    const name = data.target.display_name || data.target.url;
    let html = `
        <div class="drill-header">
            <button class="drill-close" aria-label="Close" onclick="closeDrill()">&times;</button>
            <div class="drill-title">${esc(name)}</div>
            <div class="drill-url">${esc(data.target.url)}</div>
        </div>
        <div class="drill-body">
            <div class="drill-row">
                <div class="drill-lbl">SYSTEM STATUS</div>
                <div class="drill-val ${statusCls(data.status) === 'healthy' ? 'text-emerald' : 'text-crimson'}">${data.status}</div>
            </div>
            <div class="drill-row">
                <div class="drill-lbl">SLA CONFIDENCE</div>
                <div class="drill-val">${(data.confidence * 100).toFixed(2)}%</div>
            </div>
    `;

    if (data.active_incident) {
        const inc = data.active_incident;
        html += `
            <div class="drill-row">
                <div class="drill-lbl text-crimson">ACTIVE INCIDENT</div>
                <div class="drill-val text-crimson">Started ${ago(inc.started_at)}</div>
            </div>
            <div class="drill-row">
                <div class="drill-lbl text-crimson">INCIDENT SIGNALS</div>
                <div class="drill-val">${(inc.signals_fired || []).join(', ')}</div>
            </div>
        `;
    }

    if (data.node_breakdown && data.node_breakdown.length > 0) {
        html += `<div style="margin-top: 10px; font-size: 10px; font-weight: 700; color: var(--text-dim); letter-spacing: 1px;">NODE TELEMETRY</div>`;
        data.node_breakdown.forEach(nb => {
            html += `
                <div class="drill-row">
                    <div class="drill-lbl">${esc(nb.node_id)}</div>
                    <div class="drill-val">${nb.total_ms}ms ${nb.tcp_success ? '✅' : '❌'}</div>
                </div>
            `;
        });
    }

    html += `</div>`;
    panel.innerHTML = html;
    overlay.classList.add('open');
}

function closeDrill() {
    $('#drillModal').classList.remove('open');
}

// ── UTILS ──
async function fetchJSON(url) {
    try { const r = await fetch(url); return r.ok ? r.json() : null; } catch { return null; }
}

async function refresh() {
    const [statuses, nodes, bgp, incidents] = await Promise.all([
        fetchJSON(`${API}/status`),
        fetchJSON(`${API}/nodes`),
        fetchJSON(`${API}/bgp/events?hours=4`),
        fetchJSON(`${API}/incidents?limit=20`)
    ]);

    renderStats(statuses);
    renderTelemetry(statuses);
    buildMap(nodes);
    renderBGP(bgp);
    renderIncidents(incidents);
}

// Initial draw + loop
buildMap(null);
refresh();
setInterval(refresh, 10000); // 10s refresh for C-Level feel

// Dismiss modal on escape
document.addEventListener('keydown', e => { if (e.key === 'Escape') closeDrill(); });
