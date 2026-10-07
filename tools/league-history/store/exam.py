#!/usr/bin/env python3
"""The search exam: does `find` bring back the right card?

Questions are written in plain English from the facts, with a fixed random seed, so the right
card is known for each one. No question contains an ID. Two kinds:

  * one-answer questions (a draft pick, a trade, a player season, a team season, what a traded pick
    became): scored by whether the right card is first, and whether it is in the top 10;
  * many-answer questions (every trade that moved cap space, every offer the Cardinals had
    rejected, ...): scored by how many of the top 10 are right.

    .venv/bin/python store/exam.py                         # every search method against the built index
    .venv/bin/python store/exam.py --tables vec_bge_base vec_nomic
"""
import argparse
import random
import re
import sqlite3
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from ask import connect, find   # noqa: E402

SEED = 20261005
PER_KIND = 30


def name(raw):
    last, _, first = (raw or '').partition(', ')
    return f'{first} {last}'.strip() if first else raw


def one_answer(db):
    rnd = random.Random(SEED)
    team = {(y, f): n for y, f, n in db.execute('SELECT year, fid, name FROM franchise_season')}
    qs = []
    picks = db.execute('SELECT year, round, pick, fid FROM draft_pick WHERE year > 2013 AND pid IS NOT NULL').fetchall()
    for y, r, s, f in rnd.sample(picks, PER_KIND):
        qs.append(('draft', f'Who did the {team[(y, f)]} take with pick {r}.{s:02d} in the {y} rookie draft?',
                   f'draft_pick:{y}:{r}:{s}'))
    trades = db.execute('''SELECT t.txn_id, t.year, a.from_fid, a.to_fid, p.name FROM txn t
                           JOIN txn_asset a USING (txn_id) JOIN player p ON p.pid = a.pid
                           WHERE t.type = 'TRADE' AND a.kind = 'player' ''').fetchall()
    # a player traded once between the same two teams that year, so the answer is unique
    counts = {}
    for tid, y, frm, to, n in trades:
        counts[(y, frm, to, n)] = counts.get((y, frm, to, n), 0) + 1
    trades = [t for t in trades if counts[(t[1], t[2], t[3], t[4])] == 1]
    for tid, y, frm, to, n in rnd.sample(trades, PER_KIND):
        qs.append(('trade', f'When did the {team[(y, frm)]} trade {name(n)} to the {team[(y, to)]} in {y}?', f'trade:{tid}'))
    seasons = db.execute('''SELECT c.year, p.pid, p.name FROM card c JOIN player p ON p.pid = c.pids
                            WHERE c.kind = 'player_season' ''').fetchall()
    for y, pid, n in rnd.sample(seasons, PER_KIND):
        qs.append(('player_season', f'How did {name(n)} do in the {y} season?', f'player_season:{y}:{pid}'))
    ts = db.execute('SELECT year, fid, name FROM franchise_season').fetchall()
    for y, f, n in rnd.sample(ts, PER_KIND):
        qs.append(('team_season', f'What was the {n} record in {y}?', f'team_season:{y}:{f}'))
    became = db.execute('''SELECT t.year, a.pick_year, a.pick_round, a.pick_orig, d.round, d.pick FROM txn t
                           JOIN txn_asset a USING (txn_id)
                           JOIN draft_pick d ON d.year = a.pick_year AND d.round = a.pick_round
                           WHERE t.type = 'TRADE' AND a.kind = 'pick' AND a.pick_slot IS NULL AND d.pid IS NOT NULL
                             AND (d.comments IS NULL AND d.fid = a.pick_orig)''').fetchall()
    for y, py, pr, orig, r, s in rnd.sample(sorted(set(became)), PER_KIND // 2):
        ordn = {1: '1st', 2: '2nd', 3: '3rd'}.get(pr, f'{pr}th')
        qs.append(('pick_became', f'Which player did the {team.get((py, orig), orig)} {py} {ordn}-round pick turn into?',
                   f'draft_pick:{py}:{r}:{s}'))
    return qs


def many_answers(db):
    """Questions with many right cards, each defined by a test on the facts."""
    def ids(sql, *args):
        return {r[0] for r in db.execute(sql, args)}
    cap = re.compile(r'(\d+(\.\d+)?\s*(m\b|mil|million)|\bcap\b|\d+k\b)', re.I)
    cap_trades = {f'trade:{t}' for t, c in db.execute("SELECT txn_id, comments FROM txn WHERE type='TRADE' AND comments IS NOT NULL")
                  if cap.search(c)}
    return [
        ('Trades where one team also sent salary cap space', cap_trades),
        ('Trade offers the Arizona Cardinals made that were rejected',
         ids("SELECT 'offer:' || txn_id FROM txn WHERE type = 'TRADE_REJECTION' AND fid = '0025'")),
        ('Teams that won the Legacy NFL Super Bowl',
         ids("SELECT card_id FROM card WHERE kind = 'team_season' AND body LIKE '%Won %Super%'")),
        ('Rookie draft picks who never started a game for anyone',
         ids("SELECT card_id FROM card WHERE kind = 'draft_pick' AND body LIKE '%Never started%' AND year > 2013")),
        ('Players the Cardinals put on injured reserve in 2024',
         ids("SELECT card_id FROM card WHERE kind = 'player_season' AND year = 2024 AND body LIKE '%injured reserve by Arizona Cardinals%'")),
        ('Trades made at the trade deadline in midseason, week 8 or 9',
         ids("SELECT card_id FROM card WHERE kind = 'trade' AND (body LIKE '%season week 8)%' OR body LIKE '%season week 9)%')")),
        ('Quarterbacks traded for a first-round pick',
         ids('''SELECT DISTINCT 'trade:' || a.txn_id FROM txn_asset a JOIN player p ON p.pid = a.pid
                JOIN txn t USING (txn_id) WHERE t.type = 'TRADE' AND p.position = 'QB'
                AND EXISTS (SELECT 1 FROM txn_asset b WHERE b.txn_id = a.txn_id AND b.kind = 'pick'
                            AND b.pick_round = 1 AND b.from_fid = a.to_fid)''')),
        ('Kickers who were benched when they should have started',
         ids("SELECT card_id FROM card c JOIN player p ON p.pid = c.pids WHERE c.kind = 'player_season' AND p.position = 'PK' AND c.body LIKE '%should have started%'")),
    ]


def score(db, table, how):
    qs = one_answer(db)
    by_kind = {}
    for kind, q, gold in qs:
        hits = [h['card'] for h in find(db, q, how, 10, table)]
        k = by_kind.setdefault(kind, [0, 0, 0])
        k[0] += 1
        k[1] += hits[:1] == [gold]
        k[2] += gold in hits
    many = []
    for q, gold in many_answers(db):
        hits = [h['card'] for h in find(db, q, how, 10, table)]
        many.append((q, sum(h in gold for h in hits), min(10, len(gold))))
    return by_kind, many


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--tables', nargs='*')
    a = ap.parse_args()
    db = connect()
    try:
        models = dict(db.execute('SELECT tbl, model FROM vec_model'))
    except sqlite3.OperationalError:
        models = {}         # no meaning index built yet: exact words only
    tables = a.tables if a.tables is not None else sorted(models)
    runs = [('words', None)] + [(how, t) for t in tables for how in ('meaning', 'hybrid')]
    print('One-answer questions: right card first / in the top 10')
    results = {}
    for how, t in runs:
        results[(how, t)] = score(db, t or 'card_vec', how)
    kinds = list(next(iter(results.values()))[0])
    print(f'{"method":46s}' + ''.join(f'{k:>18s}' for k in kinds) + f'{"all":>14s}')
    for (how, t), (by_kind, _) in results.items():
        label = how if not t else f'{how} ({models.get(t, t)})'
        tot = [sum(v[i] for v in by_kind.values()) for i in range(3)]
        print(f'{label:46s}' + ''.join(f'{v[1]:>7d}/{v[2]:<3d}of {v[0]:<3d}' for v in by_kind.values())
              + f'{tot[1]:>6d}/{tot[2]:<3d}of {tot[0]}')
    print('\nMany-answer questions: right cards in the top 10')
    qs = [q for q, _, _ in next(iter(results.values()))[1]]
    for i, q in enumerate(qs):
        print(f'  {q}')
        for (how, t), (_, many) in results.items():
            label = how if not t else f'{how} ({models.get(t, t)})'
            print(f'      {label:46s} {many[i][1]:>2d} of {many[i][2]}')


if __name__ == '__main__':
    sys.exit(main())
