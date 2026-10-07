-- The Legacy NFL facts database: MFL's records as MFL states them.
--
-- Rules every table follows:
--   * Values are MFL's, unchanged: no arithmetic, no outside data, nothing estimated.
--     The only reading done is decoding MFL's own codes (pick codes, "added|dropped" lists),
--     and the decoded row keeps the original code beside it.
--   * Every row says where it came from: `src` is the file under league-archive/raw/.
--   * IDs are text: franchise "0025", player "0151" keep their leading zeros.
--   * Times are Unix seconds, as MFL gives them; `day` columns give the date MFL shows, in US
--     Eastern time. Seasons are MFL's league years.
--
-- Estimates (wins above replacement, forecasts, prices) never go in this file's tables.

PRAGMA foreign_keys = ON;

-- Every raw file loaded, so any row can be traced to the exact bytes it came from.
CREATE TABLE source_file (
  src     TEXT PRIMARY KEY,          -- path under league-archive/raw/
  bytes   INTEGER NOT NULL,
  sha256  TEXT NOT NULL
);

-- Anything the loader skipped or set aside, with how many rows and why. Empty means every
-- record in every file was loaded.
CREATE TABLE load_note (
  step    TEXT NOT NULL,
  src     TEXT,
  note    TEXT NOT NULL,
  n       INTEGER NOT NULL
);

-- ---------- the league ----------

-- How the league was set up each season (league.json).
CREATE TABLE season (
  year                INTEGER PRIMARY KEY,
  league_id           TEXT NOT NULL,     -- 2013-2015 ran under different MFL league IDs
  name                TEXT,
  start_week          INTEGER,
  end_week            INTEGER,
  last_regular_week   INTEGER,           -- MFL's own setting; playoffs follow it
  salary_cap          REAL,
  roster_size         INTEGER,
  taxi_squad          TEXT,
  injured_reserve     TEXT,
  starters            TEXT,              -- MFL's lineup requirements, as JSON
  raw                 TEXT NOT NULL,     -- the whole league.json record, as JSON
  src                 TEXT NOT NULL REFERENCES source_file
);

-- Each franchise in each season, as named that season (league.json).
CREATE TABLE franchise_season (
  year       INTEGER NOT NULL REFERENCES season,
  fid        TEXT NOT NULL,
  name       TEXT,
  abbrev     TEXT,
  division   TEXT,
  division_name   TEXT,
  conference      TEXT,
  conference_name TEXT,
  src        TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, fid)
);

-- MFL's standings, verbatim (leagueStandings.json). MFL counts points over every week it
-- scored, playoffs included; wins and losses are head-to-head regular season. Columns MFL did not
-- publish that season are NULL. `raw` keeps every column MFL sent.
CREATE TABLE standings (
  year        INTEGER NOT NULL,
  fid         TEXT NOT NULL,
  h2h_w       INTEGER, h2h_l INTEGER, h2h_t INTEGER,
  pf          REAL,                     -- points for, as MFL totals them
  pa          REAL,                     -- points against
  pp          REAL,                     -- potential points: the best lineup the team could have set
  all_play_w  INTEGER, all_play_l INTEGER, all_play_t INTEGER,
  eff         REAL,                     -- MFL's lineup efficiency, percent
  raw         TEXT NOT NULL,
  src         TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, fid)
);

-- Scoring rules each season (rules.json), one row per rule.
CREATE TABLE scoring_rule (
  year       INTEGER NOT NULL,
  positions  TEXT NOT NULL,             -- MFL's "DT|DE|LB" list
  event      TEXT NOT NULL,             -- MFL's event code, decoded in rule_code
  range      TEXT,
  points     TEXT,                      -- MFL's formula, e.g. "*0.5"
  src        TEXT NOT NULL REFERENCES source_file
);

-- MFL's dictionary of scoring codes (allRules.json).
CREATE TABLE rule_code (
  code        TEXT PRIMARY KEY,
  short       TEXT,
  description TEXT,
  src         TEXT NOT NULL REFERENCES source_file
);

-- League calendar events MFL recorded (calendar.json).
CREATE TABLE calendar_event (
  year       INTEGER NOT NULL,
  event_id   TEXT NOT NULL,
  type       TEXT,
  title      TEXT,
  start_ts   INTEGER,
  end_ts     INTEGER,
  src        TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, event_id)
);

-- ---------- players ----------

-- One row per player. Identity comes from MFL's detailed player lists (players DETAILS=1);
-- where seasons disagree, the latest season's list wins and the disagreement is kept in
-- player_conflict.
CREATE TABLE player (
  pid           TEXT PRIMARY KEY,
  name          TEXT NOT NULL,          -- MFL's "Last, First"
  position      TEXT,                   -- latest position MFL lists
  birthdate     TEXT,                   -- ISO date, from MFL's Unix time
  birthdate_ts  INTEGER,
  draft_year    INTEGER,                -- NFL draft
  draft_team    TEXT,
  draft_round   INTEGER,
  draft_pick    INTEGER,
  college       TEXT,
  height        TEXT,
  weight        TEXT,
  first_year    INTEGER,                -- first and last MFL season lists that include him
  last_year     INTEGER,
  src           TEXT NOT NULL REFERENCES source_file
);

