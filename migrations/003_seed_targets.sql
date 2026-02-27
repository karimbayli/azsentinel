-- ============================================================
-- Sentinel V2: Migration 003 — Expand categories + seed targets
-- Adds: OTHER, GLOBAL, FINTECH categories
-- Seeds: ~120 monitored targets across all categories
-- ============================================================
-- Step 1: Expand the category CHECK constraint
ALTER TABLE targets DROP CONSTRAINT IF EXISTS targets_category_check;
ALTER TABLE targets
ADD CONSTRAINT targets_category_check CHECK (
        category IN (
            'GOV',
            'BANK',
            'FINTECH',
            'ISP',
            'MEDIA',
            'OTHER',
            'GLOBAL',
            'ANCHOR'
        )
    );
-- Step 2: Seed comprehensive target list
-- Uses ON CONFLICT to be idempotent (safe to re-run)
-- ── Government (30) ────────────────────────────────────────
INSERT INTO targets (
        url,
        category,
        criticality,
        enabled,
        display_name
    )
VALUES (
        'https://e-gov.az',
        'GOV',
        10,
        true,
        'E-Government Portal'
    ),
    (
        'https://asan.gov.az',
        'GOV',
        10,
        true,
        'ASAN Xidmət'
    ),
    (
        'https://taxes.gov.az',
        'GOV',
        9,
        true,
        'Vergilər Nazirliyi'
    ),
    (
        'https://president.az',
        'GOV',
        9,
        true,
        'Prezident Administrasiyası'
    ),
    (
        'https://meclis.gov.az',
        'GOV',
        8,
        true,
        'Milli Məclis'
    ),
    (
        'https://mia.gov.az',
        'GOV',
        9,
        true,
        'Daxili İşlər Nazirliyi'
    ),
    (
        'https://mfa.gov.az',
        'GOV',
        8,
        true,
        'Xarici İşlər Nazirliyi'
    ),
    (
        'https://edu.gov.az',
        'GOV',
        8,
        true,
        'Təhsil Nazirliyi'
    ),
    (
        'https://health.gov.az',
        'GOV',
        8,
        true,
        'Səhiyyə Nazirliyi'
    ),
    (
        'https://economy.gov.az',
        'GOV',
        8,
        true,
        'İqtisadiyyat Nazirliyi'
    ),
    (
        'https://justice.gov.az',
        'GOV',
        8,
        true,
        'Ədliyyə Nazirliyi'
    ),
    (
        'https://stat.gov.az',
        'GOV',
        7,
        true,
        'Dövlət Statistika Komitəsi'
    ),
    (
        'https://customs.gov.az',
        'GOV',
        8,
        true,
        'Gömrük Komitəsi'
    ),
    (
        'https://migration.gov.az',
        'GOV',
        8,
        true,
        'Miqrasiya Xidməti'
    ),
    (
        'https://mod.gov.az',
        'GOV',
        8,
        true,
        'Müdafiə Nazirliyi'
    ),
    (
        'https://fhn.gov.az',
        'GOV',
        8,
        true,
        'Fövqəladə Hallar Nazirliyi'
    ),
    (
        'https://mincom.gov.az',
        'GOV',
        8,
        true,
        'Rəqəmsal İnkişaf Nazirliyi'
    ),
    (
        'https://eco.gov.az',
        'GOV',
        7,
        true,
        'Ekologiya Nazirliyi'
    ),
    (
        'https://mst.gov.az',
        'GOV',
        7,
        true,
        'Nəqliyyat Nazirliyi'
    ),
    (
        'https://sosial.gov.az',
        'GOV',
        7,
        true,
        'Əmək və Sosial Nazirliyi'
    ),
    (
        'https://culture.gov.az',
        'GOV',
        6,
        true,
        'Mədəniyyət Nazirliyi'
    ),
    (
        'https://sport.gov.az',
        'GOV',
        6,
        true,
        'İdman Nazirliyi'
    ),
    (
        'https://science.gov.az',
        'GOV',
        6,
        true,
        'Elm və Təhsil Nazirliyi'
    ),
    (
        'https://dsx.gov.az',
        'GOV',
        7,
        true,
        'Dövlət Sərhəd Xidməti'
    ),
    (
        'https://mek.gov.az',
        'GOV',
        7,
        true,
        'Mədəniyyət Nazirliyi Portal'
    ),
    (
        'https://azexport.az',
        'GOV',
        7,
        true,
        'AzExport'
    ),
    (
        'https://e-taxes.gov.az',
        'GOV',
        9,
        true,
        'E-Taxes Portal'
    ),
    (
        'https://nk.gov.az',
        'GOV',
        6,
        true,
        'Nazirlər Kabineti'
    ),
    (
        'https://sspf.gov.az',
        'GOV',
        7,
        true,
        'DSMF (Sosial Müdafiə)'
    ),
    (
        'https://cbar.az',
        'GOV',
        9,
        true,
        'Mərkəzi Bank'
    ) ON CONFLICT (url) DO
