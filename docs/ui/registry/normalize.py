"""Normalize the endpoint map; placement calls are explicit audit data."""
import argparse
import csv
import re
from collections import Counter
from pathlib import Path

WORKSPACES = {
    'Home': ('Seasonal card', 'Alert tray', 'Digests'),
    'War Room': ('Market', 'Valuations and pool', 'Research'),
    'Franchise HQ': ('Lineup and roster', 'Contracts and cap', 'My moves'),
    'Trade Floor': ('Trade desk and offers', 'Counterparties', 'Draft room'),
    'League Pulse': ('Now', 'Race', 'Pools / Archive'),
    'Control Room': ('App', 'Data and sources', 'League'),
}
SURFACES = ('Inspector', 'Comms', 'Calendar', 'Command bar', 'Status strip', 'Report drawer')
PLACEMENTS = {f'{n} › {w}' for n, ws in WORKSPACES.items() for w in ws} | set(SURFACES)
ALIASES = {
    'War Room › Valuations & pool': 'War Room › Valuations and pool',
    'Franchise HQ › Contracts & cap': 'Franchise HQ › Contracts and cap',
    'Franchise HQ › Roster': 'Franchise HQ › Lineup and roster',
    'Franchise HQ › Lineup': 'Franchise HQ › Lineup and roster',
    'Franchise HQ › Acquire': 'Franchise HQ › My moves',
    'Trade Floor › Trade desk': 'Trade Floor › Trade desk and offers',
    'Trade Floor › Offers': 'Trade Floor › Trade desk and offers',
    'League Pulse › Archive': 'League Pulse › Pools / Archive',
    'League Pulse › Pools': 'League Pulse › Pools / Archive',
    'League Pulse › Now (Gameday preset)': 'League Pulse › Now',
    'Control Room › Data & sources': 'Control Room › Data and sources',
    'Control Room › Data & sources (Admin role)': 'Control Room › Data and sources',
    'Control Room › League (role-gated)': 'Control Room › League',
    'Control Room › MFL utilities': 'Control Room › App',
    'Comms panel': 'Comms', 'Status bar': 'Status strip', 'Calendar (summoned)': 'Calendar',
}
JUDGEMENTS = {
    'M-001': ('League Pulse › Now', 'live games belong in Now'),
    'M-002': ('League Pulse › Race', 'standings answer the race question'),
    'M-008': ('League Pulse › Race', 'power rankings measure the race'),
    'M-032': ('Home › Digests', 'league-wide activity digest'),
    'M-058': ('Home › Digests', 'player news digest'),
    'M-063': ('Trade Floor › Draft room', 'past draft results'),
    'M-064': ('Trade Floor › Draft room', 'draft board grid'),
    'M-065': ('Trade Floor › Draft room', 'draft pricing review'),
    'M-066': ('Trade Floor › Draft room', 'draft recap'),
    'M-068': ('Trade Floor › Draft room', 'future pick ownership'),
    'M-087': ('Trade Floor › Draft room', 'NFL draft entrants'),
    'M-091': ('Control Room › League', 'configured league rules'),
    'M-093': ('Control Room › League', 'league mail audit'),
    'M-095': ('Control Room › League', 'commissioner notices'),
    'M-096': ('Calendar', 'due dates match calendar summon'),
    'M-097': ('Control Room › League', 'written league rules'),
    'M-098': ('Control Room › League', 'league accounting'),
    'M-102': ('Trade Floor › Draft room', 'draft pick identity'),
    'M-103': ('Trade Floor › Draft room', 'personal draft board'),
    'M-108': ('Control Room › App', 'personal MFL automation settings'),
    'M-114': ('League Pulse › Pools / Archive', 'past champions are history'),
    'M-137': ('Control Room › App', 'note names MFL utilities; canonical app home'),
    'M-148': ('Home › Alert tray', 'lineup legality alert'),
    'M-149': ('Home › Alert tray', 'IR compliance alert'),
    'M-153': ('Home › Alert tray', 'pending trade deadline alert'),
    'M-156': ('Home › Alert tray', 'pending waiver claim alert'),
    'M-157': ('Home › Alert tray', 'waiver claim order context'),
    'M-176': ('Trade Floor › Draft room', 'draft status'),
    'M-179': ('Trade Floor › Draft room', 'draft pace'),
    'M-182': ('League Pulse › Pools / Archive', 'note explicitly places history in Archive'),
    'M-183': ('Trade Floor › Draft room', 'canonical live draft console'),
    'M-185': ('Control Room › League', 'league bylaws'),
    'L-30': ('Trade Floor › Trade desk and offers', 'note names Offers; canonical desk'),
    'L-34': ('Control Room › Data and sources', 'Sources provenance is this workspace panel'),
    'C-02': ('League Pulse › Race', 'note wins: power rankings in Race'),
    'C-03': ('Home › Seasonal card', 'note: seasonal card owns Home'),
    'C-05': ('Home › Digests', 'note: league feed has a Home digest'),
    'S-01': ('Home › Seasonal card', 'PROVISIONAL: rail indexes the Home-owned shell'),
    'S-02': ('Inspector', 'entity surface'),
    'S-03': ('Comms', 'comms surface'),
    'S-04': ('Calendar', 'calendar surface'),
    'S-10': ('Control Room › App', 'PROVISIONAL: filter persistence is app settings'),
    'S-13': ('Home › Seasonal card', 'seasonal phase links'),
}
MERGED_PLACES = {
    'M-145': ('Inspector', 'player hub → inspector player view (spec §2)'),
    'M-146': ('Inspector', 'franchise hub → inspector franchise view (spec §2)'),
    'M-147': ('Home › Seasonal card', 'MFL HOME tab → app Home (Seasonal card owns the node)'),
    'M-152': ('Franchise HQ › Lineup and roster', 'MFL MY TEAM tab → Franchise HQ'),
    'M-159': ('Report drawer', 'MFL REPORTS tab → the report drawer'),
    'M-162': ('War Room › Valuations and pool', 'MFL PLAYERS tab → the asset board and pool'),
    'M-165': ('Home › Digests', 'MFL LATEST NEWS tab → Home digests'),
    'M-168': ('League Pulse › Race', 'MFL PLAYOFFS tab → the race'),
    'M-171': ('League Pulse › Pools / Archive', 'MFL POOLS tab → Pools'),
    'M-180': ('Status strip', 'news ticker → status strip'),
    'M-181': ('Status strip', 'scoreboard strip → status strip (gameday ticker)'),
}
# Conditional chrome retains its baseline; an ambiguous commit gate takes the higher consequence.
GRAVITY_CALLS = {'M-110': 'G0', 'L-12': 'G0', 'C-06.3': 'G3', 'S-02': 'G0'}
PLACE_RINGS = dict(zip(SURFACES, (0, 3, 0, 0, 0, 3))) | {
    'Home › Seasonal card': 1, 'Franchise HQ › Lineup and roster': 1,
    'War Room › Valuations and pool': 2, 'Home › Digests': 3,
    'League Pulse › Race': 3, 'League Pulse › Pools / Archive': 3,
}

