# Spike: Parallel table dumping

**Status:** Investigated — **deferred** (do not implement as-is)
**Date:** 2026-07-14

## Question

Can `dbdump` dump tables concurrently (a worker pool of `mysqldump`
invocations) to speed up large dumps, without losing the consistent,
restorable snapshot that `--single-transaction` gives us today?

## Finding: no consistent snapshot with parallel `mysqldump`

The current dump runs two sequential `mysqldump` passes (structure, then data)
over one connection with `--single-transaction`, producing a single
point-in-time snapshot.

Parallelising at the table level means running **N independent `mysqldump`
processes**. Each one opens its own connection and issues its own
`START TRANSACTION WITH CONSISTENT SNAPSHOT`, capturing a **separate** InnoDB
read view at a slightly different instant. So N workers dump N different points
in time. Any write committed between worker starts makes cross-table state
(foreign keys, parent/child rows, invariants) inconsistent in the resulting
dump. On a dev database with active writers this silently produces a torn
snapshot — breaking the tool's core promise.

InnoDB read views are **per session and cannot be shared** with another
connection, so you cannot hand one point-in-time to N `mysqldump` subprocesses.

The only way to get a globally consistent parallel dump is the
`mydumper`/`mysqlpump` technique:

1. a coordinator connection takes `FLUSH TABLES WITH READ LOCK` (needs `RELOAD`
   privilege; briefly stalls all writes),
2. every worker connection issues `START TRANSACTION WITH CONSISTENT SNAPSHOT`
   while the lock is held,
3. record the binlog/GTID position, then release the lock.

That coordination has to live **inside our process across a shared connection
pool** — and `mysqldump`-as-a-subprocess offers no hook to inject it. Achieving
consistency therefore means abandoning `mysqldump` for the data pass and
reimplementing row `SELECT` + `INSERT` formatting (escaping, `--hex-blob`,
generated/virtual columns, …) — i.e. rebuilding `mydumper`.

## Other blockers

- **Single large table** (common) can't be parallelised at table granularity;
  speedup is capped by the biggest table unless PK-range chunking is also built.
- **Compression is the frequent real bottleneck** and is single-threaded through
  one gzip/zstd writer; parallel dumping just backs up at the serialized
  merge/compress step.
- **More failure surface:** N processes + N connections, an ordered temp-file
  merge, kill-all-on-error, and cleanup of N temp files.

## Estimated payoff

~1.5–3× on medium, many-table databases over a fast/high-latency link with a
multi-core client; ~1× (or slightly negative) on small dev DBs, single-large-
table DBs, or when single-threaded compression dominates.

## Recommendation

**Defer.** The consistency cost is not a tuning knob — preserving it requires a
`mydumper`-scale rewrite off `mysqldump`, disproportionate to a dev-database
tool, for a modest and workload-dependent speedup.

If revisited, make it an explicit opt-in `--parallel N` that documents it
**relaxes cross-table consistency**, or adopt `mydumper` rather than hand-rolling
snapshot coordination.

## Cheaper alternative worth considering first

Most practical slowness is single-threaded compression, not dump concurrency.
Enabling multi-threaded zstd on the **existing single stream**
(`klauspost/compress` `zstd.WithEncoderConcurrency`) is a near-free 2–4× win on
compression-bound dumps with **zero** consistency impact. Not implemented here;
recorded as the recommended next step if dump wall-clock becomes a concern.
