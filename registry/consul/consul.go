package consul

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/registry"
)

// Config is the minimal Consul HTTP API configuration.
//
// Examples:
//   Address = "127.0.0.1:8500"
//   Scheme  = "http"
type Config struct {
	Address   string
	Scheme    string
	Token     string
	WaitEvery time.Duration
}

type client struct {
	baseURL string
	token   string
	hc      *http.Client
}

func newClient(cfg Config) (*client, error) {
	if cfg.Scheme == "" {
		cfg.Scheme = "http"
	}
	if cfg.Address == "" {
		return nil, fmt.Errorf("consul address is required")
	}
	if cfg.WaitEvery <= 0 {
		cfg.WaitEvery = 2 * time.Second
	}
	base := fmt.Sprintf("%s://%s", cfg.Scheme, cfg.Address)
	return &client{
		baseURL: base,
		token:   strings.TrimSpace(cfg.Token),
		hc:      &http.Client{Timeout: 5 * time.Second},
	}, nil
}

// NewConsulRegistrar creates both Registrar and Discovery for the given config.
//
// Note:
// - This is a lightweight implementation based on Consul HTTP API and Kratos registry interfaces.
// - It supports register/deregister and discovery via catalog endpoints.
func NewConsulRegistrar(cfg Config) (registry.Registrar, registry.Discovery, error) {
	c, err := newClient(cfg)
	if err != nil {
		return nil, nil, err
	}
	r := &consulRegistry{
		client:  c,
		waitEvery: cfg.WaitEvery,
	}
	return r, r, nil
}

type consulRegistry struct {
	client    *client
	waitEvery time.Duration
}

func (r *consulRegistry) Register(ctx context.Context, service *registry.ServiceInstance) error {
	address, port, err := endpointsToHostPort(service.Endpoints)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"ID":      service.ID,
		"Name":    service.Name,
		"Address": address,
		"Port":    port,
		"Meta":    service.Metadata,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, r.client.baseURL+"/v1/agent/service/register", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	r.client.decorate(req)

	resp, err := r.client.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("consul register failed: status=%d", resp.StatusCode)
	}
	return nil
}

func (r *consulRegistry) Deregister(ctx context.Context, service *registry.ServiceInstance) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, r.client.baseURL+"/v1/agent/service/deregister/"+url.PathEscape(service.ID), nil)
	if err != nil {
		return err
	}
	r.client.decorate(req)

	resp, err := r.client.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("consul deregister failed: status=%d", resp.StatusCode)
	}
	return nil
}

func (r *consulRegistry) GetService(ctx context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	// Using catalog endpoint: it includes ServiceAddress/ServicePort/ServiceMeta.
	// Endpoint docs: GET /v1/catalog/service/<service>
	u := r.client.baseURL + "/v1/catalog/service/" + url.PathEscape(serviceName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	r.client.decorate(req)

	resp, err := r.client.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("consul catalog failed: status=%d", resp.StatusCode)
	}

	var entries []struct {
		ServiceID      string            `json:"ServiceID"`
		ServiceName    string            `json:"ServiceName"`
		ServiceAddress string            `json:"ServiceAddress"`
		ServicePort    int               `json:"ServicePort"`
		ServiceMeta    map[string]string `json:"ServiceMeta"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}

	out := make([]*registry.ServiceInstance, 0, len(entries))
	for _, e := range entries {
		out = append(out, &registry.ServiceInstance{
			ID:       e.ServiceID,
			Name:     e.ServiceName,
			Version:  "",
			Metadata: e.ServiceMeta,
			Endpoints: []string{
				fmt.Sprintf("%s:%d", e.ServiceAddress, e.ServicePort),
			},
		})
	}

	// Keep order stable for equality hashing.
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *consulRegistry) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	return newPollingWatcher(ctx, r, serviceName, r.waitEvery)
}

type pollingWatcher struct {
	ctx         context.Context
	cancel      context.CancelFunc
	serviceName string
	reg         *consulRegistry

	first bool
	lastHash string
}

func newPollingWatcher(parent context.Context, reg *consulRegistry, serviceName string, waitEvery time.Duration) (*pollingWatcher, error) {
	ctx, cancel := context.WithCancel(parent)
	return &pollingWatcher{
		ctx:         ctx,
		cancel:      cancel,
		serviceName: serviceName,
		reg:         reg,
		first:      true,
		lastHash:   "",
	}, nil
}

func (w *pollingWatcher) Next() ([]*registry.ServiceInstance, error) {
	waitEvery := w.reg.waitEvery
	for {
		select {
		case <-w.ctx.Done():
			return nil, w.ctx.Err()
		default:
		}

		list, err := w.reg.GetService(w.ctx, w.serviceName)
		if err != nil {
			// Retry on transient errors, but respect ctx cancellation.
			select {
			case <-time.After(waitEvery):
				continue
			case <-w.ctx.Done():
				return nil, w.ctx.Err()
			}
		}

		currentHash := hashServiceList(list)
		if w.first {
			w.first = false
			w.lastHash = currentHash
			return list, nil
		}
		if currentHash != w.lastHash {
			w.lastHash = currentHash
			return list, nil
		}

		select {
		case <-time.After(waitEvery):
		case <-w.ctx.Done():
			return nil, w.ctx.Err()
		}
	}
}

func (w *pollingWatcher) Stop() error {
	w.cancel()
	return nil
}

func hashServiceList(list []*registry.ServiceInstance) string {
	ids := make([]string, 0, len(list))
	for _, s := range list {
		// Only hash stable fields for change detection.
		b, _ := json.Marshal([]any{s.ID, s.Name, s.Endpoints, s.Metadata})
		sum := sha256.Sum256(b)
		ids = append(ids, hex.EncodeToString(sum[:]))
	}
	sort.Strings(ids)
	sum := sha256.Sum256([]byte(strings.Join(ids, ",")))
	return hex.EncodeToString(sum[:])
}

func endpointsToHostPort(endpoints []string) (string, int, error) {
	for _, ep := range endpoints {
		// Supported formats:
		// - grpc://127.0.0.1:9000?isSecure=false
		// - http://127.0.0.1:8000
		// - 127.0.0.1:9000 (fallback)
		host, port, err := hostPortFromEndpoint(ep)
		if err == nil {
			return host, port, nil
		}
	}
	return "", 0, fmt.Errorf("failed to parse host:port from endpoints: %v", endpoints)
}

func hostPortFromEndpoint(ep string) (string, int, error) {
	if ep == "" {
		return "", 0, fmt.Errorf("empty endpoint")
	}

	// If it's already host:port, parse directly.
	if strings.Contains(ep, "://") == false {
		h, p, err := net.SplitHostPort(ep)
		if err == nil {
			pi, _ := strconv.Atoi(p)
			return h, pi, nil
		}
		// Might be host only; ignore.
		return "", 0, fmt.Errorf("invalid host:port endpoint: %s", ep)
	}

	u, err := url.Parse(ep)
	if err != nil {
		return "", 0, err
	}
	if u.Host == "" {
		return "", 0, fmt.Errorf("missing host in endpoint: %s", ep)
	}
	h, p, err := net.SplitHostPort(u.Host)
	if err != nil {
		return "", 0, err
	}
	pi, err := strconv.Atoi(p)
	if err != nil {
		return "", 0, err
	}
	return h, pi, nil
}

func (c *client) decorate(req *http.Request) {
	if c.token != "" {
		req.Header.Set("X-Consul-Token", c.token)
	}
	req.Header.Set("Content-Type", "application/json")
}