METADATA_JUDGEMENTS = {
    'M-027': 'PROVISIONAL: G3 baseline; kickoff qualifier retained in gravity_note',
    'M-057': 'PROVISIONAL: G1 baseline; gameday promotion retained in gravity_note',
    'M-067': 'PROVISIONAL: G1 baseline; clock promotion retained in gravity_note',
    'M-093': 'PROVISIONAL: G0 for GM; audit qualifier retained in gravity_note',
    'M-096': 'PROVISIONAL: G2 baseline; deadline promotion retained in gravity_note',
    'M-110': 'PROVISIONAL: G0 baseline for ambient summary; original G0/G1 retained',
    'M-148': 'PROVISIONAL: G3 baseline; gameday qualifier retained in gravity_note',
    'M-153': 'PROVISIONAL: G3 baseline; pending qualifier retained in gravity_note',
    'M-183': 'PROVISIONAL: G3 baseline; clock qualifier retained in gravity_note',
    'L-12': 'PROVISIONAL: G0 baseline for profile; original G0/G1 retained',
    'C-06.2': 'PROVISIONAL: G3 baseline; draft qualifier retained in gravity_note',
    'C-06.3': 'PROVISIONAL: G3 conservatively for ambiguous G2/G3 commit gate',
    'C-06.7': 'PROVISIONAL: G3 baseline; window qualifier retained in gravity_note',
    'S-02': 'PROVISIONAL: G0 baseline; entity-dependent G0–G3 retained in gravity_note',
    'S-04': 'PROVISIONAL: G3 baseline; hosted deadline qualifier retained in gravity_note',
    'X-03': 'PROVISIONAL: ring 1 ships Classic; ring 4 Default retained in ring_note',
}
RETIRED_IDS = """M-006 M-017 M-026 M-035 M-061 M-070 M-092 M-094 M-099 M-100 M-106
M-107 M-109 M-115 M-119 M-128 M-129 M-130 M-131 M-132 M-133 M-134 M-135 M-136
M-138 M-139 M-140 M-142 M-143 M-144 M-155 M-184 M-191""".split()
METADATA_JUDGEMENTS.update({ident: 'PROVISIONAL: retired — becomes inert G0 / ring 4, never routed'
                            for ident in RETIRED_IDS})
