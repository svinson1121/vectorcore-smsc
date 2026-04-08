package main

import (
	"strings"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/api"
	"github.com/svinson1121/vectorcore-smsc/internal/diameter"
	dcodec "github.com/svinson1121/vectorcore-smsc/internal/diameter/codec"
	"github.com/svinson1121/vectorcore-smsc/internal/diameter/sgd"
)

var diameterAppOrder = []string{"sh", "sgd", "s6c"}

type diameterRuntimePeer struct {
	Name         string
	State        string
	ConnectedAt  *time.Time
	Applications []string
}

func aggregateDiameterRuntimePeers(sgdPeers []sgd.PeerStatus, hssPeers []*diameter.Peer) []diameterRuntimePeer {
	merged := make(map[string]*diameterRuntimePeer, len(sgdPeers)+len(hssPeers))

	for _, ps := range sgdPeers {
		mergeDiameterRuntimePeer(merged, ps.Name, ps.State, ps.ConnectedAt, append(parseDiameterApplications(ps.Application), "sgd"))
	}
	for _, activeHSS := range hssPeers {
		if activeHSS == nil {
			continue
		}
		mergeDiameterRuntimePeer(
			merged,
			activeHSS.Name(),
			activeHSS.State().String(),
			activeHSS.ConnectedAt(),
			diameterApplicationsFromConfig(activeHSS.Config()),
		)
	}

	out := make([]diameterRuntimePeer, 0, len(merged))
	for _, peer := range merged {
		out = append(out, *peer)
	}
	return out
}

func mergeDiameterRuntimePeer(merged map[string]*diameterRuntimePeer, name, state string, connectedAt *time.Time, applications []string) {
	if name == "" {
		return
	}
	current, ok := merged[name]
	if !ok {
		current = &diameterRuntimePeer{Name: name}
		merged[name] = current
	}

	if current.State != "OPEN" || state == "OPEN" {
		current.State = state
	}
	current.ConnectedAt = earliestTime(current.ConnectedAt, connectedAt)
	current.Applications = mergeDiameterApplications(current.Applications, applications...)
}

func mergeDiameterApplications(current []string, incoming ...string) []string {
	seen := make(map[string]struct{}, len(current)+len(incoming))
	for _, app := range current {
		seen[app] = struct{}{}
	}
	for _, app := range incoming {
		if app == "" {
			continue
		}
		if _, ok := seen[app]; ok {
			continue
		}
		seen[app] = struct{}{}
	}

	out := make([]string, 0, len(seen))
	for _, app := range diameterAppOrder {
		if _, ok := seen[app]; ok {
			out = append(out, app)
			delete(seen, app)
		}
	}
	return out
}

func diameterApplicationsFromConfig(cfg diameter.Config) []string {
	apps := parseDiameterApplications(cfg.Application)
	for _, appID := range cfg.AppIDs {
		if app := diameterApplicationFromID(appID); app != "" {
			apps = append(apps, app)
		}
	}
	if app := diameterApplicationFromID(cfg.AppID); app != "" {
		apps = append(apps, app)
	}
	return mergeDiameterApplications(nil, apps...)
}

func diameterApplicationFromID(appID uint32) string {
	switch appID {
	case dcodec.App3GPP_Sh:
		return "sh"
	case dcodec.App3GPP_SGd:
		return "sgd"
	case dcodec.App3GPP_S6c:
		return "s6c"
	default:
		return ""
	}
}

func parseDiameterApplications(raw string) []string {
	fields := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(raw)), func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		switch field {
		case "sh", "sgd", "s6c":
			out = append(out, field)
		}
	}
	return out
}

func earliestTime(current, candidate *time.Time) *time.Time {
	if current == nil {
		return candidate
	}
	if candidate == nil {
		return current
	}
	if candidate.Before(*current) {
		return candidate
	}
	return current
}

func diameterRuntimePeerInfo(peers []diameterRuntimePeer) []api.PeerInfo {
	out := make([]api.PeerInfo, 0, len(peers))
	for _, peer := range peers {
		out = append(out, api.PeerInfo{
			Name:        peer.Name,
			Type:        "diameter_peer",
			State:       peer.State,
			Application: strings.Join(formatDiameterApplications(peer.Applications), " "),
			ConnectedAt: peer.ConnectedAt,
		})
	}
	return out
}

func formatDiameterApplications(apps []string) []string {
	out := make([]string, 0, len(apps))
	for _, app := range apps {
		switch app {
		case "sh":
			out = append(out, "Sh")
		case "sgd":
			out = append(out, "SGd")
		case "s6c":
			out = append(out, "S6c")
		}
	}
	return out
}
