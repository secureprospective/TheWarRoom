#!/usr/bin/env python3
"""Load the readings of MFL's trade notes (store/note_readings.csv) into data/mfl.db, and check
each pick reading against MFL's draft results where the draft is on record.

The readings are interpretations, not MFL's records: they live in their own table, `note_reading`,
each marked clear, inferred, ambiguous or unreadable, and unconfirmed until Christopher confirms
them. A pick reading is checked where the pick's draft is on record (2017 on): every transfer of
that pick, MFL's recorded trades and the readings together, is replayed in date order from its
original owner. The reading agrees when its sender held the pick at that moment and the chain ends
with the team MFL says used the pick.

    python3 store/notes.py        # after store/load.py
"""
import collections
import csv
import re
import sqlite3
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
DB = HERE.parent / 'data' / 'mfl.db'
CSV = HERE / 'note_readings.csv'

DDL = '''
DROP TABLE IF EXISTS note_reading;
CREATE TABLE note_reading (
  txn_id      INTEGER NOT NULL REFERENCES txn,
  from_fid    TEXT,
  to_fid      TEXT,
  kind        TEXT NOT NULL,        -- pick, cap, player, none, unreadable
  pick_year   INTEGER,
  pick_round  INTEGER,
  pick_slot   INTEGER,
  pick_orig   TEXT,
  cap_m       REAL,                 -- cap space in $ millions per season named
  cap_seasons TEXT,                 -- space-separated seasons; blank when the note does not say
  player      TEXT,                 -- a player the note says moved who is not in MFL's record
  clarity     TEXT NOT NULL,        -- clear, inferred (from other trades), ambiguous, unreadable
  reading     TEXT NOT NULL,        -- the reading in words
  draft_check TEXT,                 -- for picks whose draft is on record: agrees, sender did not hold it,
                                    -- or chain ends with another team; blank when it cannot be checked
  confirmed   INTEGER NOT NULL DEFAULT 0   -- Christopher has confirmed this reading
);
'''


def num(v, kind=int):
    return kind(v) if v not in (None, '') else None


def draft_users(db):
    """(year, round, original owner) -> the team that used the pick. The original owner is the first
    team in MFL's "Pick traded from" note, or the team that used it when there is no note."""
    anyname = collections.defaultdict(set)
    for f, n in db.execute('SELECT fid, name FROM franchise_season'):
        anyname[n].add(f)
    out = {}
    for y, r, used, com in db.execute('SELECT year, round, fid, comments FROM draft_pick'):
        chain = re.findall(r'Pick traded from ([^.\n]+)\.', com or '')
        cand = anyname.get(chain[0].strip(), set()) if chain else {used}
        if len(cand) == 1:
            out[(y, r, next(iter(cand)))] = used
    return out


def mfl_transfers(db):
    """Pick transfers MFL recorded in trades: (year, round, original owner) -> [(ts, from, to)]."""
    slot_orig = {}
    for (y, r, o), used in draft_users(db).items():
        pass
    anyname = collections.defaultdict(set)
    for f, n in db.execute('SELECT fid, name FROM franchise_season'):
        anyname[n].add(f)
    for y, r, s, used, com in db.execute('SELECT year, round, pick, fid, comments FROM draft_pick'):
        chain = re.findall(r'Pick traded from ([^.\n]+)\.', com or '')
        cand = anyname.get(chain[0].strip(), set()) if chain else {used}
        slot_orig[(y, r, s)] = next(iter(cand)) if len(cand) == 1 else None
    out = collections.defaultdict(list)
    for ts, frm, to, y, r, s, o in db.execute('''SELECT t.ts, a.from_fid, a.to_fid, a.pick_year, a.pick_round,
            a.pick_slot, a.pick_orig FROM txn t JOIN txn_asset a USING (txn_id) WHERE t.type = 'TRADE' AND a.kind = 'pick' '''):
        orig = o or slot_orig.get((y, r, s))
        if orig:
            out[(y, r, orig)].append((ts, frm, to, 'MFL'))
    return out


def replay(start, moves):
    """Walk a pick's transfers in time order. Returns the final holder and, for each move, whether
    the sender held the pick at that moment."""
    holder, held = start, []
    for ts, frm, to, who in sorted(moves, key=lambda m: m[0]):
        held.append((who, frm == holder))
        if frm == holder:
            holder = to
    return holder, held


