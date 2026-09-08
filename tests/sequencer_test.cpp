// Host test for the team-audio note parser. Built and run by test_sequencer.py.
#include <math.h>
#include <stdio.h>

#include "jingles.h"
#include "sequencer.h"

namespace {

int failures = 0;

void expect(bool ok, const char* what) {
    if (!ok) {
        printf("FAIL: %s\n", what);
        ++failures;
    }
}

bool near(float a, float b) { return fabsf(a - b) < 0.05f; }

}  // namespace

int main() {
    sequencer::Note notes[sequencer::MAX_NOTES];

    size_t n = sequencer::parse("C5:200 G5:200 R:100 C6:400", notes, sequencer::MAX_NOTES);
    expect(n == 4, "four tokens parse to four notes");
    expect(near(notes[0].hz, 523.25f), "C5 is 523.25 Hz");
    expect(near(notes[1].hz, 783.99f), "G5 is 783.99 Hz");
    expect(notes[2].hz == 0.0f && notes[2].ms == 100, "R:100 is a 100 ms rest");
    expect(near(notes[3].hz, 1046.50f) && notes[3].ms == 400, "C6:400");
    expect(sequencer::totalMs(notes, n) == 900, "total is 900 ms");

    n = sequencer::parse("A4:100", notes, sequencer::MAX_NOTES);
    expect(n == 1 && near(notes[0].hz, 440.0f), "A4 is concert pitch");

    n = sequencer::parse("F#4:100 Gb4:100 Bb4:100 c5:100", notes, sequencer::MAX_NOTES);
    expect(n == 4, "sharps, flats and lower case parse");
    expect(near(notes[0].hz, notes[1].hz), "F# and Gb are enharmonic");
    expect(near(notes[2].hz, 466.16f), "Bb4 is 466.16 Hz");
    expect(near(notes[3].hz, 523.25f), "lower-case c5 works");

    n = sequencer::parse("C5:200 bogus X9:100 C5:0 C5:99999 D5:50", notes, sequencer::MAX_NOTES);
    expect(n == 2, "bad tokens are skipped, good ones kept");
    expect(notes[1].ms == 50, "the last good note survives");

    n = sequencer::parse("", notes, sequencer::MAX_NOTES);
    expect(n == 0, "empty string is zero notes");

    n = sequencer::parse("C5:100,D5:100,\nE5:100", notes, sequencer::MAX_NOTES);
    expect(n == 3, "commas and newlines separate tokens");

    for (size_t i = 0; i < jingles::COUNT; ++i) {
        n = sequencer::parse(jingles::ALL[i].notes, notes, sequencer::MAX_NOTES);
        char what[64];
        snprintf(what, sizeof(what), "jingle '%s' parses fully", jingles::ALL[i].name);
        size_t tokens = 1;
        for (const char* p = jingles::ALL[i].notes; *p; ++p) {
            if (*p == ' ') ++tokens;
        }
        expect(n == tokens, what);
    }
    expect(jingles::find("tada") != nullptr, "tada exists");
    expect(jingles::find("nope") == nullptr, "unknown jingle is null");
    expect(jingles::find(nullptr) == nullptr, "null name is null");

    if (failures == 0) {
        printf("sequencer parser: all checks passed\n");
    }
    return failures == 0 ? 0 : 1;
}
