const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const repoRoot = path.join(__dirname, '..');
const appJS = fs.readFileSync(path.join(repoRoot, 'static', 'app.js'), 'utf8');
const i18nJS = fs.readFileSync(path.join(repoRoot, 'static', 'i18n.js'), 'utf8');
const faviconSVG = fs.readFileSync(path.join(repoRoot, 'static', 'favicon.svg'), 'utf8');
const indexHTML = fs.readFileSync(path.join(repoRoot, 'static', 'index.html'), 'utf8');

function createElement(id = '') {
    return {
        id,
        innerHTML: '',
        textContent: '',
        style: {},
        classList: {
            add() { },
            remove() { },
            toggle() { return true; },
        },
        insertAdjacentHTML(position, html) {
            this.innerHTML = position === 'afterbegin' ? html + this.innerHTML : this.innerHTML + html;
        },
        querySelector() {
            return null;
        },
        getAttribute() {
            return null;
        },
    };
}

function createLocalStorage(initial = {}) {
    const store = new Map(Object.entries(initial));
    return {
        getItem(key) {
            return store.has(key) ? store.get(key) : null;
        },
        setItem(key, value) {
            store.set(key, String(value));
        },
    };
}

function createDashboardContext(fixtures = {}) {
    const elements = {
        azMapContainer: createElement('azMapContainer'),
        incidentGrid: createElement('incidentGrid'),
        kpiBGP: createElement('kpiBGP'),
        kpiConfidence: createElement('kpiConfidence'),
        kpiEndpoints: createElement('kpiEndpoints'),
        kpiSystems: createElement('kpiSystems'),
        systemList: createElement('systemList'),
        topEndpoints: createElement('topEndpoints'),
        topIncidents: createElement('topIncidents'),
        topLastScan: createElement('topLastScan'),
        topStatus: createElement('topStatus'),
    };

    const document = {
        documentElement: { lang: 'en' },
        title: 'Netwatch.az',
        getElementById(id) {
            return elements[id] || null;
        },
        querySelector(selector) {
            if (selector.startsWith('#')) {
                return elements[selector.slice(1)] || null;
            }
            return null;
        },
        querySelectorAll() {
            return [];
        },
    };

    const fetch = async (url) => {
        if (url.endsWith('/status')) {
            return { ok: true, json: async () => fixtures.statuses || [] };
        }
        if (url.endsWith('/nodes')) {
            return { ok: true, json: async () => fixtures.nodes || [] };
        }
        if (url.includes('/bgp/events')) {
            return { ok: true, json: async () => fixtures.bgp || [] };
        }
        if (url.includes('/incidents')) {
            return { ok: true, json: async () => fixtures.incidents || [] };
        }
        return { ok: false, json: async () => null };
    };

    const context = {
        AbortSignal: { timeout: () => undefined },
        Date,
        Math,
        Promise,
        console,
        clearInterval() { },
        clearTimeout,
        document,
        fetch,
        localStorage: createLocalStorage(),
        setInterval() { return 1; },
        setTimeout,
        window: { location: { href: '#' } },
    };

    vm.createContext(context);
    return { context, elements };
}

function createI18nContext(savedLang) {
    const langToggle = createElement('langToggle');
    const localStorage = createLocalStorage(
        savedLang ? { sentinel_lang: savedLang } : {},
    );

    const document = {
        documentElement: { lang: 'en' },
        title: 'Netwatch.az',
        getElementById(id) {
            if (id === 'langToggle') {
                return langToggle;
            }
            return null;
        },
        querySelectorAll() {
            return [];
        },
    };

    const context = {
        AbortSignal: { timeout: () => undefined },
        Promise,
        console,
        document,
        fetch: async () => ({ ok: false, json: async () => ({}) }),
        localStorage,
        setTimeout,
        window: {},
    };

    vm.createContext(context);
    return { context, langToggle, localStorage };
}

test('dashboard refresh renders system health without runtime errors', async () => {
    const fixtures = {
        statuses: [
            {
                target: {
                    category: 'GOV',
                    parent_system: 'egov',
                    display_name_en: 'E-Government Portal',
                    display_name: 'E-Government Portal',
                    url: 'https://egov.az',
                },
                status: 'HEALTHY',
                confidence: 0.93,
            },
            {
                target: {
                    category: 'GOV',
                    parent_system: 'egov',
                    display_name_en: 'E-Government Payments',
                    display_name: 'E-Government Payments',
                    url: 'https://payments.egov.az',
                },
                status: 'PARTIAL_OUTAGE',
                confidence: 0.41,
            },
            {
                target: {
                    category: 'BANK',
                    parent_system: '',
                    display_name_en: 'Kapital Bank',
                    display_name: 'Kapital Bank',
                    url: 'https://kapitalbank.az',
                },
                status: 'DEGRADED',
                confidence: 0.68,
            },
        ],
        nodes: [
            {
                node_id: 'node-eu-central',
                is_alive: true,
                avg_latency_ms: 42,
            },
        ],
        bgp: [
            { event_type: 'WITHDRAW' },
        ],
        incidents: [
            {
                target_url: 'https://payments.egov.az',
                peak_status: 'PARTIAL_OUTAGE',
                peak_confidence: 0.84,
                signals_fired: ['NODE', 'BGP'],
                started_at: '2026-03-16T12:00:00Z',
                resolved_at: null,
            },
        ],
    };

    const { context, elements } = createDashboardContext(fixtures);
    vm.runInContext(appJS, context);

    await assert.doesNotReject(() => context.refresh());
    assert.match(elements.systemList.innerHTML, /sys-dot healthy/);
    assert.match(elements.systemList.innerHTML, /sys-dot outage/);
    assert.match(elements.systemList.innerHTML, /sys-dot degraded/);
    assert.equal(elements.topStatus.textContent, 'DEGRADED');
    assert.equal(String(elements.kpiEndpoints.textContent), '3');
    assert.match(elements.incidentGrid.innerHTML, /payments\.egov\.az/);
});

test('language toggle script still works with the dashboard button markup', async () => {
    const { context, langToggle, localStorage } = createI18nContext('en');
    vm.runInContext(i18nJS, context);

    await new Promise(resolve => setImmediate(resolve));

    assert.equal(typeof context.toggleLang, 'function');
    assert.equal(langToggle.textContent, 'AZ');

    context.toggleLang();

    assert.equal(langToggle.textContent, 'EN');
    assert.equal(localStorage.getItem('sentinel_lang'), 'az');
    assert.equal(context.document.documentElement.lang, 'az');
});

test('index wires the dashboard support assets', () => {
    assert.match(indexHTML, /rel="icon" href="favicon\.svg"/);
    assert.match(indexHTML, /id="langToggle"/);
    assert.match(indexHTML, /src="icons\.js"/);
    assert.match(indexHTML, /src="i18n\.js"/);
    assert.match(indexHTML, /src="app\.js"/);
});

test('favicon asset exists', () => {
    assert.match(faviconSVG, /<svg/);
    assert.match(faviconSVG, /linearGradient/);
});
