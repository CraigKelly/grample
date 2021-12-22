#!/usr/bin/env python3

import csv
import json
import sys


def valid_lines():
    """Yield all valid lines in the "VARS (ESTIMATED)" section"""
    started = False
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue

        if not started:
            if line.startswith("// VARS (ESTIMATED)"):
                started = True
            continue

        if line.startswith("// "):
            return

        yield line


def main():
    """Entry point."""
    cols = []
    rows = []
    for line in valid_lines():
        rec = json.loads(line)
        for k, v in rec["State"].items():
            rec[k] = v
        del rec["State"]

        if not cols:
            cols = list(rec.keys())

        rows.append(rec)

    for c in cols:
        if c.endswith("-Error"):
            new_col = c + "-RANK"
            sys.stderr.write("{} <= {}\n".format(new_col, c))
            rows.sort(key=lambda r: float(r[c]))

        elif c.endswith("-Convergence"):
            ec = c.replace("-Convergence", "-Error")
            new_col = c + "-RANK"
            sys.stderr.write("{} <= {} {}\n".format(new_col, c, ec))
            rows.sort(key=lambda r: (float(r[c]), float(r[ec])))

        else:
            continue

        cols.append(new_col)
        for i, r in enumerate(rows):
            r[new_col] = i + 1

    wr = csv.DictWriter(sys.stdout, fieldnames=sorted(cols))
    wr.writeheader()
    wr.writerows(rows)
    sys.stdout.flush()


if __name__ == "__main__":
    main()
