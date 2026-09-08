// ESP-NOW bridge: a spare stick on the router's USB port. Reads one command
// per line from USB serial and broadcasts it as an ESP-NOW frame on the
// event channel, so sticks whose MQTT session has dropped still get the room
// lock and the fire signal. See docs/messaging.md, "ESP-NOW".
//
//   lock | unlock | fire <epoch>     -> broadcast, acked on serial as "ack ..."
//   ?                                -> status
//
// Build and flash with: pio run -e bridge -t upload --upload-port <port>

#include <M5Unified.h>
#include <WiFi.h>
#include <esp_now.h>
#include <esp_wifi.h>
#include <string.h>

#include "env_config.h"

namespace {

constexpr const char* MAGIC = "SHOWCASE1 ";
constexpr int REPEATS = 3;      // broadcast frames are unacknowledged; say it thrice
constexpr int REPEAT_GAP_MS = 25;

const uint8_t BROADCAST[6] = {0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF};
uint32_t sent = 0;
char lastCommand[40] = "none";
char line[80];
size_t lineLength = 0;

void draw() {
    M5.Display.fillScreen(TFT_BLACK);
    M5.Display.setTextDatum(top_left);
    M5.Display.setFont(&fonts::Font2);
    M5.Display.setTextColor(TFT_CYAN);
    M5.Display.drawString("ESP-NOW BRIDGE", 4, 4);
    M5.Display.setTextColor(TFT_WHITE);
    M5.Display.setCursor(4, 26);
    M5.Display.printf("channel : %d\n", ESPNOW_CHANNEL);
    M5.Display.printf("sent    : %lu\n", static_cast<unsigned long>(sent));
    M5.Display.printf("last    : %s\n", lastCommand);
    M5.Display.printf("serial  : 115200\n");
}

bool broadcast(const char* body) {
    char frame[64];
    const int n = snprintf(frame, sizeof(frame), "%s%s", MAGIC, body);
    if (n <= 0 || n >= static_cast<int>(sizeof(frame))) {
        return false;
    }
    bool ok = true;
    for (int i = 0; i < REPEATS; ++i) {
        if (esp_now_send(BROADCAST, reinterpret_cast<const uint8_t*>(frame), n) != ESP_OK) {
            ok = false;
        }
        delay(REPEAT_GAP_MS);
    }
    return ok;
}

void handle(const char* command) {
    if (command[0] == '?' || command[0] == '\0') {
        Serial.printf("bridge channel=%d sent=%lu last=%s\n", ESPNOW_CHANNEL,
                      static_cast<unsigned long>(sent), lastCommand);
        return;
    }
    const bool known = strcmp(command, "lock") == 0 || strcmp(command, "unlock") == 0 ||
                       strncmp(command, "fire ", 5) == 0;
    if (!known) {
        Serial.printf("err unknown command: %s\n", command);
        return;
    }
    const bool ok = broadcast(command);
    if (ok) {
        ++sent;
        strncpy(lastCommand, command, sizeof(lastCommand) - 1);
        lastCommand[sizeof(lastCommand) - 1] = '\0';
    }
    Serial.printf("%s %s\n", ok ? "ack" : "err", command);
    draw();
}

}  // namespace

void setup() {
    Serial.begin(115200);
    auto cfg = M5.config();
    M5.begin(cfg);
    M5.Display.setRotation(1);
    M5.Display.setBrightness(120);

    WiFi.mode(WIFI_STA);
    WiFi.disconnect();
    esp_wifi_set_channel(ESPNOW_CHANNEL, WIFI_SECOND_CHAN_NONE);
    if (esp_now_init() != ESP_OK) {
        Serial.println("err esp_now_init failed");
    }
    esp_now_peer_info_t peer = {};
    memcpy(peer.peer_addr, BROADCAST, 6);
    peer.channel = ESPNOW_CHANNEL;
    peer.ifidx = WIFI_IF_STA;
    peer.encrypt = false;
    esp_now_add_peer(&peer);

    Serial.printf("bridge ready channel=%d\n", ESPNOW_CHANNEL);
    draw();
}

void loop() {
    M5.update();
    while (Serial.available()) {
        const char c = static_cast<char>(Serial.read());
        if (c == '\n' || c == '\r') {
            if (lineLength > 0) {
                line[lineLength] = '\0';
                handle(line);
                lineLength = 0;
            }
        } else if (lineLength + 1 < sizeof(line)) {
            line[lineLength++] = c;
        }
    }
    // Button A re-sends the last command, for a bench test without a host.
    if (M5.BtnA.wasClicked() && strcmp(lastCommand, "none") != 0) {
        handle(lastCommand);
    }
    delay(5);
}
