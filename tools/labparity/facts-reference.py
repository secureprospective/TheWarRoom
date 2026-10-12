#!/usr/bin/env python3
"""Rebuild the read-only Python facts reference on exact pins, entirely offline.

The output directory must be disposable. No Go output is read. The five absent
error-envelope pins are recorded, not reconstructed. Run this before a Go diff.
"""
import argparse
import csv
import gzip
import hashlib
import json
import re
from pathlib import Path
import shutil
import sqlite3
import subprocess
import sys


def digest(body):
    return hashlib.sha256(body).hexdigest()


def stage(args):
    repo = args.output / 'reference'
    store = repo / 'tools/league-history/store'
    store.mkdir(parents=True, exist_ok=False)
    source = args.reference / 'tools/league-history/store'
    for name in ('schema.sql', 'views.sql', 'load.py', 'checks.py', 'notes.py',
                 'cards.py', 'build.py', 'note_readings.csv'):
        shutil.copyfile(source / name, store / name)
    raw = repo / 'league-archive/raw'
    approved = {(r['file'], r['sha256']): r for r in csv.DictReader(args.gaps.open())}
    missing, held = [], []
    for pin in csv.DictReader(args.pins.open()):
        file = pin['file']
        if Path(file).is_absolute() or '..' in Path(file).parts:
            raise ValueError(f'unsafe pin: {file}')
        src = args.archive / file
        body = src.read_bytes() if src.is_file() else None
        if body is None or digest(body) != pin['sha256']:
            gap = approved.get((file, pin['sha256']))
            if gap is None or pin['bytes'] != '175':
                raise ValueError(f'no approved gap for mismatching pin: {file}')
            missing.append({**pin, 'reason': gap['reason']})
            continue
        if len(body) != int(pin['bytes']):
            raise ValueError(f'size mismatch: {file}')
        dst = raw / file
        dst.parent.mkdir(parents=True, exist_ok=True)
        dst.write_bytes(body)
        held.append(pin)
    (args.output / 'selection.json').write_text(json.dumps({
        'pins_sha256': digest(args.pins.read_bytes()), 'held': len(held),
        'missing': missing, 'notes_sha256': digest((store / 'note_readings.csv').read_bytes()),
        'reference': (args.reference / 'SOURCE.txt').read_text().strip(),
    }, indent=2) + '\n')
    return store


def extract(db_path, output):
    db = sqlite3.connect(f'file:{db_path}?mode=ro', uri=True)
    store = db_path.parent.parent / 'store'
    relations = set(re.findall(r'CREATE (?:TABLE|VIEW) (\w+)',
                              (store / 'schema.sql').read_text() + (store / 'views.sql').read_text()))
    relations.update(('note_reading', 'check_result', 'review_item'))
    tables = [row for row in db.execute("SELECT name, type FROM sqlite_master WHERE type IN ('table', 'view') ORDER BY name")
              if row[0] in relations]
    selection = json.loads((output / 'selection.json').read_text())
    pins = {r['file']: (int(r['bytes']), r['sha256']) for r in csv.DictReader((output / 'pins.csv').open())}
    excluded = {r['file'] for r in selection['missing']}
    expected = {file: data for file, data in pins.items() if file not in excluded}
    actual = {src: (size, sha) for src, size, sha in db.execute('SELECT src,bytes,sha256 FROM source_file')}
    if actual != expected:
        raise ValueError('reference source_file differs from the staged exact pins')
    proof = {
        'pins': len(pins), 'resolved': len(actual), 'missing': selection['missing'],
        'calendar_rows_2013_2015': db.execute('SELECT COUNT(*) FROM calendar_event WHERE year BETWEEN 2013 AND 2015').fetchone()[0],
        'assets_rows_2013_2015': db.execute("SELECT COUNT(*) FROM pick_owned WHERE source='assets' AND snapshot_year BETWEEN 2013 AND 2015").fetchone()[0],
        'calendar_first_recorded_year': db.execute('SELECT MIN(year) FROM calendar_event').fetchone()[0],
    }
    (output / 'gap-proof.json').write_text(json.dumps(proof, indent=2) + '\n')
    counts = {}
    # Names and columns come only from the pinned reference schema, not user data.
    with (output / 'facts.jsonl.gz').open('wb') as file:
        with gzip.GzipFile(fileobj=file, mode='wb', mtime=0, filename='') as gz:
            for table, kind in tables:
                columns = [r[1] for r in db.execute(f'PRAGMA table_info("{table}")')]
                order = ', '.join(f'"{c}"' for c in columns)
                query = f'SELECT * FROM "{table}" ORDER BY {order}'
                gz.write((json.dumps({'table': table, 'kind': kind, 'columns': columns}, separators=(',', ':')) + '\n').encode())
                count = 0
                for row in db.execute(query):
                    gz.write((json.dumps(row, separators=(',', ':'), ensure_ascii=True, allow_nan=False) + '\n').encode())
                    count += 1
                counts[table] = count
    checks = [dict(zip([c[1] for c in db.execute('PRAGMA table_info(check_result)')], r))
              for r in db.execute('SELECT * FROM check_result ORDER BY name')]
    # These identities replace the reference's permitted-open counts in a future Go port.
    exceptions = {name: [r[0] for r in db.execute('SELECT what FROM review_item WHERE check_name=? ORDER BY what', (name,))]
                  for name in sorted({r['name'] for r in checks})}
    exceptions['2013 playoff standings'] = db.execute('''
        WITH every AS (SELECT year, fid, SUM(result='W') w, SUM(result='L') l, SUM(result='T') t
                       FROM played_game WHERE opp IS NOT NULL GROUP BY year, fid)
        SELECT s.year, s.fid, s.mfl_w, s.mfl_l, s.mfl_t, s.reg_w, s.reg_l, s.reg_t
        FROM team_season s JOIN every e USING(year, fid)
        WHERE s.year=2013 AND (s.mfl_w,s.mfl_l,s.mfl_t) != (s.reg_w,s.reg_l,s.reg_t)
          AND (s.mfl_w,s.mfl_l,s.mfl_t) = (e.w,e.l,e.t) ORDER BY s.year,s.fid''').fetchall()
    exceptions['3-point bonus games'] = db.execute('''
        WITH starters AS (SELECT year,week,fid,ROUND(SUM(score),2) st FROM lineup WHERE started=1 GROUP BY year,week,fid)
        SELECT g.year,g.week,g.fid,g.score,s.st FROM played_game g JOIN starters s USING(year,week,fid)
        WHERE ABS(ROUND(g.score-s.st,2)-3) <= 0.011 ORDER BY g.year,g.week,g.fid''').fetchall()
    (output / 'counts.json').write_text(json.dumps(counts, indent=2) + '\n')
    (output / 'checks.json').write_text(json.dumps(checks, indent=2) + '\n')
    (output / 'exceptions.json').write_text(json.dumps(exceptions, indent=2) + '\n')
    db.close()
    return counts


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('reference', 'archive', 'pins', 'gaps', 'output'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    store = stage(args)
    shutil.copyfile(args.pins, args.output / 'pins.csv')
    with (args.output / 'build.log').open('w') as log:
        result = subprocess.run([sys.executable, str(store / 'build.py')], stdout=log, stderr=subprocess.STDOUT, check=False)
    (args.output / 'build-exit.txt').write_text(str(result.returncode) + '\n')
    if result.returncode != 0:
        raise SystemExit('Reference build failed; see build.log. No golden exported.')
    counts = extract(store.parent / 'data/mfl.db', args.output)
    print(f'Reference built: {len(counts)} relations, {sum(counts.values())} rows exported.')


if __name__ == '__main__':
    main()
