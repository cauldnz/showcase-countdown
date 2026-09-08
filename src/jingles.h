#pragma once

#include <string.h>

// Built-in jingles, as note strings in the same format teams compose with.
// Kept short and cheerful; the piezo and the SPK2 both handle C5-C7 well.

namespace jingles {

struct Jingle {
    const char* name;
    const char* notes;
};

const Jingle ALL[] = {
    {"ding", "E6:120"},
    {"tada", "C5:120 E5:120 G5:120 C6:360"},
    {"merge", "G5:90 C6:90 E6:90 G6:260"},
    {"coin", "B5:80 E6:320"},
    {"levelup", "C5:100 E5:100 G5:100 C6:100 E6:100 G6:300"},
    {"alarm", "A5:150 R:50 A5:150 R:50 A5:150 R:50 A5:300"},
    {"sad", "E5:250 Eb5:250 D5:250 Db5:500"},
    {"fail", "G4:200 F#4:200 F4:200 E4:500"},
    {"knock", "C5:60 R:60 C5:60 R:200 C5:60"},
    {"ok", "C6:80 G6:160"},
};

constexpr size_t COUNT = sizeof(ALL) / sizeof(ALL[0]);

inline const char* find(const char* name) {
    if (name == nullptr) {
        return nullptr;
    }
    for (size_t i = 0; i < COUNT; ++i) {
        if (strcmp(ALL[i].name, name) == 0) {
            return ALL[i].notes;
        }
    }
    return nullptr;
}

}  // namespace jingles
