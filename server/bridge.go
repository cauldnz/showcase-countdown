package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

// The ESP-NOW bridge stick relays "lock", "unlock" and "fire <epoch>" as
// broadcast frames (docs/messaging.md). It normally listens on MQTT topic
// showcase/bridge/cmd, powered from the router's USB port. A USB serial link
// is an optional second path for hosts whose kernel can drive the stick's
// USB-serial chip.

const bridgeCmdTopic = topicPrefix + "bridge/cmd"

// bridgeSend fans one command out over every available path.
func (a *App) bridgeSend(line string) {
	// Lock state is retained so a bridge that reconnects catches up; a fire
	// signal is only meaningful at the instant it is sent.
	retain := !strings.HasPrefix(line, "fire")
	if a.fleet.Connected() {
		if err := a.fleet.PublishRaw(bridgeCmdTopic, line, retain); err != nil {
			log.Printf("bridge: mqtt publish %q: %v", line, err)
		}
	}
	if a.bridge != nil {
		a.bridge.Send(line)
	}
}

// armFire sends the fire signal three seconds before the event so sticks
// that lost their clock still start on time. A stick ignores a fire whose
// epoch does not match its built-in target.
func (a *App) armFire() {
	if a.epoch.IsZero() {
		return
	}
	go func() {
		wait := time.Until(a.epoch.Add(-3 * time.Second))
		if wait < 0 {
			return
		}
		time.Sleep(wait)
		line := fmt.Sprintf("fire %d", a.epoch.Unix())
		a.bridgeSend(line)
		a.bus.Emit(Event{Kind: "lock", Text: "fire signal sent to the bridge"})
		log.Printf("bridge: %s", line)
	}()
}

// SerialBridge is the optional USB path.
type SerialBridge struct {
	mu   sync.Mutex
	port serial.Port
	name string
}

func OpenSerialBridge(name string) (*SerialBridge, error) {
	port, err := serial.Open(name, &serial.Mode{BaudRate: 115200})
	if err != nil {
		return nil, err
	}
	b := &SerialBridge{port: port, name: name}
	go b.drain()
	return b, nil
}

// drain logs whatever the relay prints so its acks land in the server log.
func (b *SerialBridge) drain() {
	buf := make([]byte, 256)
	line := ""
	for {
		n, err := b.port.Read(buf)
		if err != nil {
			log.Printf("bridge serial: read: %v", err)
			return
		}
		for _, c := range buf[:n] {
			if c == '\n' {
				if line != "" {
					log.Printf("bridge serial: %s", line)
				}
				line = ""
			} else if c != '\r' {
				line += string(c)
			}
		}
	}
}

func (b *SerialBridge) Send(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, err := b.port.Write([]byte(line + "\n")); err != nil {
		log.Printf("bridge serial: write %q: %v", line, err)
	}
}
