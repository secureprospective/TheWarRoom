#!/usr/bin/env python3
"""The fact exam: answer questions from the facts database and compare each answer with what MFL's
own web pages show a GM. MFL's pages are the answer key, because they are what Christopher checks.

Pages are fetched once, six seconds apart, and kept in data/mfl-pages/; later runs read the copies.

    python3 store/factexam.py
"""
import html
import json
import re
import sqlite3
import sys
import time
import urllib.request
from pathlib import Path

TOOL = Path(__file__).resolve().parent.parent
DB = TOOL / 'data' / 'mfl.db'
PAGES = TOOL / 'data' / 'mfl-pages'
HOST = {2013: ('www48', '51719'), 2014: ('www46', '47710'), 2015: ('www45', '21225')}
HOST.update({y: ('www45' if y <= 2021 else 'www47', '14432') for y in range(2016, 2027)})

STANDINGS_YEARS = [2014, 2016, 2019, 2021, 2025]
DRAFT_YEARS = [2018, 2021, 2023, 2024]
PLAYERS = ['14073', '15838']      # two players' pages, for birthdates and NFL draft slots


def page(name, url):
    PAGES.mkdir(parents=True, exist_ok=True)
    path = PAGES / f'{name}.html'
    if not path.exists():
        time.sleep(6)
        req = urllib.request.Request(url, headers={'User-Agent': 'TheWarRoom-league-history-exam'})
        with urllib.request.urlopen(req, timeout=60) as r:
            path.write_bytes(r.read())
    return path.read_text(encoding='utf-8', errors='ignore')


def url(year, path):
    host, lid = HOST[year]
    return f'https://{host}.myfantasyleague.com/{year}/{path}{"&" if "?" in path else "?"}L={lid}'


def cells(text):
    rows = re.findall(r'<tr[^>]*>(.*?)</tr>', text, re.S)
    out = []
    for r in rows:
        cs = [html.unescape(re.sub(r'\s+', ' ', re.sub(r'<[^>]+>', ' ', c))).strip()
              for c in re.findall(r'<t[hd][^>]*>(.*?)</t[hd]>', r, re.S)]
        if any(cs):
            out.append(cs)
    return out


