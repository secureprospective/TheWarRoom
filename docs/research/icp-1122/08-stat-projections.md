# Stat-line projection sources (Haiku pass, 11-22) — triaged notes
- Sleeper undocumented api.sleeper.app/projections/nfl/{season}/{week}: free, no key, real stat lines incl. IDP (idp_tkl, sack, int, pd, qb_hit, ff), kicker FG buckets, DST. Verified live 11-22 (9,420 rows wk5). No terms found; can break.
- ESPN kona_player_info projected stats: stat-ID keyed, no official map; terms unchecked.
- FantasyPros API: documented stat-line weekly/ROS; personal non-commercial only; bars building a competing product; commercial = custom quote.
- SportsDataIO: commercial license allows projections as model input + derived outputs, attribution, no raw resale; ~$99/mo lab tier (snippet); full feeds need sales agreement.
- PFF: licensed CSV/XML incl. all IDP; price unknown; page dated 2014.
- nflverse play-by-play: free, CC-BY-SA 4.0 (credit FTN Data 2023+); no projections -> build-our-own model.
- Decision needed (money/outward): rehearsal now on Sleeper + snapshots; own model from nflverse; commercial quote (SportsDataIO) before paid launch.
