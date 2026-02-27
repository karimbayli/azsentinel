-- ============================================================
-- Sentinel V2: Migration 004 — Add bilingual names + citizen services
-- 1. Adds display_name_en column for English translations
-- 2. Adds GOV_SERVICE category for citizen-facing services
-- 3. Seeds citizen digital services used by diaspora abroad
-- 4. Backfills English names for all existing targets
-- ============================================================
-- Step 1: Add English display name + parent system columns
ALTER TABLE targets
ADD COLUMN IF NOT EXISTS display_name_en TEXT NOT NULL DEFAULT '';
ALTER TABLE targets
ADD COLUMN IF NOT EXISTS parent_system TEXT NOT NULL DEFAULT '';
-- Step 2: Add GOV_SERVICE category
ALTER TABLE targets DROP CONSTRAINT IF EXISTS targets_category_check;
ALTER TABLE targets
ADD CONSTRAINT targets_category_check CHECK (
        category IN (
            'GOV',
            'GOV_SERVICE',
            'BANK',
            'FINTECH',
            'ISP',
            'MEDIA',
            'OTHER',
            'GLOBAL',
            'ANCHOR'
        )
    );
-- ============================================================
-- Step 3: Seed citizen-facing digital services (GOV_SERVICE)
-- ============================================================
INSERT INTO targets (
        url,
        category,
        criticality,
        enabled,
        display_name,
        display_name_en
    )
VALUES -- ── Core Digital Government Platforms ─────────────────────
    (
        'https://my.gov.az',
        'GOV_SERVICE',
        10,
        true,
        'myGov Rəqəmsal Xidmətlər',
        'myGov Digital Services'
    ),
    (
        'https://mygovid.gov.az',
        'GOV_SERVICE',
        10,
        true,
        'myGov ID Rəqəmsal Şəxsiyyət',
        'myGov Digital ID'
    ),
    (
        'https://asanpay.az',
        'GOV_SERVICE',
        9,
        true,
        'ASAN Pay Ödənişlər',
        'ASAN Pay Payments'
    ),
    (
        'https://sima.az',
        'GOV_SERVICE',
        9,
        true,
        'SIMA Rəqəmsal İmza',
        'SIMA Digital Signature'
    ),
    -- ── ASAN Ecosystem ───────────────────────────────────────
    (
        'https://asan.gov.az',
        'GOV_SERVICE',
        10,
        true,
        'ASAN Xidmət Portalı',
        'ASAN Service Portal'
    ),
    (
        'https://asanviza.gov.az',
        'GOV_SERVICE',
        8,
        true,
        'ASAN Viza',
        'ASAN Visa'
    ),
    (
        'https://asanfinans.az',
        'GOV_SERVICE',
        8,
        true,
        'ASAN Finans',
        'ASAN Finance'
    ),
    -- ── E-Government Citizen Services ────────────────────────
    (
        'https://e-gov.az',
        'GOV_SERVICE',
        10,
        true,
        'Elektron Hökumət Portalı',
        'E-Government Portal'
    ),
    (
        'https://e-taxes.gov.az',
        'GOV_SERVICE',
        9,
        true,
        'Elektron Vergi Portalı',
        'E-Taxes Portal'
    ),
    (
        'https://epoint.az',
        'GOV_SERVICE',
        8,
        true,
        'ePoint Ödənişlər',
        'ePoint Payments'
    ),
    -- ── Utility Payment Portals (used from abroad) ───────────
    (
        'https://azersu.az',
        'GOV_SERVICE',
        7,
        true,
        'Azərsu (Su Ödənişi)',
        'Azersu (Water Utility)'
    ),
    (
        'https://azerishiq.az',
        'GOV_SERVICE',
        7,
        true,
        'Azərişıq (Elektrik Ödənişi)',
        'Azerishiq (Electricity)'
    ),
    (
        'https://azeriqaz.az',
        'GOV_SERVICE',
        7,
        true,
        'Azəriqaz (Qaz Ödənişi)',
        'Azeriqaz (Gas Utility)'
    ),
    -- ── Transport & Travel (citizens abroad booking) ─────────
    (
        'https://azal.az',
        'GOV_SERVICE',
        8,
        true,
        'AZAL Hava Yolları',
        'AZAL Airlines'
    ),
    (
        'https://ady.az',
        'GOV_SERVICE',
        7,
        true,
        'ADY Dəmiryolları Bilet',
        'ADY Railways Tickets'
    ) ON CONFLICT (url) DO