UPDATE
SET category = EXCLUDED.category,
    criticality = EXCLUDED.criticality,
    display_name = EXCLUDED.display_name;
-- ── Banking & Finance (15) ─────────────────────────────────
INSERT INTO targets (
        url,
        category,
        criticality,
        enabled,
        display_name
    )
VALUES (
        'https://abb-bank.az',
        'BANK',
        9,
        true,
        'ABB Bank'
    ),
    (
        'https://pashabank.az',
        'BANK',
        9,
        true,
        'PASHA Bank'
    ),
    (
        'https://kapitalbank.az',
        'BANK',
        9,
        true,
        'Kapital Bank'
    ),
    (
        'https://rabitabank.az',
        'BANK',
        7,
        true,
        'Rabitabank'
    ),
    (
        'https://yelo.az',
        'BANK',
        7,
        true,
        'Yelo Bank'
    ),
    (
        'https://expressbank.az',
        'BANK',
        7,
        true,
        'Expressbank'
    ),
    (
        'https://nikoil.az',
        'BANK',
        7,
        true,
        'Nikoil Bank'
    ),
    (
        'https://accessbank.az',
        'BANK',
        7,
        true,
        'AccessBank'
    ),
    (
        'https://bankrespublika.az',
        'BANK',
        7,
        true,
        'Bank Respublika'
    ),
    (
        'https://xalqbank.az',
        'BANK',
        7,
        true,
        'Xalq Bank'
    ),
    (
        'https://unibank.az',
        'BANK',
        7,
        true,
        'Unibank'
    ),
    (
        'https://ibar.az',
        'BANK',
        8,
        true,
        'Beynəlxalq Bank'
    ),
    (
        'https://azerturkbank.az',
        'BANK',
        6,
        true,
        'AzərTürkBank'
    ),
    (
        'https://birbank.az',
        'BANK',
        9,
        true,
        'BirBank (Kapital)'
    ),
    (
        'https://m10.az',
        'FINTECH',
        8,
        true,
        'M10 Payments'
    ) ON CONFLICT (url) DO
UPDATE
SET category = EXCLUDED.category,
    criticality = EXCLUDED.criticality,
    display_name = EXCLUDED.display_name;
-- ── Media & News (30) ──────────────────────────────────────
INSERT INTO targets (
        url,
        category,
        criticality,
        enabled,
        display_name
    )
VALUES (
        'https://oxu.az',
        'MEDIA',
        7,
        true,
        'Oxu.az'
    ),
    (
        'https://report.az',
        'MEDIA',
        7,
        true,
        'Report.az'
    ),
    (
        'https://1news.az',
        'MEDIA',
        6,
        true,
        '1news.az'
    ),
    (
        'https://haqqin.az',
        'MEDIA',
        6,
        true,
        'Haqqin.az'
    ),
    (
        'https://modern.az',
        'MEDIA',
        6,
        true,
        'Modern.az'
    ),
    (
        'https://apa.az',
        'MEDIA',
        7,
        true,
        'APA (Azərbaycan Press)'
    ),
    (
        'https://azertag.az',
        'MEDIA',
        8,
        true,
        'AzərTAc (Dövlət)'
    ),
    (
        'https://trend.az',
        'MEDIA',
        7,
        true,
        'Trend.az'
    ),
    (
        'https://lent.az',
        'MEDIA',
        5,
        true,
        'Lent.az'
    ),
    (
        'https://day.az',
        'MEDIA',
        6,
        true,
        'Day.az'
    ),
    (
        'https://azvision.az',
        'MEDIA',
        6,
        true,
        'AzVision.az'
    ),
    (
        'https://turan.az',
        'MEDIA',
        6,
        true,
        'Turan Agentliyi'
    ),
    (
        'https://abc.az',
        'MEDIA',
        5,
        true,
        'ABC.az'
    ),
    (
        'https://publika.az',
        'MEDIA',
        5,
        true,
        'Publika.az'
    ),
    (
        'https://qafqazinfo.az',
        'MEDIA',
        5,
        true,
        'Qafqazinfo.az'
    ),
    (
        'https://musavat.az',
        'MEDIA',
        5,
        true,
        'Müsavat.az'
    ),
    (
        'https://aztv.az',
        'MEDIA',
        7,
        true,
        'AzTV'
    ),
    (
        'https://ictimai.tv',
        'MEDIA',
        7,
        true,
        'İctimai TV'
    ),
    (
        'https://xezerxeber.az',
        'MEDIA',
        6,
        true,
        'Xəzər TV'
    ),
    (
        'https://cbc.az',
        'MEDIA',
        6,
        true,
        'CBC TV'
    ),
    (
        'https://anspress.com',
        'MEDIA',
        5,
        true,
        'ANS Press'
    ),
    (
        'https://olke.az',
        'MEDIA',
        5,
        true,
        'Ölkə.az'
    ),
    (
        'https://yenisabah.az',
        'MEDIA',
        5,
        true,
        'Yeni Sabah'
    ),
    (
        'https://bizimyol.info',
        'MEDIA',
        5,
        true,
        'Bizim Yol'
    ),
    (
        'https://virtualaz.org',
        'MEDIA',
        5,
        true,
        'Virtual.az'
    ),
    (
        'https://baku.ws',
        'MEDIA',
        5,
        true,
        'Baku.ws'
    ),
    (
        'https://azadliq.info',
        'MEDIA',
        6,
        true,
        'Azadlıq'
    ),
    (
        'https://meydan.tv',
        'MEDIA',
        6,
        true,
        'Meydan TV'
    ),
    (
        'https://mediaaz.az',
        'MEDIA',
        5,
        true,
        'Media.az'
    ),
    (
        'https://caliber.az',
        'MEDIA',
        6,
        true,
        'Caliber.az'
    ) ON CONFLICT (url) DO
