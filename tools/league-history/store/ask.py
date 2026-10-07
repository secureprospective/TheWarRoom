#!/usr/bin/env python3
"""Ask the facts database. Two ways in:

    .venv/bin/python store/ask.py find "trades where a team sent cap space" [--kind trade] [--year 2015]
                                       [--team 0025] [--player 14073] [--limit 10] [--how hybrid|words|meaning]
    .venv/bin/python store/ask.py sql "SELECT * FROM team_season WHERE fid = '0025' AND year = 2024"

`find` searches the cards by exact words (FTS5) and by meaning (vectors), merges the two rankings,
and prints each hit's card ID and the fact rows it came from. Numbers in an answer must come from
those rows (`sql`), not from the card text. `sql` opens the database read-only.
"""
import argparse
import json
import re
import sqlite3
import sys
from pathlib import Path

DB = Path(__file__).resolve().parent.parent / 'data' / 'mfl.db'
RRF_K = 60          # reciprocal rank fusion constant (Cormack et al. 2009)
POOL = 100          # candidates taken from each ranking before merging


def connect(path=DB, readonly=True):
    uri = f'file:{path}?mode=ro' if readonly else str(path)
    db = sqlite3.connect(uri, uri=readonly)
    try:
        import sqlite_vec
        db.enable_load_extension(True)
        sqlite_vec.load(db)
        db.enable_load_extension(False)
    except ImportError:
        pass        # words-only search still works without the private environment
    return db


def filters(kind=None, year=None, team=None, player=None):
    where, args = [], []
    if kind:
        where.append('c.kind = ?')
        args.append(kind)
    if year:
        lo, _, hi = str(year).partition('-')
        where.append('c.year BETWEEN ? AND ?')
        args += [int(lo), int(hi or lo)]
    if team:
        where.append("(' ' || c.fids || ' ') LIKE ?")
        args.append(f'% {team} %')
    if player:
        where.append("(' ' || c.pids || ' ') LIKE ?")
        args.append(f'% {player} %')
    return (' AND ' + ' AND '.join(where)) if where else '', args


# Words that carry no meaning in a question about the league.
STOP = set('''a an and are at be by did do does for from had has have how in into is it its of on or that the
their them they this to turn turned was were what when where which who whom why with'''.split())
# The cards use MFL's short forms; questions use everyday words.
SAME = {'quarterback': 'qb', 'quarterbacks': 'qb', 'running': 'rb', 'receiver': 'wr', 'receivers': 'wr',
        'tight': 'te', 'kicker': 'pk', 'kickers': 'pk', 'linebacker': 'lb', 'linebackers': 'lb',
        'cornerback': 'cb', 'cornerbacks': 'cb', 'safeties': 's', 'first': '1st', 'second': '2nd',
        'third': '3rd', 'fourth': '4th', 'fifth': '5th'}


def words(db, query, where, args, limit=POOL):
    """Exact-word ranking (BM25). Every word is optional; rarer words weigh more."""
    terms = [SAME.get(t, t) for t in re.findall(r"[\w.]+", query.lower()) if len(t) > 1 and t not in STOP]
    if not terms:
        return []
    match = ' OR '.join(f'"{t}"' for t in terms)
    sql = f'''SELECT c.rowid FROM card_fts f JOIN card c ON c.rowid = f.rowid
              WHERE card_fts MATCH ? {where} ORDER BY bm25(card_fts, 2.0, 1.0) LIMIT ?'''
    return [r[0] for r in db.execute(sql, [match, *args, limit])]


_models = {}


def meaning(db, query, where, args, table='card_vec', limit=POOL):
    """Meaning ranking: cosine distance to every card that passes the filters."""
    from index import PREFIX, model, pack
    row = db.execute('SELECT model FROM vec_model WHERE tbl = ?', (table,)).fetchone()
    if not row:
        return []
    name = row[0]
    if name not in _models:
        _models[name] = model(name)
    q = pack(next(iter(_models[name].embed([PREFIX.get(name, ('', ''))[1] + query]))))
    if not where:
        # No filter: the index's own nearest-neighbour search, the same cosine distance.
        sql = f'SELECT rowid, distance FROM {table} WHERE embedding MATCH ? AND k = ?'
        return [r for r, _ in sorted(db.execute(sql, [q, limit]), key=lambda r: (r[1], r[0]))]
    sql = f'''SELECT c.rowid FROM {table} v JOIN card c ON c.rowid = v.rowid WHERE 1 = 1 {where}
              ORDER BY vec_distance_cosine(v.embedding, ?), c.rowid LIMIT ?'''
    return [r[0] for r in db.execute(sql, [*args, q, limit])]


def find(db, query, how='hybrid', limit=10, table='card_vec', **flt):
    where, args = filters(**flt)
    ranks = []
    if how in ('hybrid', 'words'):
        ranks.append(words(db, query, where, args))
    if how in ('hybrid', 'meaning'):
        ranks.append(meaning(db, query, where, args, table))
    score = {}
    for ranking in ranks:
        for i, rowid in enumerate(ranking):
            score[rowid] = score.get(rowid, 0.0) + 1.0 / (RRF_K + i + 1)
    best = sorted(score, key=lambda r: -score[r])[:limit]
    out = []
    for rowid in best:
        cid, kind, year, ref, title, body = db.execute(
            'SELECT card_id, kind, year, ref, title, body FROM card WHERE rowid = ?', (rowid,)).fetchone()
        out.append({'card': cid, 'kind': kind, 'year': year, 'ref': json.loads(ref), 'title': title, 'body': body,
                    'score': round(score[rowid], 4)})
    return out


def main():
    sys.path.insert(0, str(Path(__file__).parent))
    ap = argparse.ArgumentParser()
    sub = ap.add_subparsers(dest='cmd', required=True)
    f = sub.add_parser('find')
    f.add_argument('query')
    f.add_argument('--kind')
    f.add_argument('--year', help='a season or a range, e.g. 2015 or 2014-2016')
    f.add_argument('--team', help='franchise ID, e.g. 0025')
    f.add_argument('--player', help='MFL player ID')
    f.add_argument('--limit', type=int, default=10)
    f.add_argument('--how', choices=['hybrid', 'words', 'meaning'], default='hybrid')
    f.add_argument('--full', action='store_true', help='print whole cards')
    s = sub.add_parser('sql')
    s.add_argument('query')
    a = ap.parse_args()
    db = connect()
    if a.cmd == 'sql':
        cur = db.execute(a.query)
        cols = [d[0] for d in cur.description]
        print('\t'.join(cols))
        for row in cur:
            print('\t'.join('' if v is None else str(v) for v in row))
        return 0
    for hit in find(db, a.query, a.how, a.limit, kind=a.kind, year=a.year, team=a.team, player=a.player):
        print(f'{hit["card"]}  {hit["title"]}  (facts: {json.dumps(hit["ref"])})')
        if a.full:
            print('   ' + hit['body'].replace('\n', '\n   '))
    return 0


if __name__ == '__main__':
    sys.exit(main())