UPDATE
SET category = EXCLUDED.category,
    criticality = EXCLUDED.criticality,
    display_name = EXCLUDED.display_name,
    display_name_en = EXCLUDED.display_name_en;
-- ============================================================
-- Step 3b: Mobile App API Endpoints (discovered via Charles Proxy)
-- These are the REAL backend APIs that mobile apps connect to.
-- ============================================================
INSERT INTO targets (
        url,
        category,
        criticality,
        enabled,
        display_name,
        display_name_en
    )
VALUES -- ── myGov Ecosystem APIs ─────────────────────────────────
    (
        'https://mygov-api.e-gov.az',
        'GOV_SERVICE',
        10,
        true,
        'myGov API',
        'myGov API'
    ),
    (
        'https://mygov-apigw.e-gov.az',
        'GOV_SERVICE',
        10,
        true,
        'myGov API Gateway',
        'myGov API Gateway'
    ),
    (
        'https://mygov-cdn.e-gov.az',
        'GOV_SERVICE',
        8,
        true,
        'myGov CDN',
        'myGov CDN'
    ),
    (
        'https://api.mygovid.gov.az',
        'GOV_SERVICE',
        10,
        true,
        'myGov ID API',
        'myGov ID API'
    ),
    (
        'https://digital.login.gov.az',
        'GOV_SERVICE',
        10,
        true,
        'Rəqəmsal Login',
        'Digital Login Portal'
    ),
    (
        'https://apidigital.login.gov.az',
        'GOV_SERVICE',
        10,
        true,
        'Rəqəmsal Login API',
        'Digital Login API'
    ),
    -- ── E-Sosial Mobile API ──────────────────────────────────
    (
        'https://mapi.sosial.gov.az',
        'GOV_SERVICE',
        8,
        true,
        'E-Sosial Mobil API',
        'E-Social Mobile API'
    ),
    -- ── Banking Mobile APIs ──────────────────────────────────
    (
        'https://mbanking.abb-bank.az',
        'BANK',
        9,
        true,
        'ABB Mobil Bank API',
        'ABB Mobile Banking API'
    ),
    (
        'https://asm-prod.abb-bank.az',
        'BANK',
        9,
        true,
        'ABB Backend API',
        'ABB Backend API'
    ),
    (
        'https://octopus.leobank.az',
        'BANK',
        8,
        true,
        'LeoBank Mobil API',
        'LeoBank Mobile API'
    ),
    (
        'https://online.unibank.az',
        'BANK',
        8,
        true,
        'Unibank Onlayn API',
        'Unibank Online API'
    ),
    (
        'https://ibank-preprod.yelo.az',
        'BANK',
        7,
        true,
        'Yelo Bank Mobil API',
        'Yelo Bank Mobile API'
    ),
    -- ── Payment & Fintech APIs ───────────────────────────────
    (
        'https://evamx.pashapay.az',
        'FINTECH',
        9,
        true,
        'PashaPay API',
        'PashaPay API'
    ),
    (
        'https://analytics.m10payments.com',
        'FINTECH',
        8,
        true,
        'M10 Payments Backend',
        'M10 Payments Backend'
    ),
    (
        'https://api.m10.az',
        'FINTECH',
        9,
        true,
        'M10 Mobil API',
        'M10 Mobile API'
    ),
    (
        'https://pashapay.az',
        'FINTECH',
        9,
        true,
        'PashaPay',
        'PashaPay'
    ),
    (
        'https://leobank.az',
        'BANK',
        8,
        true,
        'LeoBank',
        'LeoBank'
    ),
    -- ── Telecom Mobile APIs ──────────────────────────────────
    (
        'https://kabinetim-app.azercell.com',
        'ISP',
        8,
        true,
        'Azercell Kabinetim API',
        'Azercell My Cabinet API'
    ),
    -- ── Marketplace Mobile APIs ──────────────────────────────
    (
        'https://gateway.umico.az',
        'OTHER',
        7,
        true,
        'Umico Mobil Gateway',
        'Umico Mobile Gateway'
    ),
    (
        'https://api.tap.az',
        'OTHER',
        7,
        true,
        'Tap.az Mobil API',
        'Tap.az Mobile API'
    ),
    (
        'https://api.turbo.az',
        'OTHER',
        7,
        true,
        'Turbo.az Mobil API',
        'Turbo.az Mobile API'
    ),
    (
        'https://api.bina.az',
        'OTHER',
        7,
        true,
        'Bina.az Mobil API',
        'Bina.az Mobile API'
    ),
    -- ── City Services Mobile APIs ────────────────────────────
    (
        'https://appapi.azparking.az',
        'OTHER',
        6,
        true,
        'AzParking Mobil API',
        'AzParking Mobile API'
    ),
    (
        'https://web.smsradar.az',
        'OTHER',
        6,
        true,
        'SMS Radar API',
        'SMS Radar API'
    ),
    (
        'https://app.bakubus.az',
        'OTHER',
        6,
        true,
        'BakuBus Mobil API',
        'BakuBus Mobile API'
    ),
    (
        'https://api.iticket.az',
        'OTHER',
        6,
        true,
        'iTicket Mobil API',
        'iTicket Mobile API'
    ),
    -- ── SIMA Biometric API (discovered via SIMA app) ─────────
    (
        'https://biosign-gateway-api.sima.az',
        'GOV_SERVICE',
        9,
        true,
        'SIMA Biometrik İmza API',
        'SIMA Biometric Signature API'
    ),
    -- ── PashaBank Mobile API (discovered via PashaBank app) ──
    (
        'https://mobile.pashabank.digital',
        'BANK',
        9,
        true,
        'PashaBank Mobil API',
        'PashaBank Mobile API'
    ),
    -- ── myGov ASAN Login (discovered via myGov app) ──────────
    (
        'https://asanlogin.my.gov.az',
        'GOV_SERVICE',
        10,
        true,
        'ASAN Login (myGov)',
        'ASAN Login (myGov)'
    ) ON CONFLICT (url) DO
