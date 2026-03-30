package gateway

import (
	"fmt"
	"strings"
)

// SessionKeyMain returns the key for an agent's primary (non-DM) session.
func SessionKeyMain(agentName string) string {
	return "agent:" + NormalizeKey(agentName) + ":main"
}

// SessionKeyDM returns the key for a direct-message session.
func SessionKeyDM(agentName, channel, peerID string) string {
	return "agent:" + NormalizeKey(agentName) + ":" + NormalizeKey(channel) + ":direct:" + NormalizeKey(peerID)
}

// SessionKeyGroup returns the key for a group-chat session.
func SessionKeyGroup(agentName, channel, groupID string) string {
	return "agent:" + NormalizeKey(agentName) + ":" + NormalizeKey(channel) + ":group:" + NormalizeKey(groupID)
}

// SessionKeyChannel returns the key for a channel-level session.
func SessionKeyChannel(agentName, channel, channelID string) string {
	return "agent:" + NormalizeKey(agentName) + ":" + NormalizeKey(channel) + ":channel:" + NormalizeKey(channelID)
}

// SessionKeyThread appends a thread qualifier to a base session key.
func SessionKeyThread(baseKey, threadID string) string {
	return baseKey + ":thread:" + NormalizeKey(threadID)
}

// SessionKeyCron returns the key for a cron job session.
func SessionKeyCron(agentName, jobID string) string {
	return "agent:" + NormalizeKey(agentName) + ":cron:" + NormalizeKey(jobID)
}

// SessionKeyCronRun returns the key for a single cron run.
func SessionKeyCronRun(agentName, jobID, runUUID string) string {
	return SessionKeyCron(agentName, jobID) + ":run:" + NormalizeKey(runUUID)
}

// NormalizeKey lowercases and trims whitespace from a key component.
func NormalizeKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ParsedSessionKey holds the decomposed parts of a structured session key.
type ParsedSessionKey struct {
	AgentName string
	Scope     string // "main" | "direct" | "group" | "channel" | "cron"
	Channel   string
	PeerID    string
	ThreadID  string
	JobID     string
	RunID     string
}

// ParseSessionKey decomposes a structured session key into its components.
// Format: agent:{name}:{rest...}
func ParseSessionKey(key string) (ParsedSessionKey, error) {
	parts := strings.Split(key, ":")
	if len(parts) < 3 || parts[0] != "agent" {
		return ParsedSessionKey{}, fmt.Errorf("gateway: invalid session key %q", key)
	}
	p := ParsedSessionKey{AgentName: parts[1]}
	switch parts[2] {
	case "main":
		p.Scope = "main"
	case "cron":
		p.Scope = "cron"
		if len(parts) >= 4 {
			p.JobID = parts[3]
		}
		if len(parts) >= 6 && parts[4] == "run" {
			p.RunID = parts[5]
		}
	default:
		// agent:{name}:{channel}:{scope}:{id}[:thread:{threadID}]
		p.Channel = parts[2]
		if len(parts) < 5 {
			return ParsedSessionKey{}, fmt.Errorf("gateway: invalid session key %q", key)
		}
		p.Scope = parts[3]
		p.PeerID = parts[4]
		for i := 5; i+1 < len(parts); i += 2 {
			if parts[i] == "thread" {
				p.ThreadID = parts[i+1]
			}
		}
	}
	return p, nil
}
