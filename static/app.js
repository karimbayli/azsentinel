/* ============================================================
   NETWATCH.AZ — Dashboard JavaScript Logic
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

// ── RENDER AZERBAIJAN MAP ──
// Map drawing with outer glow and neon arcs
function buildMap(nodes) {
    const container = $('#azMapContainer');
    if (!container) return;

    // Use full relative coords matching SVG viewBox
    // Let's set the viewBox to roughly 1000x600 for high resolution plotting
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

    // External Nodes
    const pos = {
        'node-eu-central': { x: 200, y: 150, lbl: 'FRANKFURT', arc: 'map-arc-cyan' },
        'node-us': { x: 100, y: 350, lbl: 'ASHBURN', arc: 'map-arc-purple' },
        'node-asia': { x: 850, y: 450, lbl: 'MUMBAI', arc: 'map-arc-cyan' }
    };

    if (nodes && nodes.length > 0) {
        nodes.forEach(n => {
            const p = pos[n.node_id];
            if (!p || n.node_id === 'node-az') return;

            // Draw arc
            const mx = (p.x + AZ_X) / 2;
            const my = Math.min(p.y, AZ_Y) - 100;
            const dStr = `M ${p.x} ${p.y} Q ${mx} ${my} ${AZ_X} ${AZ_Y}`;

            svg += `<path d="${dStr}" class="${p.arc}" />`;

            // Draw node
            const glowCls = n.is_alive ? 'map-node-glow' : 'map-node-glow';
            svg += `<circle cx="${p.x}" cy="${p.y}" r="20" class="${glowCls}" style="fill:${n.is_alive ? 'var(--cyan-glow-outer)' : 'var(--purple-glow-outer)'}" />`;
            svg += `<circle cx="${p.x}" cy="${p.y}" r="4" style="fill:${n.is_alive ? 'var(--cyan-0)' : 'var(--purple-0)'}" />`;

            // Text
            let latencyTxt = n.avg_latency_ms ? `${n.avg_latency_ms}ms` : '';
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

    // Keep exact 4 layout
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
            // Empties
            html += `
                <div class="incident-card glass-panel" style="justify-content:center; align-items:center;">
                    <span class="t-label">Awaiting telemetry...</span>
                </div>
            `;
        }
    }

    grid.innerHTML = html;
}

// ── KPIS & SECTORS ──
function renderKPIs(statuses, bgpEvents, incidents) {
    if (!statuses) return;

    const real = statuses.filter(s => s.target.category !== 'ANCHOR');

    // SLA Hero Calculation
    let totalConf = 0;
    real.forEach(s => totalConf += s.confidence);
    const avgConf = real.length > 0 ? (totalConf / real.length) : 0;

    const kpiSla = $('#kpiSLA');
    if (kpiSla) kpiSla.textContent = `${(avgConf * 100).toFixed(2)}%`;

    const topConf = $('#topConfidence');
    if (topConf) topConf.textContent = `${(avgConf * 100).toFixed(0)}%`;

    // Last Scan time
    const lu = $('#topLastScan');
    if (lu) lu.textContent = new Date().toLocaleTimeString('en-GB', { hour12: false });

    // Top Bar Incidents count
    const openCount = incidents.filter(i => !i.resolved_at).length;
    // Formatting as requested: 0|1|0 (we can use openCount for the middle)

    // BGP Anomalies count
    const bgpCount = bgpEvents.filter(e => e.event_type === 'WITHDRAW').length;
    const kb = $('#kpiBGP');
    if (kb) kb.textContent = bgpCount;

    // Sector Analysis
    const sectors = {};
    real.forEach(s => {
        const cat = s.target.category;
        if (!sectors[cat]) {
            sectors[cat] = { total: 0, healthy: 0 };
        }
        sectors[cat].total++;
        if (s.status === 'HEALTHY') sectors[cat].healthy++;
    });

    const secList = $('#sectorList');
    if (secList) {
        let sh = '';
        Object.entries(sectors).sort((a, b) => b[1].total - a[1].total).forEach(([cat, stats]) => {
            const pct = Math.round((stats.healthy / stats.total) * 100);
            let cCls = 'ok';
            let cStyle = 'color: var(--cyan-0);';
            if (pct < 100) { cCls = 'warn'; cStyle = 'color: var(--purple-0);'; }

            sh += `
                <div class="sector-item">
                    <span class="sector-name">${esc(cat)}</span>
                    <span class="sector-status" style="${cStyle}">${pct}%</span>
                </div>
            `;
        });
        secList.innerHTML = sh || '<div class="t-label">Analysis empty.</div>';
    }
}

// ── SYSTEM HEALTH ACCORDION ──
function statusCls(s) {
    if (s === 'HEALTHY') return 'healthy';
    if (s === 'DEGRADED') return 'degraded';
    return 'outage';
}

function renderSystemHealth(statuses) {
    const list = $('#systemList');
    if (!list) return;

    if (!statuses || statuses.length === 0) {
        list.innerHTML = `<div class="t-label">Analysis empty.</div>`;
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

        let healthyCount = 0;
        let worstStatus = 'HEALTHY';

        items.forEach(child => {
            if (child.status === 'HEALTHY') healthyCount++;
            if (child.status === 'DEGRADED' && worstStatus === 'HEALTHY') worstStatus = 'DEGRADED';
            if (child.status === 'PARTIAL_OUTAGE' || child.status === 'MAJOR_OUTAGE') worstStatus = 'MAJOR_OUTAGE';
        });

        // Use custom outage class if it's outage to keep the CSS mapping clean
        if (worstStatus === 'PARTIAL_OUTAGE' || worstStatus === 'MAJOR_OUTAGE') worstStatus = 'OUTAGE';

        html += `
            <div class="sys-group">
                <div class="sys-parent" onclick="this.parentElement.classList.toggle('expanded')">
                    <div class="sys-dot ${statusCls(worstStatus)}"></div>
                    <div class="sys-name">${esc(sysDisplayName)}</div>
                    <div class="sys-ratio">${healthyCount}/${items.length}</div>
                    <svg class="sys-chevron" viewBox="0 0 24 24"><path d="M7 10l5 5 5-5z"/></svg>
                </div>
                <div class="sys-children">
        `;

        items.forEach(c => {
            const childName = c.target.display_name || c.target.url.replace(/^https?:\/\//, '');
            let pStatus = statusCls(c.status);

            // Get latency
            let ms = '--';
            if (c.node_breakdown && c.node_breakdown.length > 0) {
                const azNode = c.node_breakdown.find(n => n.node_id.includes('az'));
                if (azNode && azNode.total_ms) ms = azNode.total_ms;
            }

            html += `
                <div class="sys-child">
                    <div class="sys-dot ${pStatus}"></div>
                    <div class="sys-child-name">${esc(childName)}</div>
                    <div class="sys-child-latency">${ms}ms</div>
                </div>
            `;
        });

        html += `</div></div>`;
    });

    // Option to render standalone if needed, but per request grouping is focus
    list.innerHTML = html;
}

// ── MAIN LOOP ──
async function refresh() {
    const [statuses, nodes, bgp, incidents] = await Promise.all([
        fetchJSON(`${API}/status`),
        fetchJSON(`${API}/nodes`),
        fetchJSON(`${API}/bgp/events?hours=4`),
        fetchJSON(`${API}/incidents?limit=10`)
    ]);

    renderKPIs(statuses || [], bgp || [], incidents || []);
    renderSystemHealth(statuses || []);
    buildMap(nodes);
    renderIncidents(incidents || []);
}

buildMap(null); // Draw structure instantly
refresh();
setInterval(refresh, 5000); // Pulse rapidly for high-tech feeling