UPDATE
SET category = EXCLUDED.category,
    criticality = EXCLUDED.criticality,
    display_name = EXCLUDED.display_name,
    display_name_en = EXCLUDED.display_name_en;
-- ============================================================
-- Step 4: Backfill English names for ALL existing targets
-- ============================================================
-- ── Government ───────────────────────────────────────────────
UPDATE targets
SET display_name_en = 'Tax Service'
WHERE url = 'https://taxes.gov.az';
UPDATE targets
SET display_name_en = 'Presidential Administration'
WHERE url = 'https://president.az';
UPDATE targets
SET display_name_en = 'National Parliament'
WHERE url = 'https://meclis.gov.az';
UPDATE targets
SET display_name_en = 'Ministry of Internal Affairs'
WHERE url = 'https://mia.gov.az';
UPDATE targets
SET display_name_en = 'Ministry of Foreign Affairs'
WHERE url = 'https://mfa.gov.az';
UPDATE targets
SET display_name_en = 'Ministry of Education'
WHERE url = 'https://edu.gov.az';
UPDATE targets
SET display_name_en = 'Ministry of Health'
WHERE url = 'https://health.gov.az';
UPDATE targets
SET display_name_en = 'Ministry of Economy'
WHERE url = 'https://economy.gov.az';
UPDATE targets
SET display_name_en = 'Ministry of Justice'
WHERE url = 'https://justice.gov.az';
UPDATE targets
SET display_name_en = 'State Statistics Committee'
WHERE url = 'https://stat.gov.az';
UPDATE targets
SET display_name_en = 'Customs Committee'
WHERE url = 'https://customs.gov.az';
UPDATE targets
SET display_name_en = 'Migration Service'
WHERE url = 'https://migration.gov.az';
UPDATE targets
SET display_name_en = 'Ministry of Defense'
WHERE url = 'https://mod.gov.az';
UPDATE targets
SET display_name_en = 'Ministry of Emergency'
WHERE url = 'https://fhn.gov.az';
UPDATE targets
SET display_name_en = 'Ministry of Digital Development'
WHERE url = 'https://mincom.gov.az';
UPDATE targets
SET display_name_en = 'Ministry of Ecology'
WHERE url = 'https://eco.gov.az';
UPDATE targets
SET display_name_en = 'Ministry of Transport'
WHERE url = 'https://mst.gov.az';
UPDATE targets
SET display_name_en = 'Ministry of Labour & Social'
WHERE url = 'https://sosial.gov.az';
UPDATE targets
SET display_name_en = 'Ministry of Culture'
WHERE url = 'https://culture.gov.az';
UPDATE targets
SET display_name_en = 'Ministry of Sports'
WHERE url = 'https://sport.gov.az';
UPDATE targets
SET display_name_en = 'Ministry of Science & Education'
WHERE url = 'https://science.gov.az';
UPDATE targets
SET display_name_en = 'State Border Service'
WHERE url = 'https://dsx.gov.az';
UPDATE targets
SET display_name_en = 'Culture Portal'
WHERE url = 'https://mek.gov.az';
UPDATE targets
SET display_name_en = 'AzExport'
WHERE url = 'https://azexport.az';
UPDATE targets
SET display_name_en = 'Cabinet of Ministers'
WHERE url = 'https://nk.gov.az';
UPDATE targets
SET display_name_en = 'Social Protection Fund'
WHERE url = 'https://sspf.gov.az';
UPDATE targets
SET display_name_en = 'Central Bank'
WHERE url = 'https://cbar.az';
-- ── Banking & Finance ────────────────────────────────────────
UPDATE targets
SET display_name_en = 'ABB Bank'
WHERE url = 'https://abb-bank.az';
UPDATE targets
SET display_name_en = 'PASHA Bank'
WHERE url = 'https://pashabank.az';
UPDATE targets
SET display_name_en = 'Kapital Bank'
WHERE url = 'https://kapitalbank.az';
UPDATE targets
SET display_name_en = 'Rabitabank'
WHERE url = 'https://rabitabank.az';
UPDATE targets
SET display_name_en = 'Yelo Bank'
WHERE url = 'https://yelo.az';
UPDATE targets
SET display_name_en = 'Expressbank'
WHERE url = 'https://expressbank.az';
UPDATE targets
SET display_name_en = 'Nikoil Bank'
WHERE url = 'https://nikoil.az';
UPDATE targets
SET display_name_en = 'AccessBank'
WHERE url = 'https://accessbank.az';
UPDATE targets
SET display_name_en = 'Bank Respublika'
WHERE url = 'https://bankrespublika.az';
UPDATE targets
SET display_name_en = 'Xalq Bank'
WHERE url = 'https://xalqbank.az';
UPDATE targets
SET display_name_en = 'Unibank'
WHERE url = 'https://unibank.az';
UPDATE targets
SET display_name_en = 'International Bank'
WHERE url = 'https://ibar.az';
UPDATE targets
SET display_name_en = 'AzerTurkBank'
WHERE url = 'https://azerturkbank.az';
UPDATE targets
SET display_name_en = 'BirBank (Kapital)'
WHERE url = 'https://birbank.az';
UPDATE targets
SET display_name_en = 'M10 Payments'
WHERE url = 'https://m10.az';
-- ── Media & News ─────────────────────────────────────────────
UPDATE targets
SET display_name_en = 'Oxu.az News'
WHERE url = 'https://oxu.az';
UPDATE targets
SET display_name_en = 'Report.az News'
WHERE url = 'https://report.az';
UPDATE targets
SET display_name_en = '1news.az'
WHERE url = 'https://1news.az';
UPDATE targets
SET display_name_en = 'Haqqin.az'
WHERE url = 'https://haqqin.az';
UPDATE targets
SET display_name_en = 'Modern.az'
WHERE url = 'https://modern.az';
UPDATE targets
SET display_name_en = 'APA News Agency'
WHERE url = 'https://apa.az';
UPDATE targets
SET display_name_en = 'AzerTAc (State Agency)'
WHERE url = 'https://azertag.az';
UPDATE targets
SET display_name_en = 'Trend News Agency'
WHERE url = 'https://trend.az';
UPDATE targets
SET display_name_en = 'Lent.az'
WHERE url = 'https://lent.az';
UPDATE targets
SET display_name_en = 'Day.az'
WHERE url = 'https://day.az';
UPDATE targets
SET display_name_en = 'AzVision.az'
WHERE url = 'https://azvision.az';
UPDATE targets
SET display_name_en = 'Turan Agency'
WHERE url = 'https://turan.az';
UPDATE targets
SET display_name_en = 'ABC.az'
WHERE url = 'https://abc.az';
UPDATE targets
SET display_name_en = 'Publika.az'
WHERE url = 'https://publika.az';
UPDATE targets
SET display_name_en = 'Qafqazinfo.az'
WHERE url = 'https://qafqazinfo.az';
UPDATE targets
SET display_name_en = 'Musavat.az'
WHERE url = 'https://musavat.az';
UPDATE targets
SET display_name_en = 'AzTV'
WHERE url = 'https://aztv.az';
UPDATE targets
SET display_name_en = 'Public TV (Ictimai)'
WHERE url = 'https://ictimai.tv';
UPDATE targets
SET display_name_en = 'Xezer TV'
WHERE url = 'https://xezerxeber.az';
UPDATE targets
SET display_name_en = 'CBC TV'
WHERE url = 'https://cbc.az';
UPDATE targets
SET display_name_en = 'ANS Press'
WHERE url = 'https://anspress.com';
UPDATE targets
SET display_name_en = 'Olke.az'
WHERE url = 'https://olke.az';
UPDATE targets
SET display_name_en = 'Yeni Sabah'
WHERE url = 'https://yenisabah.az';
UPDATE targets
SET display_name_en = 'Bizim Yol'
WHERE url = 'https://bizimyol.info';
UPDATE targets
SET display_name_en = 'Virtual.az'
WHERE url = 'https://virtualaz.org';
UPDATE targets
SET display_name_en = 'Baku.ws'
WHERE url = 'https://baku.ws';
UPDATE targets
SET display_name_en = 'Azadliq Radio'
WHERE url = 'https://azadliq.info';
UPDATE targets
SET display_name_en = 'Meydan TV'
WHERE url = 'https://meydan.tv';
UPDATE targets
SET display_name_en = 'Media.az'
WHERE url = 'https://mediaaz.az';
UPDATE targets
SET display_name_en = 'Caliber.az'
WHERE url = 'https://caliber.az';
-- ── ISP & Telecom ────────────────────────────────────────────
UPDATE targets
SET display_name_en = 'Delta Telecom (Backbone Provider)'
WHERE url = 'https://delta.az';
UPDATE targets
SET display_name_en = 'Aztelekom (Backbone Provider)'
WHERE url = 'https://aztelekom.az';
UPDATE targets
SET display_name_en = 'Bakcell'
WHERE url = 'https://bakcell.az';
UPDATE targets
SET display_name_en = 'Nar Mobile'
WHERE url = 'https://nar.az';
UPDATE targets
SET display_name_en = 'Azercell'
WHERE url = 'https://azercell.com';
UPDATE targets
SET display_name_en = 'CityNet'
WHERE url = 'https://citynet.az';
UPDATE targets
SET display_name_en = 'Connect'
WHERE url = 'https://connect.az';
UPDATE targets
SET display_name_en = 'KATV1'
WHERE url = 'https://katv1.az';
UPDATE targets
SET display_name_en = 'Ultel'
WHERE url = 'https://ultel.az';
UPDATE targets
SET display_name_en = 'Starlink Azerbaijan'
WHERE url = 'https://starlink.az';
-- ── Other Popular AZ Sites ───────────────────────────────────
UPDATE targets
SET display_name_en = 'Tap.az (Classifieds)'
WHERE url = 'https://tap.az';
UPDATE targets
SET display_name_en = 'Boss.az (Jobs)'
WHERE url = 'https://boss.az';
UPDATE targets
SET display_name_en = 'Bina.az (Real Estate)'
WHERE url = 'https://bina.az';
UPDATE targets
SET display_name_en = 'Turbo.az (Cars)'
WHERE url = 'https://turbo.az';
UPDATE targets
SET display_name_en = 'Lalafo.az'
WHERE url = 'https://lalafo.az';
UPDATE targets
SET display_name_en = 'Umico (E-Commerce)'
WHERE url = 'https://umico.az';
UPDATE targets
SET display_name_en = 'Kontakt Home'
WHERE url = 'https://kontakt.az';
UPDATE targets
SET display_name_en = 'Baku Electronics'
WHERE url = 'https://bakuelectronics.az';
UPDATE targets
SET display_name_en = 'iTicket.az'
WHERE url = 'https://iticket.az';
UPDATE targets
SET display_name_en = 'SOCAR'
WHERE url = 'https://socar.az';
UPDATE targets
SET display_name_en = 'BakuBus'
WHERE url = 'https://bakubus.az';
UPDATE targets
SET display_name_en = 'Baku Metro'
WHERE url = 'https://bakumetro.gov.az';
UPDATE targets
SET display_name_en = 'ADA University'
WHERE url = 'https://ada.edu.az';
UPDATE targets
SET display_name_en = 'UNEC University'
WHERE url = 'https://unec.edu.az';
UPDATE targets
SET display_name_en = 'BSU University'
WHERE url = 'https://bsu.edu.az';
UPDATE targets
SET display_name_en = 'ASOIU University'
WHERE url = 'https://asoiu.edu.az';
UPDATE targets
SET display_name_en = 'EmbaFinans'
WHERE url = 'https://embafinans.com';
UPDATE targets
SET display_name_en = 'Wolt Delivery'
WHERE url = 'https://wolt.com';
UPDATE targets
SET display_name_en = 'Bolt Ride'
WHERE url = 'https://bolt.eu';
UPDATE targets
SET display_name_en = 'Bravo Supermarket'
WHERE url = 'https://bravo.az';
UPDATE targets
SET display_name_en = 'Neptun Electronics'
WHERE url = 'https://neptun.az';
UPDATE targets
SET display_name_en = 'Irshad Electronics'
WHERE url = 'https://irshad.az';
UPDATE targets
SET display_name_en = 'CompNet'
WHERE url = 'https://compnet.az';
-- ── Global Anchors (same in both languages) ──────────────────
UPDATE targets
SET display_name_en = 'WhatsApp'
WHERE url = 'https://whatsapp.com';
UPDATE targets
SET display_name_en = 'Telegram'
WHERE url = 'https://telegram.org';
UPDATE targets
SET display_name_en = 'GitHub'
WHERE url = 'https://github.com';
UPDATE targets
SET display_name_en = 'Apple'
WHERE url = 'https://apple.com';
UPDATE targets
SET display_name_en = 'Netflix'
WHERE url = 'https://netflix.com';
UPDATE targets
SET display_name_en = 'Spotify'
WHERE url = 'https://spotify.com';
UPDATE targets
SET display_name_en = 'Reddit'
WHERE url = 'https://reddit.com';
UPDATE targets
SET display_name_en = 'YouTube'
WHERE url = 'https://youtube.com';
UPDATE targets
SET display_name_en = 'Facebook'
WHERE url = 'https://facebook.com';
UPDATE targets
SET display_name_en = 'Instagram'
WHERE url = 'https://instagram.com';
UPDATE targets
SET display_name_en = 'TikTok'
WHERE url = 'https://tiktok.com';
UPDATE targets
SET display_name_en = 'X (Twitter)'
WHERE url = 'https://x.com';
UPDATE targets
SET display_name_en = 'LinkedIn'
WHERE url = 'https://linkedin.com';
UPDATE targets
SET display_name_en = 'Wikipedia'
WHERE url = 'https://wikipedia.org';
UPDATE targets
SET display_name_en = 'Amazon'
WHERE url = 'https://amazon.com';
UPDATE targets
SET display_name_en = 'Microsoft'
WHERE url = 'https://microsoft.com';
UPDATE targets
SET display_name_en = 'Cloudflare DNS'
WHERE url = 'https://1.1.1.1';
UPDATE targets
SET display_name_en = 'Cloudflare'
WHERE url = 'https://cloudflare.com';
UPDATE targets
SET display_name_en = 'Google DNS'
WHERE url = 'https://8.8.8.8';
-- ============================================================
-- Step 5: Assign parent_system for enterprise service grouping
-- Groups related URLs (website + APIs) under one logical system
-- ============================================================
-- ── myGov System ─────────────────────────────────────────────
UPDATE targets
SET parent_system = 'mygov'
WHERE url IN (
        'https://my.gov.az',
        'https://mygovid.gov.az',
        'https://mygov-api.e-gov.az',
        'https://mygov-apigw.e-gov.az',
        'https://mygov-cdn.e-gov.az',
        'https://api.mygovid.gov.az',
        'https://digital.login.gov.az',
        'https://apidigital.login.gov.az',
        'https://asanlogin.my.gov.az'
    );