METADATA_JUDGEMENTS.update({ident: 'PROVISIONAL: with target inherits earliest target/place ring'
                            for ident in """M-145 M-146 M-147 M-150 M-151 M-152 M-158 M-159
M-160 M-162 M-163 M-164 M-165 M-166 M-167 M-168 M-169 M-171 M-175 M-177 M-178 M-180
M-181 M-186 M-187 M-188 M-189 M-190""".split()})


def validate(rows):
    by_id = {r['id']: r for r in rows}
    if len(by_id) != len(rows):
        raise ValueError('duplicate endpoint id')
    for r in rows:
        ident = r['id']
        if not re.fullmatch(r'(?:M-\d{3}|[LCSX]-\d{2}(?:\.\d+)?)', ident):
            raise ValueError(f'{ident}: invalid id')
        disposition = r['disposition']
        if disposition not in ('kept', 'merged', 'retired'):
            raise ValueError(f'{ident}: invalid disposition')
        if r['gravity'] not in ('G0', 'G1', 'G2', 'G3'):
            raise ValueError(f'{ident}: invalid gravity')
        if r['ring'] not in tuple('01234'):
            raise ValueError(f'{ident}: invalid ring')
        if disposition == 'kept' and r['placement'] not in PLACEMENTS:
            raise ValueError(f'{ident}: kept row needs valid placement')
        if disposition != 'kept' and r['placement']:
            raise ValueError(f'{ident}: {disposition} row has placement')
        targets, place = r['merged_into'].split(), r['merged_place']
        if disposition == 'merged':
            if bool(targets) == bool(place):
                raise ValueError(f'{ident}: merged row needs exactly one target kind')
            if place and place not in PLACEMENTS:
                raise ValueError(f'{ident}: invalid merged place')
            for target in targets:
                if target not in by_id or by_id[target]['disposition'] != 'kept':
                    raise ValueError(f'{ident}: merged target {target} is not kept')
        elif targets or place:
            raise ValueError(f'{ident}: non-merged row has merge target')


def normalize(path):
    with path.open(newline='', encoding='utf-8') as file:
        reader = csv.DictReader(file)
        columns = list(reader.fieldnames)
        rows = list(reader)
    if 'disposition_note' not in columns:
        columns.extend(('disposition_note', 'merged_into', 'merged_place', 'gravity_reason', 'ring_note', 'gravity_note'))
        for r in rows:
            r['disposition_note'] = r['disposition']
            r['disposition'] = r['disposition'].split()[0]
            r['merged_into'] = r['merged_place'] = ''
            if r['disposition'] == 'merged':
                r['merged_into'] = ' '.join(dict.fromkeys(re.findall(
                    r'\b(?:M-\d{3}|[LCSX]-\d{2}(?:\.\d+)?)\b', r['disposition_note'])))
                if r['id'] in MERGED_PLACES:
                    r['merged_place'] = MERGED_PLACES[r['id']][0]
            r['placement'] = ALIASES.get(r['placement'], r['placement'])
            if r['disposition'] != 'kept':
                r['placement'] = ''
            elif r['id'] in JUDGEMENTS:
                r['placement'] = JUDGEMENTS[r['id']][0]
            original = r['gravity']
            r['gravity_note'] = original
            code = re.match(r'G[0-3]', original)
            if not code and not (original == '—' and r['disposition'] == 'retired'):
                raise ValueError(f"{r['id']}: invalid gravity {original}")
            r['gravity'] = GRAVITY_CALLS.get(r['id'], code[0] if code else 'G0')
            r['gravity_reason'] = original.split(' — ', 1)[1] if ' — ' in original else (
                original[len(code[0]):].strip() if code else original)
            r['ring_note'] = r['ring']
            if r['ring'] == '—' and r['disposition'] == 'retired':
                r['ring'] = '4'
            elif r['id'] == 'X-03' and r['ring'] == '1 (Classic) / 4 (Default)':
                r['ring'] = '1'
        by_id = {r['id']: r for r in rows}
        for r in rows:
            if r['ring'] == 'with target':
                r['ring'] = str(min(int(by_id[t]['ring']) for t in r['merged_into'].split())) \
                    if r['merged_into'] else str(PLACE_RINGS[r['merged_place']])
    validate(rows)
    with path.open('w', newline='', encoding='utf-8') as file:
        writer = csv.DictWriter(file, columns, lineterminator='\n')
        writer.writeheader()
        writer.writerows(rows)
    print('Disposition:', dict(sorted(Counter(r['disposition'] for r in rows).items())))
    print('Placement:', dict(sorted(Counter(r['placement'] for r in rows if r['placement']).items())))
    print('Placement judgements:', len(JUDGEMENTS), '; merged-place judgements:', len(MERGED_PLACES))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('path', nargs='?', type=Path,
                        default=Path(__file__).resolve().parents[1] / 'endpoint-registry.csv')
    normalize(parser.parse_args().path)
