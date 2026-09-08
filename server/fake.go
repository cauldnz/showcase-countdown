package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// A fakeStick is a virtual device on the broker: same topics, same state
// message, same retained config, so the server and dashboard cannot tell it
// from hardware. It does not render; the dashboard draws its screen from the
// commands the server sees.
type fakeStick struct {
	id     string
	code   int
	voice  string
	client mqtt.Client
	mu     sync.Mutex
	team   string
	fired  bool
}

var fakes = struct {
	mu    sync.Mutex
	items map[string]*fakeStick
}{items: map[string]*fakeStick{}}

func fakeCode(id string) int {
	h := fnv.New32a()
	h.Write([]byte(id + "fake"))
	return 1000 + int(h.Sum32()%9000)
}

// startFakes brings up n virtual sticks named fake01..fakeNN.
func startFakes(brokerURL string, n int) {
	voices := []string{"root", "fifth", "melody"}
	for i := 1; i <= n; i++ {
		id := fmt.Sprintf("fake%02d", i)
		f := &fakeStick{id: id, code: fakeCode(id), voice: voices[i%3]}
		fakes.mu.Lock()
		fakes.items[id] = f
		fakes.mu.Unlock()
		go f.run(brokerURL)
	}
}

func (f *fakeStick) stateTopic() string { return fmt.Sprintf("%sdev/%s/state", topicPrefix, f.id) }

func (f *fakeStick) run(brokerURL string) {
	stateTopic := f.stateTopic()
	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID("stick-" + f.id).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetWill(stateTopic, `{"online":false}`, 1, true)
	opts.OnConnect = func(c mqtt.Client) {
		c.Subscribe(fmt.Sprintf("%sdev/%s/cmd/#", topicPrefix, f.id), 1, f.onCommand)
		c.Subscribe(topicPrefix+"all/cmd/#", 1, f.onCommand)
		f.publishState()
	}
	f.client = mqtt.NewClient(opts)
	if t := f.client.Connect(); t.Wait() && t.Error() != nil {
		log.Printf("fake %s: connect: %v", f.id, t.Error())
	}
	for range time.Tick(10 * time.Second) {
		f.publishState()
	}
}

func (f *fakeStick) publishState() {
	if f.client == nil || !f.client.IsConnectionOpen() {
		return
	}
	f.mu.Lock()
	team, fired := f.team, f.fired
	f.mu.Unlock()
	payload, _ := json.Marshal(stateMsg{
		Online: true, ID: f.id, Voice: f.voice, Battery: 100, NTP: true, Fired: fired,
		Code: f.code, Team: team, Uptime: int(time.Since(processStart).Seconds()),
		Heap: 150000, FW: "fake",
	})
	f.client.Publish(f.stateTopic(), 1, true, payload)
}

func (f *fakeStick) onCommand(_ mqtt.Client, m mqtt.Message) {
	if strings.HasSuffix(m.Topic(), "/config") {
		var c Config
		if json.Unmarshal(m.Payload(), &c) == nil && c.Team != nil {
			f.mu.Lock()
			f.team = *c.Team
			f.mu.Unlock()
			f.publishState()
		}
	}
}

// pressButton publishes a button event exactly as firmware would.
func (f *fakeStick) pressButton(button, action string) {
	if f.client == nil {
		return
	}
	payload, _ := json.Marshal(map[string]string{"button": button, "action": action})
	f.client.Publish(devTopic(f.id, "event", "button"), 0, false, payload)
}

func fakeByID(id string) *fakeStick {
	fakes.mu.Lock()
	defer fakes.mu.Unlock()
	return fakes.items[id]
}

var processStart = time.Now()
