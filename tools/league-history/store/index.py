#!/usr/bin/env python3
"""Add meaning search to the cards: embed every card with a small local model and store the
vectors in data/mfl.db (sqlite-vec). Runs offline on the CPU. Needs the project's private Python
environment:

    .venv/bin/python store/index.py                 # the chosen model
    .venv/bin/python store/index.py --model NAME    # try another (store/exam.py compares them)

The model's name is recorded in the database, and search refuses to mix vectors from two models.
"""
import argparse
import sqlite3
import struct
import sys
import time
from pathlib import Path

import sqlite_vec
from fastembed import TextEmbedding

DB = Path(__file__).resolve().parent.parent / 'data' / 'mfl.db'
MODEL = 'BAAI/bge-base-en-v1.5'
CACHE = Path(__file__).resolve().parent.parent / 'data' / 'models'

# Some models expect a marker saying whether a text is a document or a question.
PREFIX = {
    'nomic-ai/nomic-embed-text-v1.5': ('search_document: ', 'search_query: '),
    'nomic-ai/nomic-embed-text-v1.5-Q': ('search_document: ', 'search_query: '),
    'google/embeddinggemma-300m': ('title: none | text: ', 'task: search result | query: '),
    'BAAI/bge-base-en-v1.5': ('', 'Represent this sentence for searching relevant passages: '),
    'BAAI/bge-small-en-v1.5': ('', 'Represent this sentence for searching relevant passages: '),
}


def connect(path=DB):
    db = sqlite3.connect(path)
    db.enable_load_extension(True)
    sqlite_vec.load(db)
    db.enable_load_extension(False)
    return db


def model(name):
    return TextEmbedding(name, cache_dir=str(CACHE), threads=None)


def pack(vec):
    return struct.pack(f'{len(vec)}f', *vec)


def card_text(title, body):
    return f'{title}\n{body}'


def build(name=MODEL, table='card_vec', path=DB):
    db = connect(path)
    emb = model(name)
    doc_prefix = PREFIX.get(name, ('', ''))[0]
    rows = db.execute('SELECT rowid, title, body FROM card').fetchall()
    # Similar lengths together: each batch is padded to its longest card, so this saves most of the work.
    rows.sort(key=lambda r: len(r[1]) + len(r[2]))
    start = time.time()
    vectors = iter(emb.embed([doc_prefix + card_text(t, b) for _, t, b in rows], batch_size=64))
    first = next(vectors)
    dim = len(first)
    db.executescript(f'DROP TABLE IF EXISTS {table};')
    db.execute(f'CREATE VIRTUAL TABLE {table} USING vec0(embedding float[{dim}] distance_metric=cosine)')
    db.execute('CREATE TABLE IF NOT EXISTS vec_model (tbl TEXT PRIMARY KEY, model TEXT NOT NULL, dim INTEGER, cards INTEGER, built TEXT)')
    db.execute(f'INSERT INTO {table}(rowid, embedding) VALUES (?, ?)', (rows[0][0], pack(first)))
    for (rowid, _, _), vec in zip(rows[1:], vectors):
        db.execute(f'INSERT INTO {table}(rowid, embedding) VALUES (?, ?)', (rowid, pack(vec)))
    db.execute('INSERT OR REPLACE INTO vec_model VALUES (?, ?, ?, ?, datetime(\'now\'))', (table, name, dim, len(rows)))
    db.commit()
    db.execute('PRAGMA wal_checkpoint(TRUNCATE)')
    return len(rows), dim, time.time() - start


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--model', default=MODEL)
    ap.add_argument('--table', default='card_vec')
    a = ap.parse_args()
    n, dim, secs = build(a.model, a.table)
    print(f'{n:,} cards embedded with {a.model} ({dim} dimensions) in {secs:.0f} s, table {a.table}')


if __name__ == '__main__':
    sys.exit(main())