class Exam:
    def __init__(self, db):
        self.db = db
        self.results = []

    def ask(self, question, mfl_says, store_says, note=''):
        ok = str(mfl_says).strip().lower() == str(store_says).strip().lower()
        self.results.append((question, mfl_says, store_says, ok, note))

    def standings(self):
        for y in STANDINGS_YEARS:
            rows = cells(page(f'standings_{y}', url(y, 'standings')))
            names = dict(self.db.execute('SELECT name, fid FROM franchise_season WHERE year = ?', (y,)))
            for r in rows:
                if len(r) > 6 and r[0] in names and re.fullmatch(r'\d+-\d+-\d+', r[1].replace('‑', '-')):
                    fid = names[r[0]]
                    ts = self.db.execute('''SELECT reg_w, reg_l, reg_t, all_weeks_pf FROM team_season
                                            WHERE year = ? AND fid = ?''', (y, fid)).fetchone()
                    every = self.db.execute('''SELECT SUM(result='W'), SUM(result='L'), SUM(result='T') FROM played_game
                                               WHERE year = ? AND fid = ? AND opp IS NOT NULL''', (y, fid)).fetchone()
                    record = r[1].replace('‑', '-')
                    mine = f'{ts[0]}-{ts[1]}-{ts[2]}' if y != 2013 else f'{every[0]}-{every[1]}-{every[2]}'
                    self.ask(f'{r[0]}: {y} record?', record, mine, 'counted from the weekly games')
                    self.ask(f'{r[0]}: {y} points for?', f'{float(r[5]):.2f}', f'{ts[3]:.2f}',
                             'summed from the weekly games')

    def drafts(self):
        for y in DRAFT_YEARS:
            text = page(f'draft_{y}', url(y, 'options?O=17'))
            names = {n: f for f, n in self.db.execute('SELECT fid, name FROM franchise_season WHERE year = ?', (y,))}
            for r in cells(text):
                pick = next((c for c in r if re.fullmatch(r'\d+\.\d\d', c)), None)
                team = next((c for c in r if c in names), None)
                who = next((c for c in r if re.match(r"^[A-Z][\w.'\- ]+, [\w.'\- ]+ [A-Z]{2,3} [A-Z]{1,2}\b", c)), None)
                when = next((c for c in r if re.search(r' ET \d{4}$', c)), None)
                if not (pick and team and who):
                    continue
                rnd, slot = (int(x) for x in pick.split('.'))
                row = self.db.execute('''SELECT d.fid, p.name, d.day FROM draft_pick d LEFT JOIN player p USING (pid)
                                         WHERE d.year = ? AND d.round = ? AND d.pick = ?''', (y, rnd, slot)).fetchone()
                if not row:
                    self.ask(f'{y} pick {pick}: who made it?', team, 'no such pick in the store')
                    continue
                self.ask(f'{y} pick {pick}: which team?', team, self.db.execute(
                    'SELECT name FROM franchise_season WHERE year = ? AND fid = ?', (y, row[0])).fetchone()[0])
                listed = re.sub(r' [A-Z]{2,3} [A-Z]{1,2}( \(R\))?$', '', who)    # drop "HOU QB"
                then = self.db.execute('SELECT name FROM player_season WHERE year = ? AND pid = (SELECT pid FROM '
                                       'draft_pick WHERE year = ? AND round = ? AND pick = ?)', (y, y, rnd, slot)).fetchone()
                self.ask(f'{y} pick {pick}: which player?', listed, then[0] if then else row[1],
                         'the name MFL listed that season')
                if when:
                    m = re.search(r'(\w{3}) (\d+) .* ET (\d{4})$', when)
                    months = 'Jan Feb Mar Apr May Jun Jul Aug Sep Oct Nov Dec'.split()
                    shown = f'{m.group(3)}-{months.index(m.group(1)) + 1:02d}-{int(m.group(2)):02d}'
                    self.ask(f'{y} pick {pick}: what day?', shown, row[2], 'MFL shows US Eastern time')

    def players(self):
        for pid in PLAYERS:
            text = page(f'player_{pid}', f'https://www47.myfantasyleague.com/2026/player?L=14432&P={pid}')
            flat = html.unescape(re.sub(r'\s+', ' ', re.sub(r'<[^>]+>', ' ', text)))
            born = re.search(r'DOB/Age: (\w{3} \d+, \d{4})', flat)
            drafted = re.search(r'Drafted: (\d{4}) / [^/]*/ Round (\d+), Pick (\d+)', flat)
            row = self.db.execute('SELECT name, birthdate, draft_year, draft_round, draft_pick FROM player WHERE pid = ?',
                                  (pid,)).fetchone()
            if born:
                shown = time.strftime('%Y-%m-%d', time.strptime(born.group(1), '%b %d, %Y'))
                self.ask(f'{row[0]}: birthdate?', shown, row[1])
            if drafted:
                self.ask(f'{row[0]}: NFL draft?', ' '.join(drafted.groups()), f'{row[2]} {row[3]} {row[4]}')

    def run(self):
        self.standings()
        self.drafts()
        self.players()
        return self.results


def main():
    db = sqlite3.connect(f'file:{DB}?mode=ro', uri=True)
    results = Exam(db).run()
    right = sum(1 for r in results if r[3])
    wrong = [r for r in results if r[3] is False]
    unknown = [r for r in results if r[3] is None]
    print(f'{right} of {len(results)} answers match MFL\'s pages; {len(wrong)} differ; {len(unknown)} not checkable')
    for q, mfl, mine, _, note in wrong:
        print(f'  DIFFERS  {q}  MFL: {mfl}  store: {mine}  ({note})' if note else f'  DIFFERS  {q}  MFL: {mfl}  store: {mine}')
    for q, mfl, mine, _, _ in unknown:
        print(f'  ?        {q}  MFL: {mfl}  store: {mine}')
    (TOOL / 'data' / 'factexam.json').write_text(json.dumps(results, indent=1))
    return 0 if not wrong else 1


if __name__ == '__main__':
    sys.exit(main())
