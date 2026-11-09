package main

// The hosted server runs on a minimal Alpine container inside a sandbox with no zoneinfo, yet league
// rules name time zones (calendar.timezone). Embedding Go's time zone database makes every owner
// action that reads the league calendar work there, instead of failing "unknown time zone".
import _ "time/tzdata"
