/* ============================================================
   SENTINEL V2 — Internationalization (i18n)
   Auto-detects AZ IP → Azerbaijani, else English.
   Technical terms without AZ equivalents stay in English.
   ============================================================ */

const LANG = {
    en: {
        hero_subtitle: 'Azerbaijan Internet Infrastructure Monitor — Multi-node external reachability',
        stat_monitored: 'Monitored',
        stat_healthy: 'Healthy',
        stat_degraded: 'Degraded',
        stat_outages: 'Outages',
        section_map: 'Global Vantage Points',
        section_nodes: 'Node Health',
        section_incidents: 'Recent Incidents',
        section_bgp: 'BGP Events',
        section_confidence: 'Confidence Model',
        empty_nodes: 'Waiting for node data…',
        empty_targets: 'Loading targets…',
        empty_incidents: 'Loading…',
        empty_bgp: 'Loading…',
        empty_no_targets: 'No targets configured',
        empty_no_incidents: 'No incidents recorded',
        empty_no_bgp: 'No BGP events in window',
        confidence_desc: "Status is determined by combining three independent signals. Each signal's weight reflects its reliability and independence.",
        footer_note: 'UpDown.az monitors <strong>external reachability only</strong>.',
        footer_methodology: 'Read full methodology',
        footer_export: 'Export CSV',
        footer_refresh: 'Data refreshes every 30 seconds. All times UTC.',
        cat_GOV: 'Government',
        cat_GOV_SERVICE: 'Citizen Digital Services',
        cat_BANK: 'Banking & Finance',
        cat_FINTECH: 'Fintech & Payments',
        cat_ISP: 'Internet Providers',
        cat_MEDIA: 'Media & News',
        cat_OTHER: 'Popular Services',
        cat_GLOBAL: 'Global Platforms',
        cat_ANCHOR: 'Baseline Anchors',
        az_reachable: 'AZ: Reachable',
        az_unreachable: 'AZ: Unreachable',
        az_no_data: 'AZ: No data',
        global_ok: 'Global: OK',
        global_issues: 'Global: Issues',
        local_block: 'Local block detected',
        routing_issue: 'External routing issue',
        confidence_label: 'Confidence',
        criticality_label: 'Criticality',
        status_healthy: 'HEALTHY',
        status_degraded: 'DEGRADED',
        status_partial: 'PARTIAL OUTAGE',
        status_major: 'MAJOR OUTAGE',
        drill_per_node: 'Per-Node Probes',
        drill_active_incident: 'Active Incident',
        drill_started: 'Started',
        drill_peak: 'Peak',
        drill_signals: 'Signals',
        drill_timing_legend: 'TCP+TLS+HTTP',
        drill_failed: 'Failed to load details',
        incident_ongoing: 'Ongoing',
        incident_resolved: 'Resolved',
        node_latency: 'Latency',
        node_buffer: 'Buffer',
        bgp_announce: 'ANNOUNCE',
        bgp_withdraw: 'WITHDRAW',
        time_s_ago: 's ago',
        time_m_ago: 'm ago',
        time_h_ago: 'h ago',
    },
    az: {
        hero_subtitle: 'Azərbaycanın internet infrastrukturunun monitorinqi — çoxnöqtəli xarici əlçatanlıq',
        stat_monitored: 'Monitorinqdə',
        stat_healthy: 'Sağlam',
        stat_degraded: 'Zəifləyib',
        stat_outages: 'Kəsilmə',
        section_map: 'Qlobal müşahidə nöqtələri',
        section_nodes: 'Node vəziyyəti',
        section_incidents: 'Son insidentlər',
        section_bgp: 'BGP hadisələri',
        section_confidence: 'Etibarlılıq modeli',
        empty_nodes: 'Node məlumatları gözlənilir…',
        empty_targets: 'Hədəflər yüklənir…',
        empty_incidents: 'Yüklənir…',
        empty_bgp: 'Yüklənir…',
        empty_no_targets: 'Hədəf konfiqurasiya olunmayıb',
        empty_no_incidents: 'İnsident qeydə alınmayıb',
        empty_no_bgp: 'Bu zaman aralığında BGP hadisəsi yoxdur',
        confidence_desc: 'Status üç müstəqil siqnalın birləşdirilməsi ilə müəyyən edilir. Hər siqnalın çəkisi onun etibarlılığını və müstəqilliyini əks etdirir.',
        footer_note: 'UpDown.az yalnız <strong>xarici əlçatanlığı</strong> monitorinq edir.',
        footer_methodology: 'Metodologiyanı oxu',
        footer_export: 'CSV ixrac et',
        footer_refresh: 'Məlumatlar hər 30 saniyədə yenilənir. Bütün vaxtlar UTC formatındadır.',
        cat_GOV: 'Dövlət',
        cat_GOV_SERVICE: 'Rəqəmsal dövlət xidmətləri',
        cat_BANK: 'Bank və Maliyyə',
        cat_FINTECH: 'Fintech və Ödənişlər',
        cat_ISP: 'İnternet provayderləri',
        cat_MEDIA: 'Media',
        cat_OTHER: 'Populyar xidmətlər',
        cat_GLOBAL: 'Qlobal platformalar',
        cat_ANCHOR: 'Baza ankerləri',
        az_reachable: 'AZ: Əlçatandır',
        az_unreachable: 'AZ: Əlçatmaz',
        az_no_data: 'AZ: Məlumat yoxdur',
        global_ok: 'Qlobal: OK',
        global_issues: 'Qlobal: Problemlər',
        local_block: 'Yerli bloklanma aşkarlandı',
        routing_issue: 'Xarici marşrutizasiya problemi',
        confidence_label: 'Etibarlılıq',
        criticality_label: 'Kritiklik',
        status_healthy: 'SAĞLAM',
        status_degraded: 'ZƏİFLƏYİB',
        status_partial: 'QİSMƏN KƏSİLMƏ',
        status_major: 'TAM KƏSİLMƏ',
        drill_per_node: 'Node üzrə nəticələr',
        drill_active_incident: 'Aktiv insident',
        drill_started: 'Başlayıb',
        drill_peak: 'Pik',
        drill_signals: 'Siqnallar',
        drill_timing_legend: 'TCP+TLS+HTTP',
        drill_failed: 'Təfərrüatları yükləmək mümkün olmadı',
        incident_ongoing: 'Davam edir',
        incident_resolved: 'Həll olunub',
        node_latency: 'Gecikmə',
        node_buffer: 'Bufer',
        bgp_announce: 'ANNOUNCE',
        bgp_withdraw: 'WITHDRAW',
        time_s_ago: 'san əvvəl',
        time_m_ago: 'dəq əvvəl',
        time_h_ago: 'saat əvvəl',
    }
};

