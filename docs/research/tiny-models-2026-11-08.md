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