-- ── E-Government System ──────────────────────────────────────
UPDATE targets
SET parent_system = 'egov'
WHERE url IN (
        'https://e-gov.az',
        'https://e-taxes.gov.az'
    );
-- ── ASAN System ──────────────────────────────────────────────
UPDATE targets
SET parent_system = 'asan'
WHERE url IN (
        'https://asan.gov.az',
        'https://asanpay.az',
        'https://asanviza.gov.az',
        'https://asanfinans.az'
    );
-- ── SIMA System ──────────────────────────────────────────────
UPDATE targets
SET parent_system = 'sima'
WHERE url IN (
        'https://sima.az',
        'https://biosign-gateway-api.sima.az'
    );
-- ── E-Sosial System ──────────────────────────────────────────
UPDATE targets
SET parent_system = 'esosial'
WHERE url IN (
        'https://sosial.gov.az',
        'https://mapi.sosial.gov.az'
    );
-- ── ABB Bank System ──────────────────────────────────────────
UPDATE targets
SET parent_system = 'abb'
WHERE url IN (
        'https://abb-bank.az',
        'https://mbanking.abb-bank.az',
        'https://asm-prod.abb-bank.az'
    );
-- ── Kapital Bank / BirBank System ────────────────────────────
UPDATE targets
SET parent_system = 'kapitalbank'
WHERE url IN (
        'https://kapitalbank.az',
        'https://birbank.az'
    );
