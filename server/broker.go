package main

import (
	"log"
	"net"
	"strings"

	mqttsrv "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
)

// startEmbeddedBroker runs an MQTT broker inside this process. GL.iNet's
// package feeds do not carry Mosquitto, so the router gets one binary that is
// broker, MCP server and dashboard together. Anonymous, LAN only: the event
// WiFi is our own network and the sticks carry no credentials.
func startEmbeddedBroker(addr string) (*mqttsrv.Server, error) {
	// ":1883" would bind IPv6-only on some kernels (seen on the GL-MT3000),
	// leaving IPv4 clients, including our own loopback connection, unable to
	// reach it. The sticks are IPv4, so bind IPv4 explicitly.
	if strings.HasPrefix(addr, ":") {
		addr = "0.0.0.0" + addr
	}
	srv := mqttsrv.New(&mqttsrv.Options{InlineClient: false})
	if err := srv.AddHook(new(auth.AllowHook), nil); err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		return nil, err
	}
	if err := srv.AddListener(listeners.NewNet("tcp4", ln)); err != nil {
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
