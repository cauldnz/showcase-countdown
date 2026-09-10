#include "espnow_link.h"

#include <Arduino.h>
#include <esp_now.h>
#include <esp_wifi.h>
#include <stdlib.h>
#include <string.h>

namespace espnow_link {
namespace {

constexpr const char* MAGIC = "SHOWCASE1 ";

bool started = false;
volatile uint32_t received = 0;

// One slot per message kind. The callback runs on the WiFi task; the loop
// task drains with take*(). A newer frame simply replaces an unread one.
volatile bool lockPending = false;
volatile bool lockValue = false;
volatile bool firePending = false;
volatile int64_t fireEpoch = 0;

void onReceive(const uint8_t* mac, const uint8_t* data, int length) {
    const size_t magicLength = strlen(MAGIC);
    // Diagnostic: every frame, so a channel or sender problem is visible.
    Serial.printf("  espnow  : frame from %02X:%02X:%02X len %d\n", mac[3], mac[4], mac[5], length);
    if (length <= static_cast<int>(magicLength) || length > 64) {
        return;
    }
    if (memcmp(data, MAGIC, magicLength) != 0) {
        return;
    }
    char body[64];
    const size_t bodyLength = static_cast<size_t>(length) - magicLength;
    memcpy(body, data + magicLength, bodyLength);
    body[bodyLength] = '\0';
    ++received;

    if (strcmp(body, "lock") == 0) {
        lockValue = true;
        lockPending = true;
    } else if (strcmp(body, "unlock") == 0) {
        lockValue = false;
        lockPending = true;
    } else if (strncmp(body, "fire ", 5) == 0) {
        fireEpoch = strtoll(body + 5, nullptr, 10);
        firePending = true;
    }
}

}  // namespace

void begin(uint8_t channel, bool associated) {
    if (!associated && channel >= 1 && channel <= 14) {
        esp_wifi_set_channel(channel, WIFI_SECOND_CHAN_NONE);
    }
    if (started) {
        return;
    }
    if (esp_now_init() != ESP_OK) {
        Serial.println("  espnow  : init FAILED");
        return;
    }
    esp_now_register_recv_cb(onReceive);
    started = true;
    Serial.printf("  espnow  : listening%s\n", associated ? " on the AP channel" : "");
}

bool takeLock(bool* locked) {
    if (!lockPending) {
        return false;
    }
    lockPending = false;
    *locked = lockValue;
    return true;
}

bool takeFire(int64_t* epoch) {
    if (!firePending) {
        return false;
    }
    firePending = false;
    *epoch = fireEpoch;
    return true;
}

uint32_t framesReceived() { return received; }

}  // namespace espnow_link
