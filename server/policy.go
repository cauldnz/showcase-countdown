package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Limits from docs/plan.md decision 10. Audio is deliberately uncapped.
const (
	defaultShoutCooldown = 120 * time.Second
	dmCooldown           = 5 * time.Second
	maxTextLen           = 120
	maxTTL               = 60
	lockLead             = 60 * time.Second
	fanfareLen           = 12 * time.Second
	inboxSize            = 50
)

// Message is one entry in a team's inbox.
type Message struct {
	At   time.Time `json:"at"`
	From string    `json:"from"`
	Kind string    `json:"kind"` // dm or shout
	Text string    `json:"text"`
}

// Policy holds every social rule: cooldowns, mute, lock, and inboxes. Keyed by
// device id, since the claim code is the identity and a device is a team.
type Policy struct {
	mu            sync.Mutex
	eventEpoch    time.Time
	manualLock    bool
	shoutCooldown time.Duration
	lastShout     map[string]time.Time
	lastDM        map[string]time.Time
	muted         map[string]bool
	inbox         map[string][]Message
}

func NewPolicy(eventEpoch time.Time) *Policy {
	return &Policy{
		eventEpoch:    eventEpoch,
		shoutCooldown: defaultShoutCooldown,
		lastShout:     map[string]time.Time{},
		lastDM:        map[string]time.Time{},
		muted:         map[string]bool{},
		inbox:         map[string][]Message{},
	}
}

// SetShoutCooldown changes the per-team shout gap live. Zero disables it.
func (p *Policy) SetShoutCooldown(d time.Duration) {
	p.mu.Lock()
	p.shoutCooldown = d
	p.mu.Unlock()
}

func (p *Policy) ShoutCooldown() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.shoutCooldown
}

// Locked reports whether team commands are refused right now.
func (p *Policy) Locked() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.manualLock || p.autoLocked(time.Now())
}

func (p *Policy) autoLocked(now time.Time) bool {
	if p.eventEpoch.IsZero() {
		return false
	}
	return now.After(p.eventEpoch.Add(-lockLead)) && now.Before(p.eventEpoch.Add(fanfareLen))
}

func (p *Policy) LockState() (manual, auto bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.manualLock, p.autoLocked(time.Now())
}

func (p *Policy) SetManualLock(on bool) {
	p.mu.Lock()
	p.manualLock = on
	p.mu.Unlock()
}

func (p *Policy) SetMuted(deviceID string, on bool) {
	p.mu.Lock()
	if on {
		p.muted[deviceID] = true
	} else {
		delete(p.muted, deviceID)
	}
	p.mu.Unlock()
}

func (p *Policy) Muted(deviceID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.muted[deviceID]
}

func (p *Policy) MutedIDs() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, 0, len(p.muted))
	for id := range p.muted {
		out = append(out, id)
	}
	return out
}

// Gate is the check every team command passes through.
func (p *Policy) Gate(deviceID string) error {
	if p.Locked() {
		return fmt.Errorf("the room is locked for the countdown finale; try again after the fanfare")
	}
	if p.Muted(deviceID) {
		return fmt.Errorf("this stick has been muted by the organiser")
	}
	return nil
}

func (p *Policy) CheckShout(deviceID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.shoutCooldown <= 0 {
		return nil
	}
	if wait := p.shoutCooldown - time.Since(p.lastShout[deviceID]); wait > 0 && !p.lastShout[deviceID].IsZero() {
		return fmt.Errorf("shout cooldown: try again in %d seconds", int(wait.Seconds())+1)
	}
	p.lastShout[deviceID] = time.Now()
	return nil
}

func (p *Policy) CheckDM(deviceID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if wait := dmCooldown - time.Since(p.lastDM[deviceID]); wait > 0 && !p.lastDM[deviceID].IsZero() {
		return fmt.Errorf("message cooldown: try again in %d seconds", int(wait.Seconds())+1)
	}
	p.lastDM[deviceID] = time.Now()
	return nil
}

func (p *Policy) Deliver(deviceID string, m Message) {
	if m.At.IsZero() {
		m.At = time.Now()
	}
	p.mu.Lock()
	box := append(p.inbox[deviceID], m)
	if len(box) > inboxSize {
		box = box[len(box)-inboxSize:]
	}
	p.inbox[deviceID] = box
	p.mu.Unlock()
}

// Inbox returns and clears a team's messages.
func (p *Policy) Inbox(deviceID string) []Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	box := p.inbox[deviceID]
	p.inbox[deviceID] = nil
	return box
}

// cleanText trims and caps message text without stripping colour markup.
func cleanText(s string) (string, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if s == "" {
		return "", fmt.Errorf("text is empty")
	}
	if len([]rune(s)) > maxTextLen {
		return "", fmt.Errorf("text is too long: %d characters, the limit is %d", len([]rune(s)), maxTextLen)
	}
	return s, nil
}

func clampTTL(seconds int, def int) int {
	if seconds <= 0 {
		return def
	}
	if seconds > maxTTL {
		return maxTTL
	}
	return seconds
}
