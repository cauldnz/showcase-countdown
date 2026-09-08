#!/usr/bin/env python3
"""Build and run the note-parser test against the firmware's sequencer.h."""

import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def main():
    compiler = shutil.which("g++") or shutil.which("clang++")
    if compiler is None:
        print("no native C++ compiler found (g++ or clang++) - skipping")
        return 0

    with tempfile.TemporaryDirectory() as tmp:
        exe = Path(tmp) / ("sequencer_test.exe" if sys.platform == "win32" else "sequencer_test")
        subprocess.run(
            [
                compiler,
                "-std=c++11",
                "-Wall",
                "-Wextra",
                "-Werror",
                "-I",
                str(ROOT / "src"),
                str(ROOT / "tests" / "sequencer_test.cpp"),
                "-o",
                str(exe),
            ],
            check=True,
        )
        return subprocess.run([str(exe)]).returncode


if __name__ == "__main__":
    sys.exit(main())