UPDATE
SET category = EXCLUDED.category,
    criticality = EXCLUDED.criticality,
    display_name = EXCLUDED.display_name;
-- ── ISP & Telecom (10) ─────────────────────────────────────
INSERT INTO targets (
        url,
        category,
        criticality,
        enabled,
        display_name
    )
VALUES (
        'https://delta.az',
        'ISP',
        8,
        true,
        'Delta Telecom'
    ),
    (
        'https://aztelekom.az',
        'ISP',
        8,
        true,
        'Aztelekom'
    ),
    (
        'https://bakcell.az',
        'ISP',
        7,
        true,
        'Bakcell'
    ),
    (
        'https://nar.az',
        'ISP',
        7,
        true,
        'Nar Mobile'
    ),
    (
        'https://azercell.com',
        'ISP',
        8,
        true,
        'Azercell'
    ),
    (
        'https://citynet.az',
        'ISP',
        6,
        true,
        'CityNet'
    ),
    (
        'https://connect.az',
        'ISP',
        6,
        true,
        'Connect'
    ),
    (
        'https://katv1.az',
        'ISP',
        6,
        true,
        'KATV1'
    ),
    (
        'https://ultel.az',
        'ISP',
        6,
        true,
        'Ultel'
    ),
    (
        'https://starlink.az',
        'ISP',
        5,
        true,
        'Starlink Azerbaijan'
    ) ON CONFLICT (url) DO
UPDATE
SET category = EXCLUDED.category,
    criticality = EXCLUDED.criticality,
    display_name = EXCLUDED.display_name;
-- ── Other Popular AZ Sites (30) ────────────────────────────
INSERT INTO targets (
        url,
        category,
        criticality,
        enabled,
        display_name
    )
