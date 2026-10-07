-- Named views: what each everyday term means, defined once, in plain words.
-- Every view reads only the fact tables. Where a definition is a reading of MFL's data rather
-- than something MFL states, the comment says so and checks.py proves it against MFL's figures.

-- A game counts as played once MFL has a score for it. MFL fills weeks not yet played with a
-- blank score and a "T" result; those are not ties.
CREATE VIEW played_game AS
SELECT * FROM game WHERE score IS NOT NULL;

-- Regular season: the weeks in which every franchise had an opponent. Weeks 1-12 through 2020,
-- weeks 1-13 from 2021. (MFL's own "last regular week" setting is not reliable for past seasons:
-- it reads 16 for 2014.) Proved against MFL's standings by checks.py.
CREATE VIEW regular_week AS
SELECT g.year, g.week
FROM game g
GROUP BY g.year, g.week
HAVING SUM(g.opp IS NOT NULL) = (SELECT COUNT(*) FROM franchise_season f WHERE f.year = g.year);

-- All-play: each week, every team is scored against every other team's score that week.
-- MFL counts every week in which all teams scored, playoff weeks included; this view does the same.
CREATE VIEW all_play_week AS
SELECT a.year, a.week, a.fid,
       SUM(a.score > b.score) AS w, SUM(a.score < b.score) AS l, SUM(a.score = b.score) AS t
FROM game a JOIN game b ON b.year = a.year AND b.week = a.week AND b.fid <> a.fid
WHERE a.score IS NOT NULL AND b.score IS NOT NULL
GROUP BY a.year, a.week, a.fid;

-- One row per franchise per season: MFL's official figures beside the same figures counted from
-- the weekly games, so any difference is visible.
--   mfl_*       MFL's standings, unchanged (points for and against cover every week MFL scored).
--   reg_*       counted from the regular-season games.
--   all_play_*  MFL's all-play where MFL publishes it (2022 on); otherwise counted MFL's way.
CREATE VIEW team_season AS
WITH reg AS (
  SELECT g.year, g.fid,
         SUM(g.result = 'W') AS w, SUM(g.result = 'L') AS l, SUM(g.result = 'T') AS t,
         ROUND(SUM(g.score), 2) AS pf, COUNT(*) AS games
  FROM played_game g JOIN regular_week r USING (year, week)
  GROUP BY g.year, g.fid),
ap AS (
  SELECT year, fid, SUM(w) AS w, SUM(l) AS l, SUM(t) AS t FROM all_play_week GROUP BY year, fid),
tot AS (
  SELECT year, fid, ROUND(SUM(score), 2) AS pf, ROUND(SUM(opt_pts), 2) AS pp FROM played_game GROUP BY year, fid)
SELECT f.year, f.fid, f.name, f.division_name, f.conference_name,
       s.h2h_w AS mfl_w, s.h2h_l AS mfl_l, s.h2h_t AS mfl_t, s.pf AS mfl_pf, s.pa AS mfl_pa, s.pp AS mfl_pp,
       reg.w AS reg_w, reg.l AS reg_l, reg.t AS reg_t, reg.pf AS reg_pf, reg.games AS reg_games,
       tot.pf AS all_weeks_pf, tot.pp AS all_weeks_pp,
       COALESCE(s.all_play_w, ap.w) AS all_play_w, COALESCE(s.all_play_l, ap.l) AS all_play_l,
       COALESCE(s.all_play_t, ap.t) AS all_play_t,
       CASE WHEN s.all_play_w IS NOT NULL THEN 'MFL' ELSE 'counted' END AS all_play_source
FROM franchise_season f
LEFT JOIN standings s USING (year, fid)
LEFT JOIN reg USING (year, fid)
LEFT JOIN ap USING (year, fid)
LEFT JOIN tot USING (year, fid);

-- Every completed trade, one row per asset that moved, with names in words.
-- Before 2017 MFL did not record picks in trades; the trade's note may (see trade_note).
CREATE VIEW trade_asset AS
SELECT t.txn_id, t.year, t.ts, t.day, t.by_commish,
       a.from_fid, ff.name AS from_name, a.to_fid, tf.name AS to_name,
       a.kind, a.code, a.pid, p.name AS player, p.position,
       a.pick_year, a.pick_round, a.pick_slot, a.pick_orig, of.name AS pick_orig_name,
       t.comments
FROM txn t
JOIN txn_asset a USING (txn_id)
LEFT JOIN franchise_season ff ON ff.year = t.year AND ff.fid = a.from_fid
LEFT JOIN franchise_season tf ON tf.year = t.year AND tf.fid = a.to_fid
LEFT JOIN franchise_season of ON of.year = t.year AND of.fid = a.pick_orig
LEFT JOIN player p ON p.pid = a.pid
WHERE t.type = 'TRADE';

-- Offers that did not become trades, and how each ended. MFL logs the offer, then its end:
-- TRADE_REJECTION, TRADE_REVOKE (withdrawn), TRADE_OFFER_EXPIRED; TRADE_ACCEPT closes an offer
-- that became a trade.
CREATE VIEW offer AS
SELECT t.txn_id, t.year, t.ts, t.day, t.type,
       t.fid AS offered_by, t.fid2 AS offered_to, t.comments, t.expires
FROM txn t
WHERE t.type IN ('TRADE_PROPOSAL', 'TRADE_ACCEPT', 'TRADE_REJECTION', 'TRADE_REVOKE', 'TRADE_OFFER_EXPIRED');

-- Every roster move outside trades: adds, drops, roster loads, IR and taxi changes.
CREATE VIEW roster_move AS
SELECT t.txn_id, t.year, t.ts, t.day, t.type, a.role,
       COALESCE(a.to_fid, a.from_fid) AS fid, a.pid, p.name AS player, p.position, t.by_commish
FROM txn t JOIN txn_asset a USING (txn_id) LEFT JOIN player p ON p.pid = a.pid
WHERE t.type IN ('FREE_AGENT', 'WAIVER', 'LOAD_ROSTERS', 'IR', 'TAXI');

-- A player's age on a given day, in years: (julianday(day) - julianday(player.birthdate)) / 365.25.
-- Birthdates are MFL's.