-- ── M10 / PashaPay System ────────────────────────────────────
UPDATE targets
SET parent_system = 'm10'
WHERE url IN (
        'https://m10.az',
        'https://api.m10.az',
        'https://evamx.pashapay.az',
        'https://pashapay.az',
        'https://analytics.m10payments.com'
    );
-- ── PASHA Bank System ────────────────────────────────────────
UPDATE targets
SET parent_system = 'pashabank'
WHERE url IN (
        'https://pashabank.az',
        'https://mobile.pashabank.digital'
    );
-- ── LeoBank System ───────────────────────────────────────────
UPDATE targets
SET parent_system = 'leobank'
WHERE url IN (
        'https://leobank.az',
        'https://octopus.leobank.az'
    );
-- ── Unibank System ───────────────────────────────────────────
UPDATE targets
SET parent_system = 'unibank'
WHERE url IN (
        'https://unibank.az',
        'https://online.unibank.az'
    );
-- ── Yelo Bank System ─────────────────────────────────────────
UPDATE targets
SET parent_system = 'yelo'
WHERE url IN (
        'https://yelo.az',
        'https://ibank-preprod.yelo.az'
    );
-- ── Azercell System ──────────────────────────────────────────
UPDATE targets
SET parent_system = 'azercell'
WHERE url IN (
        'https://azercell.com',
        'https://kabinetim-app.azercell.com'
    );
