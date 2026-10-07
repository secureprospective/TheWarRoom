"""Proof that the move history is complete.

Start from MFL's roster snapshot for one week, replay every recorded move (trades, adds, drops)
up to the next week's snapshot, and compare. Any player the replay puts on the wrong team, or
misses, is a move the archive does not record. The match rate is what lets the Lab say that
the ledger it reasons from is the league's real history.
"""
import collections


def reconcile(lab, trades, moves):
    events = collections.defaultdict(list)      # year -> [(ts, order, kind, pid, fid)]
    for t in trades:
        for gave, receiver in ((t['aGave'], t['b']), (t['bGave'], t['a'])):
            for a in gave:
                if a['kind'] == 'player':
                    events[t['year']].append((t['ts'], 1, 'to', a['id'], receiver))
    for m in moves:
        if m['kind'] == 'add':
            events[m['year']].append((m['ts'], 2, 'to', m['pid'], m['fid']))
        elif m['kind'] == 'drop':
            events[m['year']].append((m['ts'], 0, 'drop', m['pid'], m['fid']))
    for year in events:
        events[year].sort()

    def compare(alignment):
        per_year = {}
        for year in lab.years:
            starts = lab.week_start.get(year, {}) if alignment == 'start' else lab.week_end.get(year, {})
            weeks = sorted(w for w in starts if (year, w, '0001') in lab.rosters or any((year, w, f) in lab.rosters for f in lab.franchise_names))
            hit = total = extra = changed = explained = 0
            for w, nxt in zip(weeks, weeks[1:]):
                if nxt != w + 1:
                    continue
                owner = {}
                for (y, ww, fid), roster in lab.rosters.items():
                    if y == year and ww == w:
                        for pid in roster:
                            owner[pid] = fid
                lo, hi = starts[w], starts[nxt]
                for ts, _, kind, pid, fid in events.get(year, ()):
                    if lo <= ts < hi:
                        if kind == 'drop':
                            if owner.get(pid) == fid:
                                del owner[pid]
                        else:
                            owner[pid] = fid
                actual = {}
                for (y, ww, fid), roster in lab.rosters.items():
                    if y == year and ww == nxt:
                        for pid in roster:
                            actual[pid] = fid
                before = {}
                for (y, ww, fid), roster in lab.rosters.items():
                    if y == year and ww == w:
                        for pid in roster:
                            before[pid] = fid
                moved = [pid for pid, fid in actual.items() if before.get(pid) != fid]
                changed += len(moved)
                explained += sum(1 for pid in moved if owner.get(pid) == actual[pid])
                total += len(actual)
                hit += sum(1 for pid, fid in actual.items() if owner.get(pid) == fid)
                extra += sum(1 for pid, fid in owner.items() if pid not in actual)
            if total:
                per_year[year] = {'rostered': total, 'matched': hit, 'replayedNotRostered': extra,
                                  'rate': round(hit / total, 4), 'changed': changed, 'explained': explained,
                                  'changeRate': round(explained / changed, 4) if changed else None}
        return per_year

    best = max(('start', 'end'), key=lambda a: sum(v['matched'] for v in compare(a).values()))
    per_year = compare(best)
    total = sum(v['rostered'] for v in per_year.values())
    matched = sum(v['matched'] for v in per_year.values())
    changed = sum(v['changed'] for v in per_year.values())
    explained = sum(v['explained'] for v in per_year.values())
    return {'changed': changed, 'explained': explained, 'changeRate': round(explained / changed, 4) if changed else None,
            'snapshotAt': 'first kickoff' if best == 'start' else 'last kickoff', 'byYear': per_year,
            'rostered': total, 'matched': matched, 'rate': round(matched / total, 4) if total else None}