VALUES (
        'https://tap.az',
        'OTHER',
        7,
        true,
        'Tap.az (Elanlar)'
    ),
    (
        'https://boss.az',
        'OTHER',
        6,
        true,
        'Boss.az (İş)'
    ),
    (
        'https://bina.az',
        'OTHER',
        7,
        true,
        'Bina.az (Daşınmaz)'
    ),
    (
        'https://turbo.az',
        'OTHER',
        7,
        true,
        'Turbo.az (Avtomobil)'
    ),
    (
        'https://lalafo.az',
        'OTHER',
        6,
        true,
        'Lalafo.az'
    ),
    (
        'https://umico.az',
        'OTHER',
        7,
        true,
        'Umico (E-Ticarət)'
    ),
    (
        'https://kontakt.az',
        'OTHER',
        6,
        true,
        'Kontakt Home'
    ),
    (
        'https://bakuelectronics.az',
        'OTHER',
        6,
        true,
        'Baku Electronics'
    ),
    (
        'https://azal.az',
        'OTHER',
        8,
        true,
        'AZAL (Hava Yolları)'
    ),
    (
        'https://iticket.az',
        'OTHER',
        6,
        true,
        'iTicket.az'
    ),
    (
        'https://ady.az',
        'OTHER',
        7,
        true,
        'ADY (Dəmiryolları)'
    ),
    (
        'https://socar.az',
        'OTHER',
        8,
        true,
        'SOCAR'
    ),
    (
        'https://azersu.az',
        'OTHER',
        7,
        true,
        'Azərsu (Su)'
    ),
    (
        'https://azerishiq.az',
        'OTHER',
        7,
        true,
        'Azərişıq (Elektrik)'
    ),
    (
        'https://azeriqaz.az',
        'OTHER',
        7,
        true,
        'Azəriqaz (Qaz)'
    ),
    (
        'https://bakubus.az',
        'OTHER',
        6,
        true,
        'BakuBus'
    ),
    (
        'https://bakumetro.gov.az',
        'OTHER',
        6,
        true,
        'Bakı Metropoliteni'
    ),
    (
        'https://ada.edu.az',
        'OTHER',
        6,
        true,
        'ADA Universiteti'
    ),
    (
        'https://unec.edu.az',
        'OTHER',
        5,
        true,
        'UNEC'
    ),
    (
        'https://bsu.edu.az',
        'OTHER',
        5,
        true,
        'BDU'
    ),
    (
        'https://asoiu.edu.az',
        'OTHER',
        5,
        true,
        'ADNSU'
    ),
    (
        'https://wolt.com',
        'OTHER',
        5,
        true,
        'Wolt Azerbaijan'
    ),
    (
        'https://bolt.eu',
        'OTHER',
        5,
        true,
        'Bolt'
    ),
    (
        'https://bravo.az',
        'OTHER',
        5,
        true,
        'Bravo Supermarket'
    ),
    (
        'https://neptun.az',
        'OTHER',
        5,
        true,
        'Neptun Elektronika'
    ),
    (
        'https://irshad.az',
        'OTHER',
        5,
        true,
        'İrşad Elektronika'
    ),
    (
        'https://compnet.az',
        'OTHER',
        5,
        true,
        'CompNet'
    ),
    (
        'https://embafinans.com',
        'OTHER',
        6,
        true,
        'Embafinans'
    ),
    (
        'https://day.az',
        'OTHER',
        6,
        true,
        'Day.az'
    ),
    (
        'https://epoint.az',
        'OTHER',
        6,
        true,
        'ePoint Ödəniş'
    ) ON CONFLICT (url) DO
UPDATE
SET category = EXCLUDED.category,
    criticality = EXCLUDED.criticality,
    display_name = EXCLUDED.display_name;
-- ── Global Anchor Platforms (20) ───────────────────────────
INSERT INTO targets (
        url,
        category,
        criticality,
        enabled,
        display_name
    )
VALUES (
        'https://google.com',
        'GLOBAL',
        1,
        true,
        'Google'
    ),
    (
        'https://youtube.com',
        'GLOBAL',
        1,
        true,
        'YouTube'
    ),
    (
        'https://facebook.com',
        'GLOBAL',
        1,
        true,
        'Facebook'
    ),
    (
        'https://instagram.com',
        'GLOBAL',
        1,
        true,
        'Instagram'
    ),
    (
        'https://whatsapp.com',
        'GLOBAL',
        2,
        true,
        'WhatsApp'
    ),
    (
        'https://tiktok.com',
        'GLOBAL',
        1,
        true,
        'TikTok'
    ),
    (
        'https://x.com',
        'GLOBAL',
        1,
        true,
        'X (Twitter)'
    ),
    (
        'https://linkedin.com',
        'GLOBAL',
        1,
        true,
        'LinkedIn'
    ),
    (
        'https://wikipedia.org',
        'GLOBAL',
        1,
        true,
        'Wikipedia'
    ),
    (
        'https://amazon.com',
        'GLOBAL',
        1,
        true,
        'Amazon'
    ),
    (
        'https://microsoft.com',
        'GLOBAL',
        1,
        true,
        'Microsoft'
    ),
    (
        'https://apple.com',
        'GLOBAL',
        1,
        true,
        'Apple'
    ),
    (
        'https://netflix.com',
        'GLOBAL',
        1,
        true,
        'Netflix'
    ),
    (
        'https://spotify.com',
        'GLOBAL',
        1,
        true,
        'Spotify'
    ),
    (
        'https://github.com',
        'GLOBAL',
        1,
        true,
        'GitHub'
    ),
    (
        'https://reddit.com',
        'GLOBAL',
        1,
        true,
        'Reddit'
    ),
    (
        'https://telegram.org',
        'GLOBAL',
        2,
        true,
        'Telegram'
    ),
    (
        'https://1.1.1.1',
        'ANCHOR',
        0,
        true,
        'Cloudflare DNS'
    ),
    (
        'https://cloudflare.com',
        'ANCHOR',
        0,
        true,
        'Cloudflare'
    ),
    (
        'https://8.8.8.8',
        'ANCHOR',
        0,
        true,
        'Google DNS'
    ) ON CONFLICT (url) DO
UPDATE
SET category = EXCLUDED.category,
    criticality = EXCLUDED.criticality,
    display_name = EXCLUDED.display_name;