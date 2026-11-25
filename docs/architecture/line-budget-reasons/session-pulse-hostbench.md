Test-only growth (about 32 lines in internal/store/history). The results benchmark now runs on the hosted
two-file layout the Lab uses, which showed the 400 ms 20-reader burst was a benchmark artifact (hosted: 21 ms).
The added 20-reader check proves every hosted reader after a score write sees the new scores; a planted
cross-revision score-cache break fails it.
