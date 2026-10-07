#!/usr/bin/env python3
"""Build the facts database from the MFL archive, in order: load the facts, prove them against MFL's
own figures, load the trade-note readings, write the cards. Then, in the private environment,
add meaning search:

    python3 store/build.py && .venv/bin/python store/index.py
"""
import sqlite3
import sys

import cards
import checks
import load
import notes


def main():
    path = load.build()
    db = sqlite3.connect(path)
    results = checks.Checks(db).run()
    failed = [r[0] for r in results if not r[1]]
    db.close()
    notes.main(path)
    db = sqlite3.connect(path)
    checks.write_review(results, notes.review_section(db))
    db.close()
    cards.main(path)
    for name in failed:
        print(f'CHECK FAILED: {name}')
    return 1 if failed else 0


if __name__ == '__main__':
    sys.exit(main())
