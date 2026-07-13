package platform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	NodeKindFRPS       = "frps"
	NodeKindNarwhalNAT = "narwhal_nat"
	NATProviderNarwhal = "narwhal"
)

type NATPortForwardRequest struct {
	Node       Node
	Protocol   string
	GuestPort  int
	TunnelName string
}

type NATPortForwardResult struct {
	EntryHost  string
	HostPort   int
	GuestPort  int
	Protocol   string
	Provider   string
	InstanceID string
}

type NATPortForwarder interface {
	ForwardPort(req NATPortForwardRequest) (NATPortForwardResult, error)
}

var natPortForwarder NATPortForwarder = NarwhalForwarderFromEnv()

func SetNATPortForwarderForTest(f NATPortForwarder) func() {
	previous := natPortForwarder
	natPortForwarder = f
	return func() { natPortForwarder = previous }
}

func normalizeNodeKind(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "frps", "standard":
		return NodeKindFRPS
	case "nat", "narwhal", "narwhal_nat":
		return NodeKindNarwhalNAT
	default:
		return strings.ToLower(strings.TrimSpace(v))
	}
}

func normalizeNodeForSave(n Node) Node {
	n.NodeKind = normalizeNodeKind(n.NodeKind)
	if n.NATProvider == "" && n.NodeKind == NodeKindNarwhalNAT {
		n.NATProvider = NATProviderNarwhal
	}
	n.NATProvider = strings.ToLower(strings.TrimSpace(n.NATProvider))
	n.NATInstanceID = strings.TrimSpace(n.NATInstanceID)
	n.NATInstanceName = strings.TrimSpace(n.NATInstanceName)
	n.NATEntryHost = strings.TrimSpace(n.NATEntryHost)
	if n.NodeKind == NodeKindNarwhalNAT && strings.TrimSpace(n.ServerAddr) == "" {
		n.ServerAddr = n.NATEntryHost
	}
	return n
}

func applyNATForward(node Node, protocol string, guestPort int, tunnelName string) (string, error) {
	node = normalizeNodeForSave(node)
	if node.NodeKind != NodeKindNarwhalNAT {
		return "", nil
	}
	if strings.ToLower(protocol) != "tcp" && strings.ToLower(protocol) != "udp" {
		return "", fmt.Errorf("nat forwarding supports tcp/udp only")
	}
	if natPortForwarder == nil {
		return "", fmt.Errorf("nat port forwarder is not configured")
	}
	res, err := natPortForwarder.ForwardPort(NATPortForwardRequest{Node: node, Protocol: strings.ToLower(protocol), GuestPort: guestPort, TunnelName: tunnelName})
	if err != nil {
		return "", err
	}
	host := strings.TrimSpace(res.EntryHost)
	if host == "" {
		host = strings.TrimSpace(node.NATEntryHost)
	}
	if host == "" {
		return "", fmt.Errorf("nat entry host missing")
	}
	if res.HostPort <= 0 {
		return "", fmt.Errorf("nat host port missing")
	}
	return fmt.Sprintf("%s:%d", host, res.HostPort), nil
}

type NarwhalForwarder struct {
	BaseURL    string
	Credential string
	Client     *http.Client
}

func NarwhalForwarderFromEnv() NATPortForwarder {
	credential := strings.TrimSpace(os.Getenv("NARWHAL_API_KEY"))
	if credential == "" {
		if b, err := os.ReadFile(os.Getenv("HOME") + "/.config/narwhal-nat/credentials"); err == nil {
			credential = strings.TrimSpace(string(b))
		}
	}
	if credential == "" {
		return nil
	}
	return &NarwhalForwarder{BaseURL: getenv("NARWHAL_API_BASE", "https://api.fuckip.me/api/v1"), Credential: credential, Client: &http.Client{Timeout: 30 * time.Second}}
}

