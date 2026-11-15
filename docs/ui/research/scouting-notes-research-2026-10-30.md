# Scouting notes research — 2026-10-30

## Findings 1–5

### 1. What belongs in a report

Scouts evaluate position-specific traits and explain the projected role with evidence; a fantasy owner needs the same discipline, not the entire NFL personnel file. Keep a short overall verdict, personal grade or rank, confidence, strengths, weaknesses, role/upside, and dated observations tied to games watched. An optional player comp should say *which trait* is comparable, not imply the same career. NFL grade scales differ; do not present an owner's grade as an official scouting grade. [Scouting Academy template](https://scoutingacademy.com/wp-content/uploads/2021/09/Scouting-Report-Template-Breakdown-Pagina1.pdf); [example of NFL grading criteria](https://www.si.com/nfl/giants/onsi/draft/revealing-some-criteria-used-by-giants-nfl-teams-in-grading-draft-prospects).

For fantasy, target traits connected to opportunity and production: QB processing and rushing; RB vision, contact balance and receiving; WR separation, hands and contested catches; TE route use, hands and blocking as it affects playing time. Sample size, competition and role matter. Full medical dossiers, speculative character labels, coaching-scheme encyclopedias and dozens of mandatory numeric traits add noise. Confidence is the owner's judgment, not a calculated certainty.

### 2. Existing patterns

