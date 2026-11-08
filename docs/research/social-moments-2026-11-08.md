# Research: automated memories, AI recaps and share loops (2026-11-08)

Done by the head brain directly at Christopher's request (11-08). The question: what have Facebook, Google, Spotify, Strava, Duolingo, Sleeper and the fantasy AI tools already learned about turning real activity into social sharing, and which free and open-source pieces can we use? Plug-in points are in `future-seams.md`. The build order is DIRECTION §5 (10-28).

## 1. Lessons already paid for by others

| Who | What happened | What we take |
|---|---|---|
| Facebook Year in Review (2014) | It auto-filled a cheerful card with a user's dead daughter. Eric Meyer called it "inadvertent algorithmic cruelty". Facebook apologised. | **Never pre-fill and push a moment.** Offer it, and the owner chooses to share. No forced upbeat tone. Note too: most engaged ≠ most joyful. |
| Facebook On This Day (research, 2017) | Users sorted memories into themes and wanted control. Some post types should never resurface. Recent memories interested people as well as old ones. | Moment types come from a fixed list, and each owner can switch any type off. Recent moments ("this week") matter as much as anniversaries. |
| Google Photos Memories | Lets you hide people and dates. Google's own research says people need "the final say", and that models "won't always get it right". | **Per-owner controls:** hide moments involving a named owner (a rival, an ex, a family member), mute a season, opt out entirely. |
| Spotify Wrapped | Designed from the share target inward: bold, personal, flex-worthy, "feels like a gift, not a dashboard". It's also an annual event. | Cards look like a trophy, not a stats table. A **season-end Wrapped** is the natural first big moment. |
| Strava Year in Sport / kudos loop | Do it → feed → kudos → rankings → more. The individual loop feeds a group "trend report". | Two loops: personal cards, plus a **league-wide recap** built from the same events (the Tuesday show). |
| Duolingo streaks | The streak is in the core loop. Milestone screens (100/365 days) are built for screenshots, and the company calls it its biggest growth lever. | Streaks and milestones are the cheapest, most reliable moments: rival win streaks, years in the league, titles. |
| Sleeper | Grew on chat on almost every screen, replacing group texts. Game day runs 20–35 min per user (unverified figures). | Chat is the stickiness layer. That confirms DIRECTION §5 item 5, with moments flowing into chat. |
| ESPN (IBM watsonx) | Bulleted AI player insights, not social recaps. | The big platforms haven't done league-personal moments, so the gap is open. |
| Yahoo | AI game breakdowns and draft/matchup recaps, paid tier only. | Recaps are becoming expected. We make them free and league-specific. |
| Third-party recap tools (League Rewind $2.99/wk, The Front Office, LeagueLogs, CommishCast, SmackScript) | All bolt AI recaps or roasts onto Sleeper/ESPN, mostly paid. | Real demand exists. Ours is built in and has better data: the full ledger, contracts and history. |
| Meta 2026 (pulled the Instagram tag-to-generate feature over consent; AI personas deleted in 2025); surveys show lower trust in AI content; EU AI Act labelling from 2026-08-02 (secondary source) | AI slop and non-consensual use draw backlash. | Moments are **built from facts**. AI only words a sentence, and never invents. Never use someone's likeness. Label AI-written text. |

## 2. Our design rules this produces

1. **Facts first, AI second.** A moment is detected by a rule over the league event ledger (seam `events`): a win over a named rival, a blowout, a comeback, a streak, a lopsided trade, a title. AI may phrase the caption. It never picks a fact the ledger doesn't hold.
2. **Offer, never push.** A moment appears in the owner's own space. Sharing is one deliberate tap. Nothing auto-posts.
3. **Controls from day one:** turn off moment types, hide moments involving a named owner, opt out of appearing in others' cards (consent: a card naming Dad is about Dad too).
4. **Rare and earned.** Each type has a rarity threshold, so cards feel special. Budget a few per owner per week.
5. **Gift, not dashboard.** League-branded and bold. No real-player photos or NFL logos (standing guard).
6. **Every card carries the loop:** War Room branding and a "start your own league" link.
7. **Season-end Wrapped** is the anchor event. Weekly cards feed it.

## 3. Free/open-source pieces worth using (checked 11-08)

