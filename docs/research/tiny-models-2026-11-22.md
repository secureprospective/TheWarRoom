# Tiny models — refresh pass (2026-11-22)
Adds to docs/research/tiny-models-2026-11-08.md (read that first; its control rule still holds:
a tiny model is a tool behind the command catalog, output is a preview, league changes need a person).
Gathered by a Haiku research agent (no Reddit). LEADS, not verified by Claudebox: confirm licence and size
on the official card before any use. Prospecting only; nothing is built.

## New since the 11-08 list, best fits first
| Job in TWR | Candidate | Size | Licence (as reported) | Note |
|---|---|---|---|---|
| Player projections from tables (stat lines, snaps, depth) | NVIDIA Kumo-Tabular Small | ~28M (secondary) | OpenMDW-1.1 weights, Apache code | CPU; up to 60k rows/100 cols. Best new fit for T696 |
| same | Prior Labs TabPFN-3.5 | — | research-only weights | Benchmark baseline only; NOT shippable |
| Forecasting | Amazon Chronos-2 / Chronos-Bolt small 48M | 48M | Apache-2.0 | trend lines (value over weeks) |
| Today tab ranking / rerank feed items | Hindsight hs-rerank-1 | 114M (~15M transformer) | MIT | first open tiny reranker found |
| Embeddings / search | Bekko Embedding a8m/a25m; IBM granite-embedding-97m-r2; MongoDB mdbr-leaf ~23M | 8-28M active | MIT / unconfirmed | Jina v5 nano is non-commercial |
| Plain words -> command | Cactus Needle 26M (MIT); Multiverse LittleLamb 0.3B (Apache, BFCL 51 vs FunctionGemma 27, self-reported); qvac BotPsy-0.8B (Apache) | 26M-0.8B | as noted | test on our own catalog |
| Recap voice | KittenTTS v0.8 15-80M (Apache); Kyutai Pocket TTS 100M (unconfirmed) | tiny | as noted | Tuesday recap audio |
| Speech in | FUTO asr4all-s 28M (use-restricted); Phonon-1 Micro (Apache) | tiny | as noted | voice commands |
| On-phone runtime | Google LiteRT-LM (+ .js WebGPU) | runtime | unchecked | phone-side inference later |

Not found: an open tiny learning-to-rank/recommendation model. Today ranking = rules + features + small
reranker or a gradient-boosted ranker we train (LightGBM/XGBoost lambdarank are classic FOSS for this).

Sources: huggingface.co/nvidia/Kumo-Tabular · huggingface.co/Prior-Labs/tabpfn_3 · huggingface.co/autogluon/chronos-bolt-small ·
huggingface.co/hindsight-ai/hs-rerank-1 · huggingface.co/blog/hotchpotch/bekko-embedding · huggingface.co/Cactus-Compute/needle ·
huggingface.co/MultiverseComputingCAI/LittleLamb-ToolCalling · huggingface.co/qvac/BotPsy-0.8B · github.com/KittenML/KittenTTS ·
huggingface.co/kyutai/pocket-tts · huggingface.co/futo-org/asr4all-s · developers.googleblog.com/blazing-fast-on-device-genai-with-litert-lm/
