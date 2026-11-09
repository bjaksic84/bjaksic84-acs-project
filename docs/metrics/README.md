# Metrics

Generated, not written by hand.

- `score.csv`: one row per `make score` run: the date, the commit, passed, failed and skipped
  locked tests, and passed and total per tier. Rows from sessions in progress may show a commit
  with uncommitted changes; the session files say which rows belong to which session.
- `ratchet.txt`: every locked test that has passed. `make score` fails when one of them fails
  again.

Any commit's score can be recomputed by checking it out and running `make score`, because the
suite is locked and deterministic (the `[net]` tests need the Go module proxy).