- **Sleeper:** its [documented custom player notes](https://support.sleeper.com/en/articles/5359526-how-to-use-custom-player-notes) put a visible note marker on a player card and carry league notes across seasons. But these are commissioner-authored, league-visible notes; edits trigger league chat, and the documented feature is mobile-only. That's the wrong privacy and desktop pattern for a personal scouting notebook. Do not infer that Sleeper lacks any other personal-note feature.
- **FantasyPros Draft Wizard:** [search results for its support page](https://support.fantasypros.com/hc/en-us/articles/115001308427-How-do-I-add-edit-and-use-Cheat-Sheet-notes-Premium) describe cheat-sheet notes, a useful pattern because the owner's note meets the draft decision. The page returned 403, so exact capabilities, availability and pricing are unverified.
- **Notion:** [database templates](https://www.notion.com/help/guides/using-database-templates) prefill a repeatable report structure. Useful for consistent reports; a separate general-purpose database does not inherently put an owner's note on TheWarRoom's draft board or preserve its player identity. No integration was tested.
- **Scouting Academy:** its linked template is a professional-report lead, not a consumer app. The PDF's text did not extract in this research; specific fields in it were not independently verified.

### 3. Live capture

Optimize for a selected player and a game context already set: click **+ NOTE**, tap a trait with a positive/negative qualifier, then **Save**. The observation gets the current capture time; game, quarter and game-clock time are optional and must never be invented from wall-clock time. One optional short sentence is available, not required. A tag alone is an observation, not an overall grade. Target under 10 seconds for the prepared-context path; this is a design target, not a measured result. Finding a player or setting up a game adds time.

### 4. Draft payoff

A small labeled note glyph on each player object and queue entry signals personal evidence without exposing text. Open the card for verdict and evidence. Show personal rank beside sourced, dated consensus ADP only if both exist and use comparable player pools and draft formats; a 1–5 grade is not an ADP rank and cannot be subtracted from it. A **My notes** filter and optional two-player comparison make the information usable under the clock. Neither should rearrange the owner's queue or tiers automatically.

### 5. Privacy

Private to the author by default, including badge, grade, text and metadata. A co-owner may benefit from a deliberately shared report, but sharing must be explicit, scoped to named co-owners, revocable and enforced in storage/API as well as UI. No league chat notification, public board marker or implied access from shared device use. Defer sharing until identity and permissions exist; revocation cannot erase copies already seen or exported.

## Recommended design

**Decision needed:** the current UI direction says player detail always goes to the inspector. Christopher's stated preference is a center card. Treat the following as a proposed exception requiring his ruling, not an already-approved replacement. Preserve the inspector until resolved.

### Center card, top to bottom

1. **Identity and controls:** name, position, current team/school, close; clearly marked **PRIVATE · MY REPORT**. Open by clicking the player name/body; dragging the position handle remains separate and must not open it. Keyboard activation and accessible close/focus return are required.
2. **Decision line:** owner's one-line verdict, optional overall 1–5 grade with named scale, confidence and personal rank; ADP with source/date/format when available. Blank means not assessed, not zero.
3. **Quick capture:** + note, position-relevant trait and positive/negative buttons, optional short text and game context, Save. Keep this visible without scrolling.
4. **Report:** strengths, weaknesses, projected fantasy role/upside and optional qualified comp. Editable but all optional.
5. **Evidence:** newest observations and games watched, with date, game/source and optional quarter/clock. A bounded internal list can scroll; the page cannot. Show capture time separately from game time.
6. **Actions:** save/edit in the card's action area, compare and close. Show saved/unsaved state and preserve edits on interruption; confirm before discarding unsaved text.

Hard borders, restrained rubber-grip texture, high contrast and no glow/blur. The card fits the viewport and does not resize the board; keyboard focus stays inside while open. No draft action is hidden inside a note or triggered by it.

### Quick-capture flow

1. Before or during viewing, choose the player and optionally set a game once; reuse that context until explicitly changed or cleared, showing it visibly to prevent misattribution.
2. Click **+ NOTE**, tap e.g. **Contact balance** then **+**, and Save. Display saved confirmation and permit undo/edit. No grade, sentence or game is mandatory.
3. Optionally add a sentence, quarter and manually entered clock; capture time is automatic. Never assume a player was watched for the whole game merely because an observation exists. An explicit watched-game entry may record full game vs clips.
4. Later, synthesize observations into the report and deliberately set or revise grade and confidence. Do not overwrite earlier observations when the verdict changes.

### Draft-time surfaces

- **Tiers and queue:** compact, accessible **N** glyph beside name when *this author* has a saved report or observation; no glyph for an empty report. Same player ID, same report. Name click opens card; drag handle still drags. Avoid color-only meaning and reveal no private text in other owners' views.
- **Filter:** **My notes only** in the board controls; applies to the board, not destructive queue changes. Clear filter restores the board. A no-results state offers Clear.
- **Decision line:** show e.g. `MY RANK 8 | ADP 12.4`, labeled independently with ADP source, date and matching format. No difference or value claim for missing, stale or mismatched data. Rank is optional and separate from the grade.
- **Compare:** select two players and open a bounded side-by-side view of verdict, grade, confidence, strengths/weaknesses and relevant evidence; explicit close returns to the board. This is a second-stage feature after capture and badges, not a prerequisite for notes.

### Minimal data fields

| Record | Fields | Purpose / constraints |
|---|---|---|
| Personal report | Stable real player ID, author/account ID, created/updated time | One report per author and player; player ID persists through prospect-to-NFL transition. Names/team are display data, not keys. |
| Evaluation | Optional verdict, grade 1–5, confidence low/medium/high, personal rank, strengths, weaknesses, projected role, qualified comp | User-authored; no inferred grade. Define 1 = avoid through 5 = top target in UI; blank is ungraded. Rank's player pool/format must be identified. |
| Observation | Report reference, capture time, optional text, trait and +/- | Multiple entries, individually editable/deletable; require text or a trait with qualifier. |
| Viewing context | Optional game/opponent/date, viewing type full game or clips when explicitly supplied, optional quarter and game clock | Separate capture timestamp from game timestamp; unknown stays unknown. Multiple observations may refer to one game. |
| Access | Author private by default; explicit named co-owner grants only if sharing is built | Enforce on all reads, search, badges and exports. Do not equate league membership with access. |

Consensus ADP is external reference data, not an owner's note: retain source, as-of date, player pool and scoring/draft format with any displayed value. Do not assume a compatible feed exists.

## What NOT to build

- No mandatory full scouting form, fake precision, automatic verdict, calculated confidence, career prediction or automatic queue/tier move.
- No arbitrary subtraction of a 1–5 grade from ADP; no unsourced or stale consensus presented as current.
- No automatic game-clock inference, required live-game lookup, covert sharing, league-wide note notifications or sensitive medical/character dossiers.
- No mobile app, video clipping, live transcription, collaboration suite or exhaustive trait taxonomy as prerequisites. Validate the desktop quick path first.

## Uncertain / unverified

- The canonical player ID's continuity from college prospect to NFL, cross-provider matching, account permissions and any ADP feed have not been verified in code. If identity is unresolved, stop and resolve it; never silently match by name.
- Under-10-second capture, card fit, keyboard/drag coexistence, filtering and comparison need a prototype and owner testing. Christopher must rule on the inspector/card conflict.
- The FantasyPros support page returned 403; search-result claims may be stale. The Scouting Academy PDF did not extract readable text. No competitor account, draft, integration or current pricing was tested.
