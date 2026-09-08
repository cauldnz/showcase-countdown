#pragma once

#include <Arduino.h>
#include <Preferences.h>

#include "env_config.h"

// Credentials provisioned over Improv are kept in NVS so a published firmware
// image carries no secrets. A private build may still bake defaults into .env,
// which are used only when nothing has been provisioned on the device.
namespace wifi_store {

inline Preferences& store() {
    static Preferences prefs;
    return prefs;
}

// Opening a namespace read-only before it exists logs an nvs_open error, which
// reads as a fault in the boot banner. Creating it once up front avoids that.
inline void begin() {
    if (store().begin("countdown", false)) {
        store().end();
    }
}

inline String read(const char* key, const char* fallback) {
    String value;
    if (store().begin("countdown", true)) {
        // getString() on an absent key logs at error level, so ask first.
        if (store().isKey(key)) {
            value = store().getString(key, "");
        }
        store().end();
    }
    if (value.length() == 0) {
        value = fallback;
    }
    return value;
}

inline String ssid() {
    return read("ssid", WIFI_SSID);
}

inline String password() {
    return read("password", WIFI_PASSWORD);
}

inline bool configured() {
    return ssid().length() > 0;
}

// Called from the Improv callback once a connection has actually succeeded.
inline void save(const char* ssid, const char* password) {
    if (!store().begin("countdown", false)) {
        return;
    }
    store().putString("ssid", ssid);
    store().putString("password", password);
    store().end();
}

inline void clear() {
    if (!store().begin("countdown", false)) {
        return;
    }
    store().clear();
    store().end();
}

}  // namespace wifi_store
