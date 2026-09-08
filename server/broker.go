package main

import (
	"log"

	mqttsrv "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
)

// startEmbeddedBroker runs an MQTT broker inside this process. GL.iNet's
// package feeds do not carry Mosquitto, so the router gets one binary that is
// broker, MCP server and dashboard together. Anonymous, LAN only: the event
// WiFi is our own network and the sticks carry no credentials.
func startEmbeddedBroker(addr string) (*mqttsrv.Server, error) {
	srv := mqttsrv.New(&mqttsrv.Options{InlineClient: false})
	if err := srv.AddHook(new(auth.AllowHook), nil); err != nil {
		return nil, err
	}
	if err := srv.AddListener(listeners.NewTCP(listeners.Config{ID: "tcp", Address: addr})); err != nil {
		return nil, err
	}
	go func() {
		if err := srv.Serve(); err != nil {
			log.Printf("broker: %v", err)
		}
	}()
	log.Printf("broker: embedded MQTT listening on %s", addr)
	return srv, nil
}