let currentLang = 'en';

function t(key) {
    return LANG[currentLang]?.[key] || LANG.en[key] || key;
}

function applyTranslations() {
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.getAttribute('data-i18n');
        const val = t(key);
        if (val) el.innerHTML = val;
    });
    document.title = currentLang === 'az'
        ? 'UpDown.az — Azərbaycan İnternet Monitorinqi'
        : 'UpDown.az — Azerbaijan Internet Monitor';
    const btn = document.getElementById('langToggle');
    if (btn) btn.textContent = currentLang === 'az' ? 'EN' : 'AZ';
    document.documentElement.lang = currentLang;
    // Inject section SVG icons after translation
    injectSectionIcons();
}

function injectSectionIcons() {
    if (typeof ICONS === 'undefined') return;
    const map = { shNodes: 'server', shIncidents: 'alertTri', shBGP: 'radio', shConf: 'gauge' };
    Object.entries(map).forEach(([id, icon]) => {
        const el = document.getElementById(id);
        if (el && !el.querySelector('.icon')) {
            el.insertAdjacentHTML('afterbegin', ICONS[icon]() + ' ');
        }
    });
}

function toggleLang() {
    currentLang = currentLang === 'en' ? 'az' : 'en';
    localStorage.setItem('sentinel_lang', currentLang);
    applyTranslations();
    if (typeof refresh === 'function') refresh();
}

function setLang(lang) {
    currentLang = lang;
    localStorage.setItem('sentinel_lang', currentLang);
    applyTranslations();
}

(async function detectLanguage() {
    const saved = localStorage.getItem('sentinel_lang');
    if (saved && (saved === 'az' || saved === 'en')) {
        currentLang = saved;
        applyTranslations();
        return;
    }
    try {
        const resp = await fetch('https://ipapi.co/json/', { signal: AbortSignal.timeout(3000) });
        if (resp.ok) {
            const data = await resp.json();
            if (data.country_code === 'AZ') currentLang = 'az';
        }
    } catch { /* stay English */ }
    applyTranslations();
})();
