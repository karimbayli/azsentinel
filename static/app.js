/* ============================================================
   NETWATCH.AZ — Dashboard JavaScript Logic
   Pure API Consumer (No Hardcoded HTML Bloat)
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
            const p = pos[n.node_id];
            if (!p || n.node_id === 'node-az') return;

            const mx = (p.x + AZ_X) / 2;
            const my = Math.min(p.y, AZ_Y) - 100;
            const dStr = `M ${p.x} ${p.y} Q ${mx} ${my} ${AZ_X} ${AZ_Y}`;

            svg += `<path d="${dStr}" class="${p.arc}" />`;

            const glowCls = n.is_alive ? 'map-node-glow' : 'map-node-glow';
            svg += `<circle cx="${p.x}" cy="${p.y}" r="20" class="${glowCls}" style="fill:${n.is_alive ? 'var(--cyan-glow-outer)' : 'var(--purple-glow-outer)'}" />`;
            svg += `<circle cx="${p.x}" cy="${p.y}" r="4" style="fill:${n.is_alive ? 'var(--cyan-0)' : 'var(--purple-0)'}" />`;

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
function renderKPIs(statuses, bgpEvents, incidents) {
    if (!statuses) return;

    // Filter out anchors
    const real = statuses.filter(s => s.target.category !== 'ANCHOR');

    // SLA Hero Calculation
    let totalConf = 0;
    real.forEach(s => totalConf += s.confidence);
    const avgConf = real.length > 0 ? (totalConf / real.length) : 0;

    const kpiConf = $('#kpiConfidence');
    if (kpiConf) kpiConf.textContent = `${(avgConf * 100).toFixed(2)}%`;

    // Global Top Bar
    let globalStatus = "STABLE";
    let globalColor = "var(--cyan-0)";
    const openIncs = incidents.filter(i => !i.resolved_at).length;

    if (openIncs > 0) {
        globalStatus = "DEGRADED";
        globalColor = "var(--purple-0)";
    }

    const tStat = $('#topStatus');
    if (tStat) {
        tStat.textContent = globalStatus;
        tStat.style.color = globalColor;
    }

    const te = $('#topEndpoints');
    if (te) te.textContent = real.length;

    const ti = $('#topIncidents');
    if (ti) ti.textContent = openIncs;

    const ts = $('#topLastScan');
    if (ts) ts.textContent = new Date().toLocaleTimeString('en-GB', { hour12: false });

    // Side Stack KPIs
    const bgpCount = bgpEvents.filter(e => e.event_type === 'WITHDRAW').length;
    const kb = $('#kpiBGP');
    if (kb) {
        kb.textContent = bgpCount;
        kb.style.color = bgpCount > 0 ? 'var(--purple-0)' : 'var(--cyan-0)';
    }

    const ke = $('#kpiEndpoints');
    if (ke) ke.textContent = real.length;
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
        list.innerHTML = `<div class="t-label" style="text-align:center; margin-top: 20px;">Analysis empty.</div>`;
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

    // Update Unique Systems KPI
    const kpiSys = $('#kpiSystems');
    if (kpiSys) kpiSys.textContent = Object.keys(sysGroups).length + standalone.length;

    let html = '';

    // Known systems Dictionary to beautifully format raw db slugs
    const knownSystems = {
        'mygov': 'myGov',
        'egov': 'E-Government',
        'asan': 'ASAN',
        'sima': 'SIMA',
        'esosial': 'E-Sosial',
        'abb': 'ABB Bank',
        'kapitalbank': 'Kapital Bank',
        'm10': 'm10 / PashaPay',
        'leo': 'Leobank',
        'azercell': 'Azercell',
        'bakcell': 'Bakcell',
        'nar': 'Nar Mobile',
        'agtelecom': 'Aztelekom',
        'baktelecom': 'Baktelecom',
        'katv': 'KATV1',
        'citynet': 'CityNet'
    };

    // Render grouped systems
    Object.keys(sysGroups).sort().forEach(sys => {
        const items = sysGroups[sys];
        let sysDisplayName = sys.toUpperCase();

        if (knownSystems[sys]) {
            sysDisplayName = knownSystems[sys];
        } else if (items[0].target.display_name_en) {
            sysDisplayName = items[0].target.display_name_en.split(' ')[0];
        } else if (items[0].target.display_name) {
            sysDisplayName = items[0].target.display_name.split(' ')[0];
        }

        let healthyCount = 0;
        let worstStatus = 'HEALTHY';

        items.forEach(child => {
            if (child.status === 'HEALTHY') healthyCount++;
            if (child.status === 'DEGRADED' && worstStatus === 'HEALTHY') worstStatus = 'DEGRADED';
            if (child.status === 'PARTIAL_OUTAGE' || child.status === 'MAJOR_OUTAGE') worstStatus = 'MAJOR_OUTAGE';
        });

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
            const childName = c.target.display_name_en || c.target.display_name || c.target.url.replace(/^https?:\/\//, '');
            let conf = (c.confidence * 100).toFixed(0) + '%';
            if (c.status !== 'HEALTHY') {
                conf = `<span style="color:var(--purple-0)">${conf}</span>`;
            }

            html += `
                <div class="sys-child">
                    <div class="sys-dot ${pStatus}"></div>
                    <div class="sys-child-name">${esc(childName)}</div>
                    <div class="sys-child-latency">${conf}</div>
                </div>
            `;
        });

        html += `</div></div>`;
    });

    // Render standalone targets as flat rows
    if (standalone.length > 0) {
        standalone.forEach(c => {
            const childName = c.target.display_name_en || c.target.display_name || c.target.url.replace(/^https?:\/\//, '');
            let conf = (c.confidence * 100).toFixed(0) + '%';
            if (c.status !== 'HEALTHY') {
                conf = `<span style="color:var(--purple-0)">${conf}</span>`;
            }

            html += `
                <div class="sys-group">
                    <div class="sys-parent" style="cursor: default;">
                        <div class="sys-dot ${pStatus}"></div>
                        <div class="sys-name">${esc(childName)}</div>
                        <div class="sys-ratio" style="font-size: 11px;">${conf}</div>
                        <div style="width:16px;"></div> <!-- Placeholder for alignment -->
                    </div>
                </div>
            `;
        });
    }

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

buildMap(null); // Draw map instantly
refresh();
setInterval(refresh, 5000); // 5s pulse