-- ── Nar Mobile System ────────────────────────────────────────
UPDATE targets
SET parent_system = 'nar'
WHERE url IN ('https://nar.az');
-- ── Bakcell System ───────────────────────────────────────────
UPDATE targets
SET parent_system = 'bakcell'
WHERE url IN ('https://bakcell.az');
-- ── Delta Telecom System ─────────────────────────────────────
UPDATE targets
SET parent_system = 'delta'
WHERE url IN ('https://delta.az');
-- ── Aztelekom System ─────────────────────────────────────────
UPDATE targets
SET parent_system = 'aztelekom'
WHERE url IN ('https://aztelekom.az');
-- ── Umico System ─────────────────────────────────────────────
UPDATE targets
SET parent_system = 'umico'
WHERE url IN (
        'https://umico.az',
        'https://gateway.umico.az'
    );
-- ── Tap.az System ────────────────────────────────────────────
UPDATE targets
SET parent_system = 'tap'
WHERE url IN ('https://tap.az', 'https://api.tap.az');
-- ── Turbo.az System ──────────────────────────────────────────
UPDATE targets
SET parent_system = 'turbo'
WHERE url IN ('https://turbo.az', 'https://api.turbo.az');
-- ── Bina.az System ───────────────────────────────────────────
UPDATE targets
SET parent_system = 'bina'
WHERE url IN ('https://bina.az', 'https://api.bina.az');
-- ── Utility Systems ──────────────────────────────────────────
UPDATE targets
SET parent_system = 'azersu'
WHERE url = 'https://azersu.az';
UPDATE targets
SET parent_system = 'azerishiq'
WHERE url = 'https://azerishiq.az';
UPDATE targets
SET parent_system = 'azeriqaz'
WHERE url = 'https://azeriqaz.az';
-- ── Transport Systems ────────────────────────────────────────
UPDATE targets
SET parent_system = 'azal'
WHERE url = 'https://azal.az';
UPDATE targets
SET parent_system = 'ady'
WHERE url = 'https://ady.az';
UPDATE targets
SET parent_system = 'bakubus'
WHERE url IN (
        'https://bakubus.az',
        'https://app.bakubus.az'
    );