-- Where MFL's lists disagree about a player between seasons (field, older value, newer value).
CREATE TABLE player_conflict (
  pid    TEXT NOT NULL,
  field  TEXT NOT NULL,
  year   INTEGER NOT NULL,
  value  TEXT,
  kept   TEXT
);

-- A player's position and NFL team as MFL listed them that season.
CREATE TABLE player_season (
  year      INTEGER NOT NULL,
  pid       TEXT NOT NULL,
  name      TEXT,                       -- the name MFL listed that season (some players' names change)
  position  TEXT,
  nfl_team  TEXT,
  src       TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, pid)
);

-- ---------- weeks ----------

-- Each franchise's game each week (weeklyResults). Playoff and consolation weeks appear as
-- the franchises MFL scored, with or without an opponent.
CREATE TABLE game (
  year      INTEGER NOT NULL,
  week      INTEGER NOT NULL,
  fid       TEXT NOT NULL,
  opp       TEXT,
  is_home   INTEGER,
  score     REAL,
  result    TEXT,                       -- W, L, T, or blank when MFL recorded none
  opt_pts   REAL,                       -- MFL's best possible lineup score that week
  src       TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, week, fid)
);

-- Who each franchise started and benched each week, and MFL's verdict on whether he should
-- have started (weeklyResults).
CREATE TABLE lineup (
  year          INTEGER NOT NULL,
  week          INTEGER NOT NULL,
  fid           TEXT NOT NULL,
  pid           TEXT NOT NULL,
  started       INTEGER NOT NULL,       -- 1 starter, 0 bench
  should_start  INTEGER,                -- MFL: 1 if he was in the best possible lineup
  score         REAL,
  src           TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, week, fid, pid)
);

-- Every player's score each week under the league's rules (playerScores), with MFL's
-- projection for that week (projectedScores).
CREATE TABLE player_week (
  year       INTEGER NOT NULL,
  week       INTEGER NOT NULL,
  pid        TEXT NOT NULL,
  score      REAL,
  projected  REAL,
  PRIMARY KEY (year, week, pid)
);

-- NFL injury report as MFL published it each week (injuries).
CREATE TABLE injury_week (
  year        INTEGER NOT NULL,
  week        INTEGER NOT NULL,
  pid         TEXT NOT NULL,
  status      TEXT,
  details     TEXT,
  exp_return  TEXT,
  src         TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, week, pid)
);

-- Rosters as MFL stored them each week (weekly rosters), and at season's end (rosters.json,
-- week 0 here). Before 2018 MFL's stored weekly rosters barely change within a season; use the
-- transactions for movement in those years.
CREATE TABLE roster_week (
  year             INTEGER NOT NULL,
  week             INTEGER NOT NULL,     -- 0: MFL's end-of-season roster
  fid              TEXT NOT NULL,
  pid              TEXT NOT NULL,
  status           TEXT,                 -- ROSTER, TAXI_SQUAD, INJURED_RESERVE
  salary           REAL,
  contract_year    TEXT,
  contract_status  TEXT,
  contract_info    TEXT,
  src              TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, week, fid, pid)
);

-- Every contract MFL held at the time of the pull, per season (salaries.json).
CREATE TABLE contract (
  year             INTEGER NOT NULL,
  pid              TEXT NOT NULL,
  salary           REAL,
  contract_year    TEXT,
  contract_status  TEXT,
  contract_info    TEXT,
  src              TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, pid)
);

-- Cap charges (salaryAdjustments.json): dead money from cuts, buyouts, commissioner moves.
-- Each season's file repeats earlier cuts that still charge it.
CREATE TABLE cap_charge (
  year         INTEGER NOT NULL,
  adj_id       TEXT NOT NULL,
  fid          TEXT,
  amount       REAL,
  description  TEXT,
  ts           INTEGER,
  src          TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, adj_id)
);

-- The NFL games each week (nflSchedule), one row per team.
CREATE TABLE nfl_game (
  year      INTEGER NOT NULL,
  week      INTEGER NOT NULL,
  team      TEXT NOT NULL,
  opp       TEXT,
  is_home   INTEGER,
  score     INTEGER,
  spread    REAL,
  kickoff   INTEGER,
  src       TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, week, team)
);

-- Fantasy points each NFL team allowed by position over the season (pointsAllowed.json).
CREATE TABLE points_allowed (
  year      INTEGER NOT NULL,
  nfl_team  TEXT NOT NULL,
  position  TEXT NOT NULL,
  points    REAL,
  src       TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, nfl_team, position)
);

-- ---------- picks and the draft ----------

-- Every rookie-draft selection (draftResults.json). pid is NULL where MFL marks the pick
-- skipped or forfeited ("----").
CREATE TABLE draft_pick (
  year      INTEGER NOT NULL,
  round     INTEGER NOT NULL,
  pick      INTEGER NOT NULL,
  fid       TEXT,                       -- who made the pick
  pid       TEXT,
  ts        INTEGER,
  day       TEXT,                       -- the date MFL shows (US Eastern)
  comments  TEXT,                       -- MFL's note, e.g. "[Pick traded from ...]"
  src       TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, round, pick)
);

