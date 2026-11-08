# Future platform direction: AI-first seams, social moments, tiny models (2026-11-08)

**Read this first** in any session about social features, chat, moments, AI helpers or server growth. It records everything decided and researched on 11-08 between compact #2 and Tom's 12:20 CT reset. **This is documentation only; nothing here is built, measured or scheduled.**

## 1. Christopher's direction (11-08, binding)

1. **The shape of the product:** TWR is **AI first, CLI first, with a GUI overlay** so non-technical people can use it. Most future work will be AI-driven.
2. **Keep the code malleable:** build the server with open, **labelled seams** for future builds (moments, shares, invites, chat, new AI powers). Don't be rigid. **Security stays top notch.**
3. **The past failure to avoid:** unlabelled stubs were missed when the real feature was later built. So every seam is labelled, registered and gated (§2).
4. **Fantasy football is a network** of friends, family and workmates. Growth comes from that community, not marketing (10-28 ruling; DIRECTION §5; `docs/RESUME-2026-11-08.md` §12). His 10–30 league mates are the first real community.
5. **Tiny models are prospecting only:** no measuring or testing now. Keep sessions focused: this is research and documentation for later.
6. **Open question:** "leave that out of the picture": does it mean the social features themselves are out for now (seams only)? Asked, not yet answered.

## 2. What was built today (the only code change)

- `docs/architecture/future-seams.md`: the registry of five seams and their rules.

  | Seam | Where | What plugs in |
  |---|---|---|
  | `commands` | `internal/commandcatalog/workflow_commands.go` | new actions, derived, never hand-listed |
  | `scopes` | `internal/accounts/tokens.go` | new token powers (prefix families) |
  | `events` | `internal/store/history/events.go` | moments, shows and feeds read the one ledger |
  | `stream` | `internal/serverhttp/invalidations.go` | "changed for you" pings only, never contents |
  | `storage` | `internal/db/migrations.go` | feature-owned tables via migrations |

- `FUTURE-SEAM(<name>):` comment markers at each seam. Find them with `grep -rn "FUTURE-SEAM(" internal`.
- Gate `internal/seams` → `TestFutureSeamsAreRegistered`. It fails if a marker and the registry disagree; a planted mutant was caught in both directions.
- Pointers in `CLAUDE.md` (Key documents) and `SYSTEM_MAP.md` (step 6).
- Commits on `session/twr-lab-templates` (Claude-OS): `94ac1e66` seams, `ce3eb489` social research, `68b6689c` tiny models research, plus this doc.

## 3. Research: social moments (detail in `docs/research/social-moments-2026-11-08.md`)

Lessons from Facebook Year in Review / On This Day, Google Photos, Spotify Wrapped, Strava, Duolingo, Sleeper, ESPN/IBM, Yahoo, paid recap add-ons, and the 2026 Meta backlash. They become **design rules**:
1. Facts first, AI second (detect from the ledger; AI only phrases).
2. Offer, never push; one deliberate tap to share.
3. Controls from day one (moment types, hide a named owner, opt out of appearing in cards).
4. Rare and earned.
5. A gift, not a dashboard; no player photos or NFL logos.
6. Every card carries the "start your own league" loop.
7. A season-end Wrapped anchors the year.

**Open-source pieces:**
- **Sharing:** the Web Share API.
- **Cards:** rendered on the phone with modern-screenshot or snapdom (MIT). The server-side fallback is ogre (Go, MIT, young).
- **Chat transport:** our stream first; Centrifugo (MIT) if scale needs it.
- **Captions:** templates first; a small open model is optional.

**The market gap:** nobody does free, built-in, league-personal moments.

## 4. Research: tiny specialised models (detail and full maker catalog in `docs/research/tiny-models-2026-11-08.md`)

- **Makers:** Liquid AI (most likely the "Desert Alpha" he remembered; to confirm), Google Gemma, IBM Granite, Alibaba Qwen, Hugging Face SmolLM, Meta, Pleias, Katanemo, Salesforce, GLiNER, Kokoro, Distil Labs (platform).
- **Fits in TWR:**
  - plain words → catalog command preview (FunctionGemma 270M / LFM2.5-350M);
  - messy text extraction (MFL text, dead money);
  - semantic search over rules, ledger and chat (EmbeddingGemma);
  - chat moderation flags (Granite Guardian HAP);
  - moment captions;
  - audio Tuesday show (Kokoro).
- **Control pattern:** a tiny model returns a structured candidate plus a confidence score. Low confidence, or anything that changes the league, goes to the head brain or a human. It is always only a preview through the catalog gates, labelled "via AI" in the ledger.
- **Licence watch:**
  - Gemma Terms and LFM ($10M revenue cap) carry restrictions.
  - The MobileLLM licence conflicts across sources.
  - Arch-Router has a custom licence.
  - Prefer Apache-2.0 makers (Qwen, Granite, SmolLM, GLiNER v2.1+, Kokoro).

## 5. Infrastructure estimate for one league (unmeasured; for planning only)

| Part | Memory | CPU | Disk |
|---|---|---|---|
| League server (as today) | 4 GB container (process ~23 MB measured) | 2 cores | ~8 GB (DB ~0.5 GB with full history) |
| AI helper, **separate container**, models load on demand | 4 GB (worst case all loaded ~3 GB; typically <1 GB) | 2–4 cores, no GPU | ~3 GB of model files |
| **Total** | ~8 GB | 4–6 cores | ~11 GB |

Why a separate helper:
- An AI fault can't take the league down.
- The helper holds no league data or keys.
- One helper can serve many leagues.
- The app works fully with it off.

## 6. When this becomes work (not now)

After the client/server stability work and the 10–30 league-mates phase: moments, then chat (DIRECTION §5 order). Before building, check every licence on the official model card, re-verify the marketing-sourced numbers, and measure the chosen models on the lab.
