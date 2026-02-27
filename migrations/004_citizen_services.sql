-- ============================================================
-- Sentinel V2: Migration 004 — Add citizen-facing digital services
-- These are government/public services that Azerbaijani citizens
-- use from ABROAD (diaspora, travelers). Must be probed from all
-- vantage points, not just node-az.
-- ============================================================
-- Step 1: Add GOV_SERVICE category for citizen-facing digital services
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
-- Step 2: Seed citizen-facing digital services
-- These are critical for Azerbaijani citizens abroad
INSERT INTO targets (
        url,
        category,
        criticality,
        enabled,
        display_name
    )
VALUES -- ── Core Digital Government Platforms ─────────────────────
    (
        'https://my.gov.az',
        'GOV_SERVICE',
        10,
        true,
        'myGov (Rəqəmsal Xidmətlər)'
    ),
    (
        'https://mygovid.gov.az',
        'GOV_SERVICE',
        10,
        true,
        'myGov ID (Rəqəmsal Şəxsiyyət)'
    ),
    (
        'https://asanpay.az',
        'GOV_SERVICE',
        9,
        true,
        'ASAN Pay (Ödənişlər)'
    ),
    (
        'https://sima.az',
        'GOV_SERVICE',
        9,
        true,
        'SIMA (Rəqəmsal İmza)'
    ),
    -- ── ASAN Ecosystem ───────────────────────────────────────
    (
        'https://asan.gov.az',
        'GOV_SERVICE',
        10,
        true,
        'ASAN Xidmət Portal'
    ),
    (
        'https://asanviza.gov.az',
        'GOV_SERVICE',
        8,
        true,
        'ASAN Visa'
    ),
    (
        'https://asanfinans.az',
        'GOV_SERVICE',
        8,
        true,
        'ASAN Finans'
    ),
    -- ── E-Government Citizen Services ────────────────────────
    (
        'https://e-gov.az',
        'GOV_SERVICE',
        10,
        true,
        'E-Government Portal'
    ),
    (
        'https://e-taxes.gov.az',
        'GOV_SERVICE',
        9,
        true,
        'E-Taxes (Vergi Portalı)'
    ),
    (
        'https://epoint.az',
        'GOV_SERVICE',
        8,
        true,
        'ePoint Ödənişlər'
    ),
    -- ── Utility Payment Portals (used from abroad) ───────────
    (
        'https://azersu.az',
        'GOV_SERVICE',
        7,
        true,
        'Azərsu (Su Ödənişi)'
    ),
    (
        'https://azerishiq.az',
        'GOV_SERVICE',
        7,
        true,
        'Azərişıq (Elektrik Ödənişi)'
    ),
    (
        'https://azeriqaz.az',
        'GOV_SERVICE',
        7,
        true,
        'Azəriqaz (Qaz Ödənişi)'
    ),
    -- ── Transport & Travel (citizens abroad booking) ─────────
    (
        'https://azal.az',
        'GOV_SERVICE',
        8,
        true,
        'AZAL (Hava Yolları)'
    ),
    (
        'https://ady.az',
        'GOV_SERVICE',
        7,
        true,
        'ADY (Dəmiryolları Bilet)'
    ) ON CONFLICT (url) DO
UPDATE
SET category = EXCLUDED.category,
    criticality = EXCLUDED.criticality,
    display_name = EXCLUDED.display_name;