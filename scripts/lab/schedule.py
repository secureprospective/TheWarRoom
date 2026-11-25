#!/usr/bin/env python3
import argparse, json, os, sys

FIRST = 13

def circle_round(teams, rnd):
    n = len(teams)
    rot = rnd % (n - 1)
    at = lambda j: teams[1 + (j + rot) % (n - 1)]
    return [(teams[0], at(0))] + [(at(i), at(n - 1 - i)) for i in range(1, n // 2)]

def command(league, name, args, reason):
    slug = league.split('.', 1)[1]
    return {'league': league, 'command': name, 'args': args, 'reason': reason,
            'request': '%s-%s' % (name.replace('.', '-'), slug), 'effective': 'now', 'visibility': 'league'}

def main():
    p = argparse.ArgumentParser()
    p.add_argument('--league', required=True); p.add_argument('--teams', required=True, type=int); p.add_argument('--out', required=True)
    a = p.parse_args()
    if not a.league.startswith('native.'):
        print('league id must start with native.', file=sys.stderr); return 2
    if a.teams < 4 or a.teams > 32 or a.teams % 2:
        print('teams must be even and between 4 and 32', file=sys.stderr); return 2
    ids = ['franchise.%02d' % i for i in range(1, a.teams + 1)]
    weeks = [{'week': w, 'games': []} for w in range(1, FIRST)]
    for w in range(FIRST, 19):
        pairs = circle_round(ids, w - FIRST)
        weeks.append({'week': w, 'games': [{'id': 'w%d-g%02d' % (w,k), 'home': h, 'away': t} for k,(h,t) in enumerate(pairs)]})
    os.makedirs(a.out, exist_ok=True)
    docs = [('schedule-generate.json', command(a.league, 'schedule.generate', {'schedule': {'weeks': 18, 'fixtures': weeks}}, 'Rehearsal schedule')),
            ('schedule-publish.json', command(a.league, 'schedule.publish', {'schedule': {}}, 'Rehearsal schedule'))]
    for fn, doc in docs:
        with open(os.path.join(a.out, fn), 'w') as f: json.dump(doc, f, indent=1); f.write('\n')
    return 0

if __name__ == '__main__': raise SystemExit(main())