func (f *NarwhalForwarder) ForwardPort(req NATPortForwardRequest) (NATPortForwardResult, error) {
	if f.Client == nil {
		f.Client = &http.Client{Timeout: 30 * time.Second}
	}
	vmID := strings.TrimSpace(req.Node.NATInstanceID)
	if vmID == "" {
		vm, err := f.resolveInstance(req.Node.NATInstanceName)
		if err != nil {
			return NATPortForwardResult{}, err
		}
		vmID = fmt.Sprint(vm["id"])
		if req.Node.NATEntryHost == "" {
			req.Node.NATEntryHost = fmt.Sprint(vm["machine_entry_host"])
		}
	}
	forwards, err := f.forwards(vmID)
	if err != nil {
		return NATPortForwardResult{}, err
	}
	protocol := strings.ToLower(strings.TrimSpace(req.Protocol))
	for _, mapping := range forwards {
		if strings.ToLower(fmt.Sprint(mapping["protocol"])) == protocol && intFromAny(mapping["guest_port"]) == req.GuestPort {
			return NATPortForwardResult{EntryHost: req.Node.NATEntryHost, HostPort: intFromAny(mapping["host_port"]), GuestPort: req.GuestPort, Protocol: protocol, Provider: NATProviderNarwhal, InstanceID: vmID}, nil
		}
	}
	used := map[int]bool{}
	for _, mapping := range forwards {
		used[intFromAny(mapping["host_port"])] = true
	}
	var last error
	for i := 0; i < 10; i++ {
		hostPort := chooseNATPort(used)
		used[hostPort] = true
		payload := map[string]any{"protocol": protocol, "host_port": hostPort, "guest_port": req.GuestPort, "description": strings.TrimSpace(req.TunnelName)}
		if _, err := f.request("POST", "/vms/"+vmID+"/port-forwards", payload); err != nil {
			last = err
			text := strings.ToLower(err.Error())
			if !(strings.Contains(text, "port") || strings.Contains(text, "exist") || strings.Contains(text, "used") || strings.Contains(text, "occupied") || strings.Contains(text, "conflict")) {
				return NATPortForwardResult{}, err
			}
			continue
		}
		verify, err := f.forwards(vmID)
		if err != nil {
			return NATPortForwardResult{}, err
		}
		for _, mapping := range verify {
			if strings.ToLower(fmt.Sprint(mapping["protocol"])) == protocol && intFromAny(mapping["guest_port"]) == req.GuestPort && intFromAny(mapping["host_port"]) == hostPort {
				return NATPortForwardResult{EntryHost: req.Node.NATEntryHost, HostPort: hostPort, GuestPort: req.GuestPort, Protocol: protocol, Provider: NATProviderNarwhal, InstanceID: vmID}, nil
			}
		}
		last = fmt.Errorf("nat mapping read-back verification failed")
	}
	return NATPortForwardResult{}, fmt.Errorf("allocate nat remote port: %w", last)
}

func (f *NarwhalForwarder) resolveInstance(query string) (map[string]any, error) {
	data, err := f.request("GET", "/vms?page=1&page_size=100", nil)
	if err != nil {
		return nil, err
	}
	root, _ := data.(map[string]any)
	items, _ := root["vms"].([]any)
	var matches []map[string]any
	needle := strings.ToLower(strings.TrimSpace(query))
	for _, item := range items {
		vm, ok := item.(map[string]any)
		if !ok {
			continue
		}
		for _, key := range []string{"id", "remark", "machine_name"} {
			value := strings.ToLower(strings.TrimSpace(fmt.Sprint(vm[key])))
			if value != "" && (value == needle || strings.Contains(value, needle)) {
				matches = append(matches, vm)
				break
			}
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("narwhal instance not found")
	}
	return nil, fmt.Errorf("narwhal instance match is ambiguous")
}

func (f *NarwhalForwarder) forwards(vmID string) ([]map[string]any, error) {
	data, err := f.request("GET", "/vms/"+vmID+"/port-forwards", nil)
	if err != nil {
		return nil, err
	}
	items, _ := data.([]any)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f *NarwhalForwarder) request(method, path string, in any) (any, error) {
	var body *bytes.Reader
	if in == nil {
		body = bytes.NewReader(nil)
	} else {
		b, _ := json.Marshal(in)
		body = bytes.NewReader(b)
	}
	url := strings.TrimRight(f.BaseURL, "/") + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+f.Credential)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "frp-platform-nat/0.1")
	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var envelope struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || envelope.Code != 0 {
		return nil, fmt.Errorf("narwhal api %s: %s", resp.Status, envelope.Msg)
	}
	var out any
	if len(envelope.Data) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(envelope.Data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func chooseNATPort(used map[int]bool) int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 100; i++ {
		p := r.Intn(64512) + 1024
		if !used[p] {
			return p
		}
	}
	for p := 1024; p <= 65535; p++ {
		if !used[p] {
			return p
		}
	}
	return 0
}

func intFromAny(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case json.Number:
		i, _ := x.Int64()
		return int(i)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(x))
		return i
	default:
		return 0
	}
}
