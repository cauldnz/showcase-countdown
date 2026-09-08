#pragma once

#include <math.h>
#include <stddef.h>
#include <stdint.h>
#include <string.h>

// Note-string parser for team audio. Free of Arduino dependencies so it can be
// compiled and tested on the host.
//
// Format: whitespace-separated tokens, each NAME[#|b]OCTAVE:MS or R:MS.
//   "C5:200 G5:200 R:100 C6:400"   "F#4:150 Bb4:150"
// Octaves 0-8. Durations 1-60000 ms. Unknown tokens are skipped rather than
// failing the whole string, so a small typo costs one note, not the tune.

namespace sequencer {

struct Note {
    float hz;     // 0 = rest
    uint16_t ms;
};

constexpr size_t MAX_NOTES = 96;

inline int semitoneOf(char letter) {
    switch (letter) {
        case 'C': case 'c': return 0;
        case 'D': case 'd': return 2;
        case 'E': case 'e': return 4;
        case 'F': case 'f': return 5;
        case 'G': case 'g': return 7;
        case 'A': case 'a': return 9;
        case 'B': case 'b': return 11;
        default: return -1;
    }
}

inline float hzForMidi(int midi) {
    return 440.0f * powf(2.0f, (midi - 69) / 12.0f);
}

// Parses one token. Returns false if it is not a note or rest.
inline bool parseToken(const char* token, size_t length, Note* out) {
    if (length < 3) {
        return false;
    }
    const char* colon = static_cast<const char*>(memchr(token, ':', length));
    if (colon == nullptr) {
        return false;
    }
    long ms = 0;
    for (const char* p = colon + 1; p < token + length; ++p) {
        if (*p < '0' || *p > '9') return false;
        ms = ms * 10 + (*p - '0');
        if (ms > 60000) return false;
    }
    if (ms <= 0) {
        return false;
    }
    out->ms = static_cast<uint16_t>(ms);

    const size_t nameLength = static_cast<size_t>(colon - token);
    if (nameLength == 1 && (token[0] == 'R' || token[0] == 'r')) {
        out->hz = 0.0f;
        return true;
    }
    const int semitone = semitoneOf(token[0]);
    if (semitone < 0) {
        return false;
    }
    size_t i = 1;
    int accidental = 0;
    if (i < nameLength && (token[i] == '#' || token[i] == 's')) {
        accidental = 1;
        ++i;
    } else if (i < nameLength && token[i] == 'b') {
        accidental = -1;
        ++i;
    }
    if (i + 1 != nameLength || token[i] < '0' || token[i] > '8') {
        return false;
    }
    const int octave = token[i] - '0';
    out->hz = hzForMidi((octave + 1) * 12 + semitone + accidental);
    return true;
}

// Fills notes[] from text; returns the count. Stops at MAX_NOTES.
inline size_t parse(const char* text, Note* notes, size_t capacity) {
    size_t count = 0;
    const char* p = text;
    while (*p && count < capacity) {
        while (*p == ' ' || *p == '\t' || *p == '\n' || *p == ',' || *p == '\r') {
            ++p;
        }
        if (*p == '\0') {
            break;
        }
        const char* start = p;
        while (*p && *p != ' ' && *p != '\t' && *p != '\n' && *p != ',' && *p != '\r') {
            ++p;
        }
        Note note;
        if (parseToken(start, static_cast<size_t>(p - start), &note)) {
            notes[count++] = note;
        }
    }
    return count;
}

inline uint32_t totalMs(const Note* notes, size_t count) {
    uint32_t total = 0;
    for (size_t i = 0; i < count; ++i) {
        total += notes[i].ms;
    }
    return total;
}

}  // namespace sequencer