def review_section(db):
    """The readings Christopher should look at, as markdown lines for the review list."""
    names = {(y, f): n for y, f, n in db.execute('SELECT year, fid, name FROM franchise_season')}
    rows = db.execute('''SELECT n.txn_id, t.year, t.day, t.fid, t.fid2, t.comments, n.clarity, n.reading, n.draft_check
                          FROM note_reading n JOIN txn t USING (txn_id)
                          WHERE n.clarity <> 'clear' OR n.draft_check NOT IN ('agrees')
                          ORDER BY t.ts, n.rowid''').fetchall()
    out = ['', '## Trade notes: readings to confirm', '',
           'Before 2017 MFL kept draft picks and cap money in trades only as notes. Every note has been read '
           '(store/note_readings.csv). Below are the readings that are not plainly stated, or that do not line '
           'up with the pick histories where a draft is on record. A pick history can also fail to line up '
           'because an earlier trade moved the pick without any note.', '']
    last = None
    for tid, y, d, a, b, note, clarity, reading, check in rows:
        if tid != last:
            out += ['', f'**Trade {tid}, {d}: {names.get((y, a), a)} and {names.get((y, b), b)}**', '',
                    f'> {note}' if note else '> (no note)', '']
            last = tid
        flags = [x for x in (clarity if clarity != 'clear' else None, check if check and check != 'agrees' else None) if x]
        out.append(f'- {reading} ({"; ".join(flags)})')
    return out


def main(path=DB):
    db = sqlite3.connect(path)
    db.executescript(DDL)
    users = draft_users(db)
    mfl = mfl_transfers(db)
    ts_of = dict(db.execute("SELECT txn_id, ts FROM txn WHERE type = 'TRADE'"))
    readings = list(csv.DictReader(CSV.open()))
    noted = collections.defaultdict(list)
    for i, r in enumerate(readings):
        if r['kind'] == 'pick' and r['pick_orig'] and r['pick_year'] and r['from_fid']:
            key = (int(r['pick_year']), int(r['pick_round']), r['pick_orig'])
            noted[key].append((ts_of[int(r['txn_id'])], r['from_fid'], r['to_fid'], i))
    verdict = {}
    for key, moves in noted.items():
        if key not in users:
            continue        # no draft on record for this pick
        used = users[key]
        end_mfl, _ = replay(key[2], mfl.get(key, []))
        end_all, held = replay(key[2], mfl.get(key, []) + moves)
        for (who, ok) in held:
            if who != 'MFL':
                verdict[who] = ('agrees' if ok and end_all == used else
                                'sender did not hold it' if not ok else
                                'chain ends with another team')
    rows, tally = [], collections.Counter()
    for i, r in enumerate(readings):
        check = verdict.get(i)
        y, rnd = num(r['pick_year']), num(r['pick_round'])
        tally[(r['kind'], check)] += 1
        rows.append((num(r['txn_id']), r['from_fid'] or None, r['to_fid'] or None, r['kind'], y, rnd,
                     num(r['pick_slot']), r['pick_orig'] or None, num(r['cap_m'], float), r['cap_seasons'] or None,
                     r['player'] or None, r['clarity'], r['reading'], check))
    db.executemany('INSERT INTO note_reading (txn_id, from_fid, to_fid, kind, pick_year, pick_round, pick_slot, '
                   'pick_orig, cap_m, cap_seasons, player, clarity, reading, draft_check) '
                   'VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)', rows)
    missing = db.execute('''SELECT COUNT(*) FROM note_reading WHERE txn_id NOT IN (SELECT txn_id FROM txn)''').fetchone()[0]
    if missing:
        raise SystemExit(f'{missing} readings name a transaction that does not exist')
    db.commit()
    print(f'{len(rows)} readings of {len({r[0] for r in rows})} trade notes')
    for (kind, check), n in sorted(tally.items(), key=str):
        print(f'  {kind:10s} {check or "no draft on record to check":30s} {n}')
    for row in db.execute('''SELECT n.txn_id, n.draft_check, n.reading FROM note_reading n
                              WHERE draft_check NOT IN ('agrees') '''):
        print('  ', row)


if __name__ == '__main__':
    sys.exit(main())
