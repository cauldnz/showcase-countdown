// showcase-server: the MCP server, dashboard and broker bridge for the sticks.
// See docs/plan.md and docs/messaging.md.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	var (
		listen  = flag.String("listen", envOr("LISTEN", ":8080"), "HTTP listen address")
		broker  = flag.String("broker", envOr("MQTT_URL", "tcp://127.0.0.1:1883"), "MQTT broker URL")
		secret  = flag.String("secret", os.Getenv("ORGANISER_SECRET"), "organiser secret (env ORGANISER_SECRET)")
		event   = flag.String("event", envOr("EVENT_DATETIME", ""), "event instant, RFC3339 with offset, or unix epoch")
		title   = flag.String("title", envOr("EVENT_NAME", "Countdown"), "event name for the dashboard")
		fakeN   = flag.Int("fake", envInt("FAKE_STICKS", 0), "number of virtual sticks to run")
		bridgeP = flag.String("bridge", os.Getenv("BRIDGE_PORT"), "optional serial port of the ESP-NOW relay stick, e.g. COM5 or /dev/ttyUSB0 (MQTT is the default path)")
		embed   = flag.String("embedded-broker", envOr("EMBEDDED_BROKER", ""), "run an MQTT broker in-process on this address, e.g. :1883, and connect to it")
		teams   = flag.String("teams-file", envOr("TEAMS_FILE", ""), "JSON file that keeps team names across restarts, e.g. /etc/showcase/teams.json")
	)
	flag.Parse()

	epoch, err := parseEvent(*event)
	if err != nil {
		log.Fatalf("bad -event %q: %v", *event, err)
	}

	bus := NewBus(500)
	fleet := NewFleet(bus)
	if n := fleet.LoadTeams(*teams); n > 0 {
		log.Printf("teams: loaded %d from %s", n, *teams)
	}
	fleet.onTeamsChanged = func() { fleet.SaveTeams(*teams) }
	app := &App{
		fleet: fleet, policy: NewPolicy(epoch), bus: bus,
		secret: *secret, title: *title, epoch: epoch,
	}

	if *bridgeP != "" {
		b, err := OpenSerialBridge(*bridgeP)
		if err != nil {
			log.Printf("bridge: %v (continuing without it)", err)
		} else {
			app.bridge = b
			log.Printf("bridge: serial relay on %s", *bridgeP)
		}
	}
	app.armFire()

	if *embed != "" {
		if _, err := startEmbeddedBroker(*embed); err != nil {
			log.Fatalf("embedded broker: %v", err)
		}
		// Connect to our own broker over IPv4 loopback whatever it binds to.
		_, port, _ := strings.Cut(*embed, ":")
		*broker = "tcp://127.0.0.1:" + port
		time.Sleep(200 * time.Millisecond)
	}

	// Start HTTP before the broker connection: fleet.Connect retries until it
	// succeeds, and the dashboard must be reachable even while it does.
	mcpHandler := app.mcpHandler()

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpHandler)
	// /mcp/<code> lets a team connect with nothing but a URL. The code is
	// copied into the header the tools read, so both forms behave the same.
	mux.HandleFunc("/mcp/{code}", func(w http.ResponseWriter, r *http.Request) {
		if code := r.PathValue("code"); code != "" {
			r.Header.Set(headerCode, code)
		}
		mcpHandler.ServeHTTP(w, r)
	})
	app.dashboardRoutes(mux)

	srv := &http.Server{Addr: *listen, Handler: logRequests(mux)}
	// IPv4 explicitly: on the GL-MT3000 a dual-stack socket never receives
	// IPv4 connections, and every stick and laptop in the room is IPv4.
	ln, err := net.Listen("tcp4", *listen)
	if err != nil {
		log.Fatalf("listen %s: %v", *listen, err)
	}
	go func() {
		log.Printf("listening on %s  (MCP at /mcp/<code>, dashboard at /)", ln.Addr())
		if err := srv.Serve(ln); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	if !epoch.IsZero() {
		log.Printf("event at %s (%d)", epoch.Format(time.RFC3339), epoch.Unix())
	}
	if *secret == "" {
		log.Printf("warning: no organiser secret; organiser tools are disabled")
	}

	if err := fleet.Connect(*broker); err != nil {
		log.Printf("mqtt: initial connect failed: %v (retrying in background)", err)
	}
	if *fakeN > 0 {
		startFakes(*broker, *fakeN)
		log.Printf("fake fleet: %d virtual sticks", *fakeN)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}

func (a *App) mcpHandler() http.Handler {
	server := a.mcpServer()
	// Identity is per request; stateless transport also accepts client protocol metadata.
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true})
}

func envInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return v
	}
	return def
}

func parseEvent(s string) (time.Time, error) {
	s = strings.TrimSpace(strings.Trim(s, `"`))
	if s == "" {
		return time.Time{}, nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(n, 0), nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("want RFC3339 with offset or a unix epoch: %w", err)
	}
	return t, nil
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/events") {
			log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}
