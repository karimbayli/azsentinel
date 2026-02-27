# Sentinel V2 — How It Works

> A plain-language guide to understanding internet monitoring and what this system does

---

## Table of Contents

1. [The Problem We Solve](#the-problem-we-solve)
2. [How the Internet Works (Simplified)](#how-the-internet-works-simplified)
3. [What BGP Is and Why It Matters](#what-bgp-is-and-why-it-matters)
4. [What Sentinel V2 Does](#what-sentinel-v2-does)
5. [How We Detect an Outage](#how-we-detect-an-outage)
6. [System Architecture (Visual)](#system-architecture)
7. [Signals and Confidence Scoring](#signals-and-confidence-scoring)
8. [What We CAN and CANNOT See](#what-we-can-and-cannot-see)
9. [Real-World Scenarios](#real-world-scenarios)
10. [Glossary](#glossary)

---

## The Problem We Solve

Imagine you're in Baku and your bank's website stops loading. You ask yourself:
- Is it just me?
- Is it the bank's server?
- Is the entire country's internet down?

**You have no way to know.** Neither does anyone else — there is no public, independent system that monitors whether Azerbaijan's key internet services are reachable from the outside world.

Sentinel V2 answers this question **objectively and transparently** by checking Azerbaijan's critical websites from multiple countries simultaneously.

---

## How the Internet Works (Simplified)

Think of the internet as a global postal system:

```
Your Computer → Your ISP → Regional Exchange → International Cables → Destination Server
```

### The Key Players in Azerbaijan

| Player | Role | Analogy |
|--------|------|---------|
| **ISPs** (Delta Telecom, Bakcell, Azertelecom) | Carry your data to the world | Local post offices |
| **IXPs** (Internet Exchange Points) | Where ISPs hand off data to each other | Regional sorting centers |
| **International Links** | Undersea cables and land routes connecting AZ to the world | Highway system between cities |
| **DNS** | Translates "google.com" into an IP address (like a phone number) | Phone book |
| **BGP** | The routing protocol that tells the internet how to reach Azerbaijan | Road signs and GPS routing |

### What Can Go Wrong?

1. **A website crashes** → only that website is affected
2. **An ISP has problems** → all customers of that ISP lose access
3. **International cables are cut** → the entire country may lose connectivity
4. **DNS is blocked or broken** → websites can't be found, even if they're running
5. **BGP routes are withdrawn** → the internet "forgets" how to reach Azerbaijan

Sentinel V2 detects scenarios 1-5 from the **outside looking in**.

---

## What BGP Is and Why It Matters

### The Post Office Analogy

BGP (Border Gateway Protocol) is like the system of road signs that tells mail trucks how to deliver your letter from New York to Baku.

Every ISP in the world **announces** its address to its neighbors:

> "I am AS29049 (Delta Telecom). I know how to deliver mail to these neighborhoods: 5.134.0.0/16, 94.20.0.0/15, ..."

These announcements propagate globally — within minutes, every ISP on earth knows how to reach Delta Telecom's customers.

### What a "Withdrawal" Means

If Delta Telecom **withdraws** its announcement:

> "I NO LONGER know how to deliver to 5.134.0.0/16"

...then the rest of the internet removes that address from its maps. Traffic destined for those IP addresses has **nowhere to go**. It's like removing a city from every GPS map in the world simultaneously.

### Why We Monitor This

BGP withdrawals are often the **earliest** signal of a major outage. They happen at the infrastructure level, minutes before users start noticing problems. By watching BGP, Sentinel V2 can detect outages sometimes before they're even reported.

**Where we get BGP data:** RIPE NCC (the organization that manages internet addresses for Europe/Middle East) operates **26 monitoring stations** around the world called Route Collectors. They listen to BGP announcements 24/7. We subscribe to their real-time data feed.

---

## What Sentinel V2 Does

### The Core Idea

We place **watchers** (probe nodes) in different countries. Each watcher tries to connect to Azerbaijan's key websites every 60 seconds. If multiple watchers from different countries can't reach a site, we know the problem is on Azerbaijan's side, not the watcher's.

### What We Monitor

| Category | Examples | Why |
|----------|----------|-----|
| **Government** | e-gov.az, president.az | Critical public services |
| **Banking** | ibar.az, kapitalbank.az | Financial infrastructure |
| **ISPs** | bakcell.az, azercell.az | Internet providers themselves |
| **Media** | azertag.az, report.az | Information access |

### What We Check (Four Layers)

Every 60 seconds, each probe node runs this test against every target:

```
Step 1: DNS Resolution
   "What is the IP address of e-gov.az?"
   → Measures: Can we find the address? How long did it take?
   → If this fails: DNS is broken (service may still be running, but nobody can find it)

Step 2: TCP Connection
   "Can we establish a basic network connection to that IP address?"
   → Measures: Is the server accepting connections? How long did it take?
   → If this fails: Server is down OR the network path is broken

Step 3: TLS Handshake
   "Can we establish a secure (encrypted) connection?"
   → Measures: Is the security certificate valid? Is it expired?
   → If this fails: Security configuration problem

Step 4: HTTP Request
   "Can we actually load the webpage?"
   → Measures: What status code? How fast was the first byte?
   → If this fails: The application/website itself has a problem
```

This layered approach lets us pinpoint **where** the failure is occurring.

---

## System Architecture

```
                    ┌─────────────── THE INTERNET ───────────────┐
                    │                                             │

  🇳🇱 Amsterdam      🇺🇸 New York       🇸🇬 Singapore      🇩🇪 Frankfurt
  ┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
  │  Probe   │     │  Probe   │     │  Probe   │     │  Probe   │
  │  Node    │     │  Node    │     │  Node    │     │  Node    │
  └────┬─────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
       │                │                │                │
       │     Results    │    sent every  │    60 seconds  │
       │    ──────────────────────────────────────────►   │
       │                │                │                │
       │                ▼                │                │
       │     ┌─────────────────────┐     │                │
       └────►│   Central Server    │◄────┘                │
             │   (Baku, Azerbaijan)│◄──────────────────────┘
             │                     │
             │  ┌───────────────┐  │
             │  │Correlation    │  │      ┌────────────────┐
             │  │Engine         │◄─┼──────┤ RIPE RIS Live  │
             │  │               │  │      │ (BGP data)     │
             │  │ Node + BGP +  │  │      └────────────────┘
             │  │ Social        │  │
             │  │ = Confidence  │  │      ┌────────────────┐
             │  └───────┬───────┘  │◄─────┤ Telegram       │
             │          │          │      │ (social data)  │
             │          ▼          │      └────────────────┘
             │  ┌───────────────┐  │
             │  │ Alert System  │──┼─────► Telegram Alerts
             │  └───────────────┘  │
             │                     │
             │  ┌───────────────┐  │
             │  │ Status Page   │──┼─────► Public Dashboard
             │  └───────────────┘  │
             │                     │
             │  ┌───────────────┐  │
             │  │ Grafana       │──┼─────► Detailed Metrics
             │  └───────────────┘  │
             └─────────────────────┘
```

### What Happens If Azerbaijan Goes Offline?

This is the critical design challenge: **the Central Server is inside Azerbaijan.** If the country's internet is cut, the probe nodes can't send their results to the server.

**Solution:** Each probe node has a **local storage buffer** (like a black box recorder on an airplane). When it can't reach the Central Server:

1. Results are saved locally on the probe's disk
2. The probe keeps trying to reconnect with increasing wait times (5s → 10s → 30s → 2min)
3. When connectivity is restored, all buffered results are sent in chronological order

No data is ever lost — even during a complete national outage.

---

## Signals and Confidence Scoring

### The Three Signals

Sentinel V2 doesn't rely on a single data source. It combines **three independent signals** to reach a conclusion:

| Signal | Weight | What It Detects | Source |
|--------|--------|-----------------|--------|
| 🌐 **Multi-Node Probe Failure** | 50% | Multiple countries can't reach the target | Our own probes |
| 📡 **BGP Route Withdrawal** | 30% | Internet routing to Azerbaijan is disrupted | RIPE NCC data |
| 💬 **Social Media Spike** | 20% | People are reporting outages on Telegram | Telegram channels |

### How We Calculate Confidence

```
Confidence = (Node Signal × 0.5) + (BGP Signal × 0.3) + (Social Signal × 0.2)
```

**Example scenarios:**

| Scenario | Node | BGP | Social | Confidence | Status |
|----------|------|-----|--------|------------|--------|
| Everything fine | 0 | 0 | 0 | **0.0** | 🟢 HEALTHY |
| Social chatter only | 0 | 0 | 0.1 | **0.1** | 🟢 HEALTHY |
| Probes failing + social | 0.5 | 0 | 0.2 | **0.7** | 🟠 PARTIAL_OUTAGE |
| Probes + BGP + social | 0.5 | 0.3 | 0.2 | **1.0** | 🔴 MAJOR_OUTAGE |

### Status Levels

| Status | Confidence | Meaning |
|--------|-----------|---------|
| 🟢 **HEALTHY** | 0.0 – 0.3 | Target is reachable from all locations |
| 🟡 **DEGRADED** | 0.3 – 0.5 | Some issues detected, may be slow or intermittent |
| 🟠 **PARTIAL_OUTAGE** | 0.5 – 0.8 | Significant reachability problems |
| 🔴 **MAJOR_OUTAGE** | 0.8 – 1.0 | Target is effectively unreachable from multiple countries |

### Safety Measures (Preventing False Alarms)

We have several safeguards to avoid false positives:

- **Minimum 2 probe nodes must agree** before declaring a node-level failure
- **BGP requires 3+ prefix withdrawals** (a single route flap is ignored)
- **Social signal is halved** if probe data doesn't confirm the chatter
- **Anchor targets** (Google, Cloudflare) are checked first — if a probe can't reach *them*, the probe itself is unreliable and its results are discounted

---

## What We CAN and CANNOT See

### ✅ What We CAN Measure

- Whether Azerbaijan's websites are reachable from Europe, USA, Asia
- How long it takes to connect (latency)
- Whether BGP routes to Azerbaijan are being withdrawn globally
- Whether DNS for Azerbaijan services is resolving correctly
- Whether TLS certificates are valid and not expired
- Social media signals about service disruptions

### ❌ What We CANNOT Measure

- **Internal ISP performance** — we can't see what happens inside Bakcell's or Azertelecom's internal network
- **Speed for users inside Azerbaijan** — we measure from outside, not from Baku
- **Content blocking** — if a site is selectively blocked for certain users, our probes might still see it as "up"
- **Mobile network quality** — we only test via fixed internet connections
- **Internal peering** — we can't see traffic exchange between ISPs inside Azerbaijan

### Why This Matters

We designed the system to be **honest about its limitations**. Our `/methodology` page publicly explains exactly what we measure, how we measure it, and what we can't measure. This transparency is critical for credibility.

---

## Real-World Scenarios

### Scenario 1: A Single Website Goes Down

**What happens:** e-gov.az crashes due to a server error  
**What Sentinel V2 sees:**
- ❌ All 4 probes report HTTP 500 for e-gov.az
- ✅ All other targets are healthy
- ✅ No BGP changes
- ✅ No social spike

**Result:** e-gov.az → 🟠 PARTIAL_OUTAGE (only this target affected)

### Scenario 2: Submarine Cable Cut

**What happens:** The fiber optic cable between AZ and Europe is damaged  
**What Sentinel V2 sees:**
- ❌ ALL targets fail from EU/US probes
- ❌ BGP: 20+ prefix withdrawals across multiple ASNs
- ❌ Telegram: spike in "internet yoxdur" mentions
- ⚠️ AZ local probe still works (domestic routing may be unaffected)

**Result:** ALL targets → 🔴 MAJOR_OUTAGE (confidence 1.0)

### Scenario 3: DNS Poisoning / Blocking

**What happens:** DNS resolution for certain sites is interfered with  
**What Sentinel V2 sees:**
- ❌ DNS resolution fails (new `DNS_RESOLVE` error type)
- ✅ TCP would succeed if we could resolve the address
- No BGP changes (routing is fine, DNS is the problem)

**Result:** Affected targets → 🟡 DEGRADED with clear indication of DNS-layer failure

### Scenario 4: A Probe Node Has Problems

**What happens:** Our Singapore server has network issues  
**What Sentinel V2 sees:**
- ❌ Singapore probe can't reach ANYTHING (including Google, Cloudflare)
- ✅ Amsterdam, New York, Frankfurt probes see everything as healthy

**Result:** Singapore probe marked as **unreliable** — its results are automatically discounted. No false alarm triggered.

---

## Glossary

| Term | Simple Definition |
|------|-------------------|
| **ASN** | Autonomous System Number — a unique ID for each network operator (like a postal code for ISPs) |
| **BGP** | Border Gateway Protocol — the system ISPs use to tell each other how to route traffic |
| **DNS** | Domain Name System — translates website names (google.com) into IP addresses (142.250.80.46) |
| **HMAC** | A method for signing messages to prove they haven't been tampered with |
| **ISP** | Internet Service Provider — the company that gives you internet access |
| **Latency** | The time it takes for data to travel from point A to point B (measured in milliseconds) |
| **Prefix** | A block of IP addresses owned by an ISP (like a range of house numbers on a street) |
| **Probe Node** | A server in another country that checks if Azerbaijan's internet is reachable |
| **RIPE NCC** | The organization that manages internet addresses for Europe and the Middle East |
| **Route Collector** | A server run by RIPE NCC that listens to BGP announcements from ISPs |
| **TCP** | Transmission Control Protocol — the basic method computers use to establish connections |
| **TLS** | Transport Layer Security — encryption that protects data in transit (the "S" in HTTPS) |
| **TTFB** | Time to First Byte — how long until the server starts sending data back |
| **Withdrawal** | When an ISP tells the internet it can no longer deliver traffic to certain addresses |

---

## How the Business of Internet Monitoring Works

### Who Needs This?

| Audience | Why They Care |
|----------|---------------|
| **Government regulators** | Need objective data on internet quality and availability |
| **ISPs themselves** | Want to prove their SLA compliance and detect issues early |
| **Businesses** | Need to know if their customers can reach their services |
| **Journalists & researchers** | Need factual data about internet freedom and censorship |
| **International organizations** | Track digital rights and internet accessibility globally |
| **Investors** | Assess digital infrastructure risk in a country |

### How Existing Companies Do This

| Company | How It Works | Revenue Model |
|---------|-------------|---------------|
| **RIPE Atlas** | 11,000+ volunteer-hosted probes worldwide | Free (funded by RIPE NCC membership) |
| **ThousandEyes** (Cisco) | Cloud-based monitoring service | SaaS subscription ($$$) |
| **Kentik** | Network observability platform | SaaS subscription ($$$$) |
| **IODA** (Georgia Tech) | Academic internet outage detection | Free (research) |
| **OONI** | Volunteer-run censorship detection | Free (open source) |
| **Cloudflare Radar** | Leverages their global CDN for visibility | Free (marketing for Cloudflare) |

### Where Sentinel V2 Fits

Sentinel V2 is purpose-built for **Azerbaijan's digital ecosystem** specifically. Unlike global platforms:
- **Focused**: Monitors AZ-specific targets (gov, banking, ISPs, media)
- **Transparent**: Publishes methodology, admits limitations
- **Independent**: Not funded by or affiliated with any ISP
- **Resilient**: Works even if Azerbaijan's internet goes down
- **Open**: Public status page and API for anyone to query

---

*Document version: 1.0 — Generated 2026-02-26*
