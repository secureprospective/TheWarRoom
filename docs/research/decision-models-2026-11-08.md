# Research: decision models (Jev, Clef) and making tiny models decide better (2026-11-08)

Prospecting only; nothing is built or measured. The head brain researched this directly at Christopher's request. It complements `tiny-models-2026-11-08.md` and the hub `docs/plans/FUTURE-PLATFORM-DIRECTION-2026-11-08.md`.

## 1. What a decision model is

A "System One" model. It takes **state** (text or JSON) plus **typed questions** and returns **typed answers with probabilities**. It never writes prose, so there's no output to parse and no reasoning tokens to wait for.
- Question types: yes/no; **choice** (one of the labelled options you supply); **score** (ordered levels you define).
- The output is the pick, the probability of each option, and a confidence value.

## 2. The two models

| | **Jev** (TypeSafe AI, launched ~2026-09) | **Clef / Clef-flash** (Cloudflare, 2026-10-01) |
|---|---|---|
| Access | TypeSafe API (waitlist/console; reports vary), OpenRouter, Cloudflare, Vercel | Workers AI (`@cf/cloudflare/clef`, `@cf/cloudflare/clef-flash`), REST `/ai/run`, AI Gateway |
| Weights | closed | **open, Apache-2.0** on Hugging Face (Clef on Qwen 27B, flash on Qwen3.5-9B, LoRA adapters) |
| API | state + questions | **API-compatible with Jev**: we can swap without code changes |
| Speed (vendor) | median 524 ms | Clef 209 ms (p95 239); **flash 39 ms** (p95 122) |
| Quality (vendor evals) | 67.8% on TypeSafe's 4 workflows, level with GPT-5.6 Terra, below Sol and Opus 5, at ~1/200 the cost | beat Jev 3 of 4 workflows; leads 7 of 10 decision benchmarks |
| Price | $0.042 per million input tokens, output free (~$4.20 per 100k requests of 1k tokens); may be subsidised | not stated in the launch post |
| Context | 32k–262k (reports conflict) | 64k; also images (up to 4) |
| Data | not trained on prompts; **retention unclear; zero retention for enterprise only**; US-hosted; no SOC 2 in public docs | Cloudflare says it does **not read, store or train** on requests unless you opt into fine-tuning |
| Fine-tuning | n/a | an RL fine-tuning service on your own data (AI Gateway captures a dataset); hands-on today, self-serve later; cost not stated |

**Pick for TWR: Clef-flash on Workers AI.** It's the fastest, has open weights (no lock-in), the clearest data terms, and the Jev-compatible API keeps Jev as a fallback. We already depend on Cloudflare (Tunnel, R2). Self-hosting Clef needs a GPU (9B/27B), so it's not a fit for the league box; hosted it is.

## 3. What the research says about making small models decide better

- **Cascade with calibrated confidence.** A small model answers. Low confidence escalates to a bigger model or a person. Raw LLM confidence is miscalibrated, so fit thresholds on real labelled examples: isotonic calibration (UCCI: −31% cost at equal accuracy) or conformal sets with ~500 examples per task (RouteNLP). If the prediction set holds more than one option, escalate (Conformal Cascade). The guarantees break when the data shifts, so re-fit each season.
- **Route first, cascade second.** Classify the intent upfront, and add confidence escalation only where difficulty varies.
- **Generate, then select.** The small model proposes candidates and a separate selector or verifier picks. 1.7B selectors improve best-of-N (NVIDIA). Sub-1B judges are unreliable, so the judge should be bigger than the proposer. Cross-model juries beat self-consistency: a model judging its own output shares its blind spots.
- **Escalate only on disagreement.** If the tiny model and the decision model agree, accept. If they disagree, go to the head brain or a human. This keeps the expensive model for the hard cases.
- **The teacher makes the student better.** Log every decision-model answer and every human or head-brain override. That log becomes the training set to fine-tune the tiny models (distillation), or Clef itself via Cloudflare's RL service.

## 4. The TWR pattern: "engine frames, tiny extracts, decision model chooses, human confirms"

1. **Our engine frames the options.** It computes the legal options and the values (valuation, cap, rules). Correctness #1: the decision model never invents a value or an option and never overrides the engine's numbers (engines are done; ruling 10-31).
2. **The tiny model extracts the state.** Plain words, MFL text or chat go into structured state (FunctionGemma / LFM2.5 / GLiNER, per the tiny-models doc).
3. **The decision model chooses** among the engine's options, given the owner's own posture and preferences, and returns probabilities.
4. **The cascade:**
   - Agreement plus high calibrated confidence → shown as a **suggestion**.
   - A split or low confidence → the head brain (Claude/GPT through the CLI) or the owner.
   - **Anything that changes the league is always a catalog preview the human confirms** (seam `commands`), recorded "via AI" (R9).
5. **Learn:** owner overrides plus outcomes feed calibration and later fine-tuning.

## 5. Where it would help owners and the system

| Use | Question type | Options come from |
|---|---|---|
| Start/sit and lineup tie-breaks | choice | engine's legal lineups |
| Waiver bid band | score (low/med/high/max) | engine's value and cap room |
| Trade offer: accept / counter / decline | choice | engine's trade value plus owner posture (the owner selects the posture; AI only suggests, D28) |
| Plain words → which command family | choice | the 93 catalog commands |
| "Is this moment brag-worthy?" and rarity | score | moments detector facts |
| Chat message severity (after the tiny filter) | score | moderation flags; commish decides |
| Commish jobs board: urgent? needs a ruling? | yes/no, choice | jobs board items |

## 6. Guards and open points

- **Privacy:** send minimal state (ids and numbers, not owners' personal details). A league-level setting to turn AI off. Clef's terms are better than Jev's here.
- **Cost:** pennies per league per season at Jev's price. Clef's price is unknown, so check before use.
- **Seam:** a future "AI provider" adapter would be a **new seam**. It needs Christopher's ruling and a registry row and marker in the same commit. It doesn't exist today.
- All benchmarks and latencies above are vendor-reported. The funding figure for TypeSafe ($870M) is from one source.
- Nothing is needed for the stress test or the league-mates phase.

## Sources
- https://blog.cloudflare.com/clef-decision-models/ · https://developers.cloudflare.com/changelog/post/2026-10-01-clef-workers-ai/
- https://dev.classmethod.jp/en/articles/cloudflare-clef-overview/ · https://letsdatascience.com/news/cloudflare-releases-open-weight-clef-decision-models-c52886d6
- https://www.requesty.ai/blog/typesafe-jev-explained · https://openrouter.ai/typesafe/jev-1.13 · https://www.layer3labs.io/guides/jev-pricing · https://www.marktechpost.com/2026/09/19/typesafe-ai-releases-jev/
- https://neuraltrust.ai/blog/jev-typesafe-ai-security · https://innfactory.ai/en/ai-models/typesafe-jev/ · https://www.techtimes.com/articles/328875/20261013/typesafe-ai-raises-870m-jev-decision-model-that-never-generates-text.htm
- https://arxiv.org/abs/2605.18796 (UCCI) · https://arxiv.org/html/2604.23577 (RouteNLP) · https://www.alphaxiv.org/abs/2607.25018.md (Conformal Cascade) · https://arxiv.org/html/2601.07965v1
- https://tianpan.co/blog/2025-11-03-llm-routing-model-cascades
- https://arxiv.org/pdf/2607.10139 (LLMs as a Jury) · https://wavect.io/blog/llm-as-a-verifier/
