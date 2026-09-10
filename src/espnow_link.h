#pragma once

#include <stdint.h>

// ESP-NOW receiver for the two messages that must land even when a stick's
// MQTT session is gone: the room lock and the fire signal. Frames are plain
// text with a magic prefix, broadcast by the bridge stick on the AP's channel
// (docs/messaging.md, "ESP-NOW").
//
//   SHOWCASE1 lock
//   SHOWCASE1 unlock
//   SHOWCASE1 fire <epoch>

namespace espnow_link {

// Call once WiFi is in station mode. Safe to call again; later calls no-op.
// When the station is not associated, the radio is parked on `channel` so
// broadcasts from the bridge still arrive.
void begin(uint8_t channel, bool associated);

// Drain one pending frame. Each returns true at most once per frame.
bool takeLock(bool* locked);
bool takeFire(int64_t* epoch);

uint32_t framesReceived();

}  // namespace espnow_link
