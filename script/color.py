#!/usr/bin/env python3

import os
import sys

try:
    import colorclass
except ImportError:
    print("color.py requires colorclass -- ATTEMPTING INSTALL", file=sys.stderr)
    import subprocess

    subprocess.run(
        "python3 -m pip install --user --upgrade colorclass", shell=True, check=True
    )
    import colorclass

# Handle Windows vs everyone else
if os.name == "nt":
    colorclass.Windows.enable(auto_colors=True)
else:
    colorclass.set_dark_background()


def line_process(line):
    """Add any necessary color to the line and return the final, corrected line."""
    # path/file.go:3:14: Warning about line 1, col 14 on file.go
    flds = line.strip().split(":")
    if len(flds) < 3:
        return line.rstrip()  # Not in a format we recognize

    clr = colorclass.Color

    # Note that we currently do nothing to msg
    # Also note that col doesn't always show up
    fname, line, *rest = flds
    if len(rest) > 1:
        col, *rest = rest
    else:
        col = " "

    fname = clr("{autoyellow}{u}%s{/u}{/autoyellow}" % fname)
    line = clr("{autogreen}%s{/autogreen}" % line)
    col = clr("{autogreen}%s{/autogreen}" % col)

    delim = clr("{b}:{/b}")

    return delim.join([fname, line, col, *rest])


def main():
    for line in sys.stdin:
        sys.stdout.write(line_process(line))
        sys.stdout.write("\n")


if __name__ == "__main__":
    main()
