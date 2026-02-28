/* ============================================================
   NETWATCH.AZ — Dashboard JavaScript Logic (BFF Pattern)
   ============================================================ */
const API = '/api/v1';
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

function timeOnly(iso) {
    if (!iso) return '--:--';
    const d = new Date(iso);
    return d.toLocaleTimeString('en-GB') + ' UTC';
}

async function fetchJSON(url) {
    try {
        const r = await fetch(url);
        if (!r.ok) return null;
        return await r.json();
    } catch {
        return null;
    }
}

// ── RENDER AZERBAIJAN MAP ──
function buildMap(nodes) {
    const container = $('#azMapContainer');
    if (!container) return;

    const W = 1000;
    const H = 600;
    const AZ_X = 500;
    const AZ_Y = 300;

    let svg = `<svg viewBox="0 0 ${W} ${H}" style="width:100%; height:100%;">`;

    // Abstract Azerbaijan Silhouette
    svg += `<path class="az-silhouette" d="M350 250 Q 400 200 450 250 T 600 280 Q 700 250 650 350 T 500 400 Q 400 400 350 350 Z" />`;

    // Hub pulse
    svg += `<circle cx="${AZ_X}" cy="${AZ_Y}" r="6" class="map-pulse" />`;
    svg += `<circle cx="${AZ_X}" cy="${AZ_Y}" r="4" class="map-node" />`;
    svg += `<circle cx="${AZ_X}" cy="${AZ_Y}" r="24" class="map-node-glow" />`;
    svg += `<text x="${AZ_X}" y="${AZ_Y - 30}" text-anchor="middle" class="node-label">BAKU</text>`;

    const pos = {
        'node-eu-central': { x: 200, y: 150, lbl: 'FRANKFURT', arc: 'map-arc-cyan' },
        'node-us': { x: 100, y: 350, lbl: 'ASHBURN', arc: 'map-arc-purple' },
        'node-asia': { x: 850, y: 450, lbl: 'MUMBAI', arc: 'map-arc-cyan' }
    };

    if (nodes && nodes.length > 0) {
        nodes.forEach(n => {
            const p = pos[n.id];
            if (!p || n.id === 'node-az') return;

            const mx = (p.x + AZ_X) / 2;
            const my = Math.min(p.y, AZ_Y) - 100;
            const dStr = `M ${p.x} ${p.y} Q ${mx} ${my} ${AZ_X} ${AZ_Y}`;

            svg += `<path d="${dStr}" class="${p.arc}" />`;

            const glowCls = n.is_alive ? 'map-node-glow' : 'map-node-glow';
            svg += `<circle cx="${p.x}" cy="${p.y}" r="20" class="${glowCls}" style="fill:${n.is_alive ? 'var(--cyan-glow-outer)' : 'var(--purple-glow-outer)'}" />`;
            svg += `<circle cx="${p.x}" cy="${p.y}" r="4" style="fill:${n.is_alive ? 'var(--cyan-0)' : 'var(--purple-0)'}" />`;

            let latencyTxt = n.latency_ms ? `${n.latency_ms}ms` : '';
            svg += `<text x="${p.x}" y="${p.y + 24}" text-anchor="middle" class="node-label">${p.lbl} ${latencyTxt}</text>`;
        });
    }

    svg += `</svg>`;
    container.innerHTML = svg;
}

// ── INCIDENTS ──
function renderIncidents(incidents) {
    const grid = $('#incidentGrid');
    if (!grid) return;

    let html = '';

    for (let i = 0; i < 4; i++) {
        if (i < incidents.length) {
            const inc = incidents[i];
            const isPurple = inc.peak_status === 'MAJOR_OUTAGE' || inc.peak_status === 'PARTIAL_OUTAGE';
            const colorCls = isPurple ? 'purple glow-purple' : 'glow-cyan';
            const textCls = isPurple ? 'var(--purple-0)' : 'var(--cyan-0)';
            const iconSvg = isPurple
                ? '<svg viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>'
                : '<svg viewBox="0 0 24 24"><path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/></svg>';

            html += `
                <div class="incident-card glass-panel ${colorCls}">
                    <div class="ic-top">
                        <div class="ic-icon" style="background: ${textCls};">
                            ${iconSvg}
                        </div>
                        <div class="ic-titles">
                            <div class="ic-name">${esc(inc.target_url.replace(/^https?:\/\//, ''))}</div>
                            <div class="ic-tag">${inc.peak_status}</div>
                        </div>
                    </div>
                    <div class="ic-meta">
                        <div class="ic-meta-row">CONFIDENCE: <strong style="color:${textCls}">${Math.round(inc.peak_confidence * 100)}%</strong> • ${(inc.signals_fired || []).join(' / ')}</div>
                        <div class="ic-meta-row" style="margin-top:4px;">Started ${timeOnly(inc.started_at)} | ${inc.resolved_at ? 'Resolved' : 'Ongoing'}</div>
                    </div>
                    <div class="ic-btn" onclick="window.location.href='#'">Open Incident Briefing</div>
                </div>
            `;
        } else {
            html += `
                <div class="incident-card glass-panel" style="justify-content:center; align-items:center;">
                    <span class="t-label">Awaiting telemetry...</span>
                </div>
            `;
        }
    }

    grid.innerHTML = html;
}