-- Who owned which future picks at the time of each season's pull (futureDraftPicks.json),
-- and the picks listed as tradable assets (assets.json).
CREATE TABLE pick_owned (
  snapshot_year  INTEGER NOT NULL,
  source         TEXT NOT NULL,         -- futureDraftPicks or assets
  fid            TEXT NOT NULL,         -- owner
  pick_year      INTEGER,
  round          INTEGER,
  slot           INTEGER,               -- only for current-year picks
  orig_fid       TEXT,                  -- the team the pick originally belonged to
  code           TEXT,                  -- MFL's code, where given
  description    TEXT,
  src            TEXT NOT NULL REFERENCES source_file
);

-- ---------- transactions ----------

-- Every transaction MFL logged (transactions.json), one row each, all types: trades, offers,
-- rejections, revokes, accepts, expiries, free-agent moves, waivers, IR, taxi, roster loads.
CREATE TABLE txn (
  txn_id      INTEGER PRIMARY KEY,      -- stable order: season, then position in MFL's file
  year        INTEGER NOT NULL,
  ts          INTEGER,
  day         TEXT,                     -- the date MFL shows (US Eastern)
  type        TEXT NOT NULL,
  fid         TEXT,                     -- the franchise acting (trades: the first side)
  fid2        TEXT,                     -- trades and offers: the second side
  by_commish  INTEGER NOT NULL DEFAULT 0,
  expires     INTEGER,
  comments    TEXT,
  raw         TEXT NOT NULL,            -- MFL's record, as JSON
  src         TEXT NOT NULL REFERENCES source_file,
  src_idx     INTEGER NOT NULL          -- position in MFL's list
);

-- Each asset each transaction moves, decoded from MFL's lists. `code` is MFL's token as given.
--   role:  sent      a trade or offer side gave it (from_fid -> to_fid)
--          add/drop  joined or left a roster (free agent, waiver, roster load)
--          ir_on/ir_off, taxi_on/taxi_off   roster status changes
--   kind:  player | pick
-- Pick codes: FP_<orig>_<year>_<round> is a future pick; DP_<r>_<s> is a pick in the current
-- season's draft counted from zero (DP_3_20 is round 4, pick 21).
CREATE TABLE txn_asset (
  txn_id     INTEGER NOT NULL REFERENCES txn,
  role       TEXT NOT NULL,
  from_fid   TEXT,
  to_fid     TEXT,
  kind       TEXT NOT NULL,
  pid        TEXT,
  pick_year  INTEGER,
  pick_round INTEGER,
  pick_slot  INTEGER,
  pick_orig  TEXT,
  code       TEXT NOT NULL
);

-- What owners advertised (tradeBait.json).
CREATE TABLE trade_bait (
  year             INTEGER NOT NULL,
  fid              TEXT NOT NULL,
  ts               INTEGER,
  will_give_up     TEXT,                -- MFL's asset codes
  in_exchange_for  TEXT,                -- the owner's words
  src              TEXT NOT NULL REFERENCES source_file
);

-- ---------- playoffs ----------

CREATE TABLE playoff_bracket (
  year        INTEGER NOT NULL,
  bracket_id  TEXT NOT NULL,
  name        TEXT,
  winner_title TEXT,
  start_week  INTEGER,
  teams       INTEGER,
  src         TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, bracket_id)
);

CREATE TABLE playoff_game (
  year        INTEGER NOT NULL,
  bracket_id  TEXT NOT NULL,
  week        INTEGER,
  game_id     TEXT,
  fid         TEXT,
  side        TEXT,                     -- home or away
  points      REAL,
  seed        TEXT,
  src         TEXT NOT NULL REFERENCES source_file
);

-- ---------- all of MFL (not just this league) ----------

-- MFL-wide average draft position and auction value each season: how the wider market
-- valued players. Not this league's data, but MFL's.
CREATE TABLE mfl_adp (
  year       INTEGER NOT NULL,
  pid        TEXT NOT NULL,
  rank       INTEGER,
  avg_pick   REAL,
  min_pick   INTEGER,
  max_pick   INTEGER,
  drafts     INTEGER,
  src        TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, pid)
);

CREATE TABLE mfl_aav (
  year       INTEGER NOT NULL,
  pid        TEXT NOT NULL,
  rank       INTEGER,
  avg_value  REAL,
  min_value  REAL,
  max_value  REAL,
  src        TEXT NOT NULL REFERENCES source_file,
  PRIMARY KEY (year, pid)
);

CREATE INDEX txn_year_type ON txn(year, type);
CREATE INDEX txn_asset_pid ON txn_asset(pid);
CREATE INDEX txn_asset_txn ON txn_asset(txn_id);
CREATE INDEX lineup_pid ON lineup(pid);
CREATE INDEX roster_pid ON roster_week(pid);
CREATE INDEX player_week_pid ON player_week(pid);