| Need | Pick | License | Notes |
|---|---|---|---|
| Share from the phone | **Web Share API** (`navigator.share` with `files`, `navigator.canShare`) | Web standard | Safari 14+, Chrome Android. Not in Android WebView. Fallback: download / copy link. No Facebook/SMS integrations needed. |
| Render the card image **on the device** | **modern-screenshot** or **snapdom** (DOM → PNG) | MIT | Card = a normal page component turned into an image on the phone. Zero server CPU, fits the lean server. **Leading choice.** |
| Render on the server (link previews / Open Graph images) | **ogre** (pure Go HTML/CSS → PNG, no CGo) | MIT | Young (8 stars, open issues). Use only if link previews need a server image. Pin it and test it before use. |
| Same, JS/Rust | Satori + resvg, Takumi | MIT-family | Would add Node/Rust to the server. Avoid. |
| Live delivery for chat | Our own stream seam first; **Centrifugo** (Go, MIT) if scale needs it | MIT | Matrix/Dendrite (Go, Apache-2.0) and Zulip (Python) are whole chat servers, heavy for 10–30 users. |
| Caption wording | Small open-weight model via llama.cpp/Ollama (Qwen / Gemma 4 small / Ministral, Apache-2.0 per secondary sources) **or** the owner's own AI through the CLI | Apache-2.0 (verify each model card) | Optional. Template captions work with no AI at all. CPU-only means ~5–10 tok/s for 7–8B (unverified), fine for one sentence. |

Not needed: any social-network API, any paid image or video service, AI video (dropped 10-28).

## 4. How this lands on the seams (no code now)

- `events`: moment detectors read the ledger. That needs no new log, only a `moments` table later (`storage`).
- `commands`: `moment.share` / `moment.hide` / `moment.prefs.set` arrive as normal commands, so the CLI, the AI and the screens all get them.
- `scopes`: a possible "share-only" helper token later.
- `stream`: a "new moment for you" signal; contents are read through authorized reads.

## 5. Unverified / to check before building

User-count and virality numbers above come from secondary marketing sources. The ogre maturity and LLM licences need checking against primary pages at build time. Web Share file support also needs a check on his league mates' actual phones, which they can do as part of the stress test.

## Sources
- Facebook Year in Review: https://www.businessinsider.in/a-design-flaw-in-facebooks-year-in-review-feature-caused-users-to-view-pictures-of-their-dead-relatives/articleshow/45677134.cms · https://thesocietypages.org/cyborgology/?p=19539
- Facebook memories research: https://research.facebook.com/blog/2017/09/facebook-memories-the-research-behind-the-products-that-connect-you-with-your-past/
- Google Photos: https://medium.com/people-ai-research/a-snapshot-of-ai-powered-reminiscing-in-google-photos-5a05d2f2aa46 · https://www.androidpolice.com/2021/05/31/how-to-hide-people-pets-dates-and-certain-creation-types-in-google-photos-memories/
- Spotify Wrapped: https://sproutsocial.com/insights/spotify-wrapped/ · https://nogood.io/2025/01/20/spotify-wrapped-marketing-strategy/
- Strava / Duolingo: https://mechanics-playbook-next.pages.dev/case-studies/strava/ · https://growth-dives.kit.com/posts/how-runna-turned-the-london-marathon-into-a-content-loop · https://rosetta.to/u/lennyspodcast/behind-the-product-duolingo-streaks-jackson-shuttleworth-group-pm-retention-team
- Sleeper: https://foxbusiness.com/features/fantasy-sports-platform-sleeper-giving-espn-yahoo-sports-a-run-for-its-money
- ESPN/IBM: https://betanews.com/2025/09/24/espn-fantasy-football-adds-ai-powered-weekly-insights-from-ibm-watsonx/ · Yahoo: https://gadgetbond.com/yahoo-sports-game-breakdowns-ai-nfl-coverage/
- Recap tools: https://apps.apple.com/us/app/league-rewind-fantasy-football/id6499177011 · https://www.tfofantasy.com/ · https://leaguelogs.com/ · https://commishcast.ai/ · https://smackscript.com/
- Meta backlash: https://techcrunch.com/2026/07/10/meta-removes-controversial-ai-feature-on-instagram-after-backlash/ · https://fortune.com/2025/01/04/meta-ai-accounts-bots-false-representation-social-media-facebook-instagram
- Web Share: https://web.dev/articles/web-share
- Rendering: https://github.com/macawls/ogre · https://classic.yarnpkg.com/en/package/modern-screenshot · https://www.jsdelivr.com/package/npm/@zumer/snapdom
- Chat: https://dev.to/amirtaherkhani/global-open-source-chat-platform-evaluation-469b
- Small LLMs: https://www.morphllm.com/open-source-llm
