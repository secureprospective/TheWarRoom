# Captured live-game fixtures

Captured 2026-10-12; personal-use facts only. Original bodies remain outside the repository. No synthetic plays. Trims retain only parser fields; no logos, news, odds, articles, videos or broadcasts.

- `espn-core-plays.json`: original SHA256 `bec3ee03c3383c866737e841c3358c131fa5a31fc5ff2afdc7a1162fd33a7a7c`; trimmed 196,228 B, SHA256 `265c5cba1493b1f9ab71a7a067cae72cdc5c3da213621211bd4958f226704994`.
- `espn-scoreboard.json`: original SHA256 `43cad65977d3631d6363a2bfbbb8166bdd86232ace1ab39c4ee4a3d069f12a2b`; trimmed 6,240 B, SHA256 `c13214e27444a744c81e2e56b8060ef70d9b68b108509c716ec2b7d984dc126c`.
- `espn-summary-401872991.json`: original SHA256 `6ed1622714cbf742de737a89d2bf4efa3010de5e31ff49dee3edb4783730305e`; trimmed 89,058 B, SHA256 `690fed0c9bf29aee46bf76054f4acf3f18d9cd199534b66a72c7353fc34b356d`.
- `espn-summary-401872994.json`: original SHA256 `361c6386ddbe842eeb8d661d15f8e496ea4e02b60c9d1e675d7c7c5ebed59fcf`; trimmed 308 B, SHA256 `a7147179f7a6b80490723b60e80207e21b2811e858690618c5dbb7a0b089577b`.
- `yahoo-scoreboard.json`: original SHA256 `8e36c54d9a1312e7da95bcc29f87310b827f83f97233eae12dcd038020635dd8`; trimmed 6,552 B, SHA256 `8d63428989791f97c30365777fc972f6745d6fe738e3de4700b519affb887603`.

Summary retains four player box categories, header and drive plays. Core retains participant athlete references and play state; references are parsed, never followed. Scoreboards retain only scores, teams and status (Yahoo also last play).
Total trimmed JSON: 298386 B.

The IPC wire fixtures in `frontend/src/app/now/live*.testdata.json` are generated from these captures by `livegame_test.go`, not additional external bodies. The crosswalk bridge in that test is an explicit synthetic CSV fixture (two MFL ids → captured ESPN participant ids); it proves id joining, not that those MFL ids are the live directory's current links. Fox bifrost was not captured and is not implemented.
