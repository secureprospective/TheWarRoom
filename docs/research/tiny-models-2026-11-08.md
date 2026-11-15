# Research: tiny specialised models (<1 GB) for TheWarRoom (2026-11-08)

Prospecting only; nothing is built. The head brain researched this directly at Christopher's request. The rule: a tiny model is a **tool** behind the CLI or catalog. A head brain (Claude/GPT) or a human drives it. Its output is always a preview that passes the same role, confirmation and revision gates (seam `commands`). It never acts alone.

## Candidates (checked 11-08; verify each licence on the official model card before use)

| Job in TWR | Model | Size | Licence (as found) | Fit |
|---|---|---|---|---|
| Plain words → command ("drop X, add Y") | **FunctionGemma 270M** (Google; fine-tuned for your own tools: 58% → 85% on Google's task) · **LFM2.5-350M / 230M** (Liquid AI, tool calling) | ~270–350M params, <500 MB | Gemma Terms · LFM Open License v1.0 (free under $10M revenue) | Catalog = tool list. Fine-tune on our 93 commands. The output is a preview the human confirms. |
| Pull structure out of messy text (MFL text, dead money, trade notes, news) | **LFM2.5-350M** extraction (successor of LFM2-350M-Extract); VL-450M-Extract for screenshots | 350–450M | LFM Open v1.0 | Strong fit: we already parse MFL text by hand. |
| Search rules, ledger, chat ("when did we vote on taxi?") and match player names | **EmbeddingGemma 308M** | <200 MB RAM quantized | Gemma Terms | In-server semantic search; also de-duplicates moments. |
| Chat moderation (flag, don't block; commish reviews) | **Granite Guardian HAP 38M / 125M** (IBM) · Tiny-Toxic-Detector 2M · Llama Guard 3-1B-INT4 (440 MB) | 2M–440 MB | Apache-2.0 likely (IBM) · Llama licence | Cheap first pass for first-class chat. |
| Moment captions / short recaps | LFM2.5-350M or a small Gemma 4 / Qwen (Apache-2.0) | <1 GB quantized | varies | Optional; templates first (facts-first rule). |
| Audio Tuesday show (what CommishCast sells) | **Kokoro 82M** TTS | ~327 MB, CPU | Apache-2.0 (mirror card) | Later; moments → script → voice. |

**Company Christopher was thinking of:** most likely **Liquid AI** ("Liquid Nanos": 230M–1.2B task models for extraction, tool use and RAG). Also possible: **Distil Labs** (a platform that distils your own tiny model from ~50 examples). Confirm with him.

## How they run with our stack

- **hugot** (Go, Apache-2.0): ONNX pipelines in-process for embeddings and classification. It has a pure-Go backend (no cgo), which suits the small classifiers and embedder. The ORT backend is faster.
- **llama.cpp server** as a sidecar for the generative tiny models (tool calling, extraction, captions). One process, loaded on demand.
- Memory: each model is <500 MB, and the server CT is 4 GB. Load on demand and count against the process-RSS gate (ruling 11-08). Measure on the lab before adopting any of them.

## Head-brain control pattern

1. The tiny model does the narrow, high-volume job (parse, classify, embed, phrase).
2. It emits a structured candidate (a catalog command preview, a flag, a match) with a confidence score.
3. Low confidence or anything that changes the league goes to a head brain or the human. Everything else is still only a preview.
4. Every tiny-model output is labelled "via AI" in the ledger (existing R9 rule).

## Risks / to verify

Licences conflict across secondary sources (FunctionGemma: Gemma Terms vs claims of Apache; the LFM licence has a $10M revenue cap). Benchmarks are vendor claims. Fine-tuning needs a small labelled set from our own command traces. None is needed for the stress test or the 10–30 league-mates phase.

## Sources
- https://blog.google/technology/developers/functiongemma/ · https://ai.google.dev/gemma/docs/mobile-actions
- https://www.liquid.ai/ja/blog/introducing-liquid-nanos-frontier-grade-performance-on-everyday-devices · https://docs.liquid.ai/lfm/models/lfm2-350m-extract.md · https://venturebeat.com/ai/liquid-ais-smallest-model-yet-lfm2-5-230m-beats-models-4x-its-size-at-data-extraction-can-run-anywhere/ · https://huggingface.co/LiquidAI/LFM2.5-VL-450M-Extract
- https://www.wearedevelopers.com/videos/1654/distil-labs-small-model-training-made-simple
- https://developers.googleblog.com/en/introducing-embeddinggemma/
- https://huggingface.co/ibm-granite/granite-guardian-hap-125m · https://huggingface.co/AssistantsLab/Tiny-Toxic-Detector · https://arxiv.org/pdf/2411.17713
- https://pkg.go.dev/github.com/knights-analytics/hugot
- https://huggingface.co/nvidia/kokoro-82M-onnx-opt

## Catalog: who makes tiny models (second pass, 11-08)

Sizes are parameters; disk is roughly 0.5–1 byte per parameter when quantized. Licences are **as found** in secondary sources: confirm on the official model card before use. "⚠" means the licence restricts or conflicts.

### General small models (fine-tune or prompt for a task)

| Lab / project | Models under ~1B | Licence | Note |
|---|---|---|---|
| Alibaba Qwen | Qwen3-0.6B, Qwen3.5-0.8B (multimodal), Qwen2.5-0.5B | Apache-2.0 | Most-downloaded sub-1B family (ATOM report, Q2 2026) |
| Google Gemma | Gemma 3 270M, Gemma 3 1B, Gemma 4 edge (2B+) | Gemma Terms ⚠ (Gemma 4: Apache-2.0) | 270M is the base for FunctionGemma |
| IBM Granite | Granite 4.0 Nano 350M + 1B (hybrid-SSM and transformer, 8 models) | Apache-2.0 | Built for local/browser use |
| Hugging Face SmolLM | SmolLM2 135M / 360M / 1.7B, SmolLM3 3B | Apache-2.0 | Fully open training recipe |
| Liquid AI | LFM2.5 230M / 350M, LFM2 1.2B; "Liquid Nanos" task models | LFM Open v1.0 ⚠ (free under $10M revenue) | Most likely the company Christopher remembered |
| Meta | Llama 3.2 1B; MobileLLM-R1 950M (reasoning) | Llama licence ⚠; MobileLLM licence conflicts (Apache vs FAIR non-commercial) ⚠ | |
| Pleias | Pleias-RAG 350M / 1B (answers with inline citations, GGUF for CPU) | check card | Good for "answer from our rules with the source" |
| Others just above 1B | Mistral Ministral 3B (Apache-2.0), Microsoft Phi-4-mini 3.8B, Cohere Tiny Aya 3.35B, NVIDIA Nemotron Nano, Apple OpenELM 270M–3B, TinyLlama 1.1B | varies | Over the <1 GB line once quantized, except OpenELM-270M |

### Task-specialised tiny models

| Job | Model (maker) | Size | Licence |
|---|---|---|---|
| Function / tool calling | FunctionGemma 270M (Google); LFM2.5-230M/350M (Liquid); xLAM-2 1B (Salesforce, unverified) | 230M–1B | Gemma Terms ⚠ · LFM ⚠ · check |
| Structured extraction | LFM2.5-350M, LFM2.5-VL-450M-Extract (Liquid) | 350–450M | LFM ⚠ |
| Named things in text (players, teams, dates), zero-shot | GLiNER small v2.1 / v2.5 (urchade / gliner-community) | small (DeBERTa-v3-small backbone) | Apache-2.0 (v1 small was CC-BY-NC ⚠) |
| Embeddings / search | EmbeddingGemma 308M (Google); all-MiniLM-L6-v2 (sentence-transformers, ~22M) | <200 MB RAM | Gemma Terms ⚠ · Apache-2.0 |
| Routing (which model or tool gets the request) | Arch-Router 1.5B (Katanemo) | 1.5B | custom research licence ⚠ |
| Toxicity / chat moderation | Granite Guardian HAP 38M / 125M (IBM); Tiny-Toxic-Detector 2M; bert-tiny-toxicity 4M (ONNX); GLiGuard (Fastino) | 2M–125M | Apache-2.0 likely · check |
| Safety / prompt-injection guard | Llama Guard 3-1B-INT4 (440 MB); Prompt Guard 2 22M (Meta) | 22M–1B | Llama licence ⚠ |
| Text to speech | Kokoro 82M | ~327 MB | Apache-2.0 |
| Speech to text | Moonshine tiny (Useful Sensors); Whisper tiny/base (OpenAI) | 27M–74M | not verified this pass |
| Vision (screenshots, images) | Moondream 0.5B; LFM2.5-VL-450M | 0.5B | check |

### Platforms for making our own tiny model
- **Distil Labs:** distils a task model from about 50 examples or production traces (vendor claims).
- **Unsloth / Hugging Face TRL:** fine-tune any of the above (FunctionGemma docs use these).
- **Runtimes:** llama.cpp / Ollama (GGUF); ONNX Runtime; **hugot** (Go, Apache-2.0, pure-Go backend) for in-server classifiers and embeddings.

Extra sources: https://arxiv.org/pdf/2604.07190 · https://www.bentoml.com/blog/the-best-open-source-small-language-models · https://huggingface.co/blog/ibm-granite/granite-4-nano · https://osai-index.eu/model/smollm · https://arxiv.org/pdf/2504.18225 · https://github.com/urchade/GLiNER · https://huggingface.co/katanemo/Arch-Router-1.5B · https://arxiv.org/abs/2509.24945