// ── KPIS ──
function renderKPIs(kpis) {
    if (!kpis) return;

    const kpiMs = $('#kpiMonitoredSystems');
    if (kpiMs) kpiMs.textContent = kpis.monitored_systems || '0';

    const topConf = $('#topEndpoints');
    if (topConf) topConf.textContent = kpis.total_endpoints || '0';

    const lu = $('#topLastScan');
    if (lu && kpis.last_scan_time) lu.textContent = new Date(kpis.last_scan_time).toLocaleTimeString('en-GB', { hour12: false });

    const incs = $('#topActiveIncs');
    if (incs) incs.textContent = kpis.active_incidents_count || '0';

    const kb = $('#kpiBGP');
    if (kb) kb.textContent = kpis.bgp_anomalies_count || '0';
}

// ── SYSTEM HEALTH ACCORDION ──
function statusCls(s) {
    if (s === 'HEALTHY') return 'healthy';
    if (s === 'DEGRADED') return 'degraded';
    return 'outage';
}

function renderSystemHealth(systems) {
    const list = $('#systemList');
    if (!list) return;

    if (!systems || systems.length === 0) {
        list.innerHTML = `<div class="t-label">Analysis empty.</div>`;
        return;
    }

    let html = '';

    systems.forEach(sys => {
        // Flat style if it's a standalone
        if (sys.id.startsWith('standalone_')) {
            const ms = (sys.endpoints && sys.endpoints[0] && sys.endpoints[0].latency_ms) ? sys.endpoints[0].latency_ms : '--';
            html += `
                <div class="sys-group">
                    <div class="sys-parent" style="cursor: default;">
                        <div class="sys-dot ${statusCls(sys.status)}"></div>
                        <div class="sys-name">${esc(sys.display_name)}</div>
                        <div class="sys-ratio" style="font-size: 11px;">${ms}ms</div>
                        <div style="width:16px;"></div>
                    </div>
                </div>
            `;
            return;
        }

        // Accordion style
        html += `
            <div class="sys-group">
                <div class="sys-parent" onclick="this.parentElement.classList.toggle('expanded')">
                    <div class="sys-dot ${statusCls(sys.status)}"></div>
                    <div class="sys-name">${esc(sys.display_name)}</div>
                    <div class="sys-ratio">${sys.endpoints_healthy}/${sys.endpoints_total}</div>
                    <svg class="sys-chevron" viewBox="0 0 24 24"><path d="M7 10l5 5 5-5z"/></svg>
                </div>
                <div class="sys-children">
        `;

        if (sys.endpoints) {
            sys.endpoints.forEach(c => {
                let ms = c.latency_ms ? c.latency_ms : '--';
                html += `
                    <div class="sys-child">
                        <div class="sys-dot ${statusCls(c.status)}"></div>
                        <div class="sys-child-name">${esc(c.display_name)}</div>
                        <div class="sys-child-latency">${ms}ms</div>
                    </div>
                `;
            });
        }
        html += `</div></div>`;
    });

    list.innerHTML = html;
}

// ── MAIN LOOP ──
async function refresh() {
    const data = await fetchJSON(`${API}/dashboard/summary`);
    if (!data) return;

    renderKPIs(data.kpis);
    renderSystemHealth(data.systems);
    buildMap(data.map_nodes);
    renderIncidents(data.incidents);
}

buildMap(null); // Draw structure instantly
refresh();
setInterval(refresh, 5000); // Pulse rapidly for high-tech feeling
