package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"go.bug.st/serial"
)

// Bridge talks to the ESP-NOW relay stick over USB serial. Lines are the
// whole protocol: "lock", "unlock", "fire <epoch>". Docs: docs/messaging.md.
type Bridge struct {
	mu   sync.Mutex
	port serial.Port
	name string
}

func OpenBridge(name string) (*Bridge, error) {
	port, err := serial.Open(name, &serial.Mode{BaudRate: 115200})
	if err != nil {
		return nil, err
	}
	b := &Bridge{port: port, name: name}
	go b.drain()
	return b, nil
}

// drain logs whatever the relay prints so its acks land in the server log.
func (b *Bridge) drain() {
	buf := make([]byte, 256)
	line := ""
	for {
		n, err := b.port.Read(buf)
		if err != nil {
			log.Printf("bridge: read: %v", err)
			return
		}
		for _, c := range buf[:n] {
			if c == '\n' {
				if line != "" {
					log.Printf("bridge: %s", line)
				}
				line = ""
			} else if c != '\r' {
				line += string(c)
			}
		}
	}
}

func (b *Bridge) Send(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, err := b.port.Write([]byte(line + "\n")); err != nil {
		log.Printf("bridge: write %q: %v", line, err)
	}
}

// armFire sends the fire signal three seconds before the event so sticks
// that lost their clock still start on time. The stick ignores a fire whose
// epoch does not match its built-in target.
func (b *Bridge) armFire(epoch time.Time) {
	if epoch.IsZero() {
		return
	}
	go func() {
		wait := time.Until(epoch.Add(-3 * time.Second))
		if wait < 0 {
			return
		}
		time.Sleep(wait)
		b.Send(fmt.Sprintf("fire %d", epoch.Unix()))
		log.Printf("bridge: fire sent for %d", epoch.Unix())
	}()
}