UPDATE targets
SET parent_system = 'bakumetro'
WHERE url = 'https://bakumetro.gov.az';
-- ── City & Other Services ────────────────────────────────────
UPDATE targets
SET parent_system = 'azparking'
WHERE url = 'https://appapi.azparking.az';
UPDATE targets
SET parent_system = 'smsradar'
WHERE url = 'https://web.smsradar.az';
UPDATE targets
SET parent_system = 'iticket'
WHERE url IN (
        'https://iticket.az',
        'https://api.iticket.az'
    );
UPDATE targets
SET parent_system = 'epoint'
WHERE url = 'https://epoint.az';
UPDATE targets
SET parent_system = 'socar'
WHERE url = 'https://socar.az';
UPDATE targets
SET parent_system = 'lalafo'
WHERE url = 'https://lalafo.az';
-- ── Remaining Banks (standalone) ─────────────────────────────
UPDATE targets
SET parent_system = 'rabitabank'
WHERE url = 'https://rabitabank.az';
UPDATE targets
SET parent_system = 'expressbank'
WHERE url = 'https://expressbank.az';
UPDATE targets
SET parent_system = 'nikoil'
WHERE url = 'https://nikoil.az';
UPDATE targets
SET parent_system = 'accessbank'
WHERE url = 'https://accessbank.az';
UPDATE targets
SET parent_system = 'bankrespublika'
WHERE url = 'https://bankrespublika.az';
UPDATE targets
SET parent_system = 'xalqbank'
WHERE url = 'https://xalqbank.az';
UPDATE targets
SET parent_system = 'ibar'
WHERE url = 'https://ibar.az';
UPDATE targets
SET parent_system = 'azerturkbank'
WHERE url = 'https://azerturkbank.az';
UPDATE targets
SET parent_system = 'cbar'
WHERE url = 'https://cbar.az';