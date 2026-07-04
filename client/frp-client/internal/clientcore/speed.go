package clientcore

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SpeedTestRunRequest struct {
	APIBase            string `json:"api_base"`
	Token              string `json:"token"`
	Type               string `json:"type"`
	DownloadBytes      int64  `json:"download_bytes"`
	UploadBytes        int64  `json:"upload_bytes"`
	DurationSeconds    int    `json:"duration_seconds"`
	BandwidthLimitKbps int    `json:"bandwidth_limit_kbps"`
}

type SpeedTestResult struct {
	Type                        string  `json:"type"`
	TunnelID                    int64   `json:"tunnel_id"`
	PublicURL                   string  `json:"public_url"`
	DownloadAverageKbps         float64 `json:"download_average_kbps"`
	DownloadPeakKbps            float64 `json:"download_peak_kbps"`
	UploadAverageKbps           float64 `json:"upload_average_kbps"`
	UploadPeakKbps              float64 `json:"upload_peak_kbps"`
	LatencyMs                   float64 `json:"latency_ms"`
	BytesIn                     int64   `json:"bytes_in"`
	BytesOut                    int64   `json:"bytes_out"`
	EffectiveBandwidthLimitKbps int     `json:"effective_bandwidth_limit_kbps"`
	LimitRatio                  float64 `json:"limit_ratio"`
}

type speedTestTunnel struct {
	ID                          int64     `json:"id"`
	Type                        string    `json:"type"`
	LocalHost                   string    `json:"local_host"`
	LocalPort                   int       `json:"local_port"`
	RemotePort                  int       `json:"remote_port"`
	Domain                      string    `json:"domain"`
	PublicURL                   string    `json:"public_url"`
	EffectiveBandwidthLimitKbps int       `json:"effective_bandwidth_limit_kbps"`
	ExpiresAt                   time.Time `json:"expires_at"`
}

type benchmarkService struct {
	typ    string
	host   string
	port   int
	close  func() error
	closed sync.Once
}

func (m *Manager) RunSpeedTest(ctx context.Context, in SpeedTestRunRequest) (SpeedTestResult, error) {
	typ := strings.ToLower(strings.TrimSpace(in.Type))
	if typ == "" {
		typ = "tcp"
	}
	if in.DownloadBytes <= 0 {
		in.DownloadBytes = 8 * 1024 * 1024
	}
	if in.UploadBytes <= 0 {
		in.UploadBytes = 8 * 1024 * 1024
	}
	if in.DurationSeconds <= 0 {
		in.DurationSeconds = 15
	}
	if in.DurationSeconds > 60 {
		in.DurationSeconds = 60
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(in.DurationSeconds+20)*time.Second)
	defer cancel()
	bench, err := startBenchmarkService(typ)
	if err != nil {
		return SpeedTestResult{}, err
	}
	defer bench.Close()
	tunnel, err := createRemoteSpeedTestTunnel(ctx, in.APIBase, in.Token, typ, bench.host, bench.port, in.BandwidthLimitKbps)
	if err != nil {
		return SpeedTestResult{}, err
	}
	cleaned := false
	cleanup := func() {
		if cleaned {
			return
		}
		cleaned = true
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_ = finishRemoteSpeedTestTunnel(cleanupCtx, in.APIBase, in.Token, tunnel.ID)
		if _, err := m.SyncFromServer(cleanupCtx, in.APIBase, in.Token); err == nil {
			_ = m.Restart()
		}
	}
	defer cleanup()
	if _, err := m.SyncFromServer(ctx, in.APIBase, in.Token); err != nil {
		return SpeedTestResult{}, err
	}
	if err := m.Restart(); err != nil {
		return SpeedTestResult{}, err
	}
	result, err := runProtocolProbeWithRetry(ctx, typ, tunnel.PublicURL, in.DownloadBytes, in.UploadBytes, 8*time.Second)
	if err != nil {
		logs, _ := m.Logs(12000)
		return SpeedTestResult{}, fmt.Errorf("speed probe failed: %w\nfrpc logs:\n%s", err, logs)
	}
	result.Type = typ
	result.TunnelID = tunnel.ID
	result.PublicURL = tunnel.PublicURL
	result.EffectiveBandwidthLimitKbps = tunnel.EffectiveBandwidthLimitKbps
	if result.EffectiveBandwidthLimitKbps > 0 {
		observed := math.Max(result.DownloadAverageKbps, result.UploadAverageKbps)
		result.LimitRatio = observed / float64(result.EffectiveBandwidthLimitKbps)
	}
	_ = reportTrafficToServer(ctx, in.APIBase, in.Token, tunnel.ID, result.BytesIn, result.BytesOut)
	cleanup()
	return result, nil
}

func (m *Manager) Restart() error {
	if err := m.Stop(); err != nil {
		return err
	}
	return m.Start()
}

func (b *benchmarkService) Close() error {
	var err error
	b.closed.Do(func() {
		if b.close != nil {
			err = b.close()
		}
	})
	return err
}

func startBenchmarkService(typ string) (*benchmarkService, error) {
	switch typ {
	case "http", "https":
		return startHTTPBenchmarkService(typ)
	case "tcp":
		return startTCPBenchmarkService()
	case "udp":
		return startUDPBenchmarkService()
	default:
		return nil, fmt.Errorf("unsupported speed test type %q", typ)
	}
}

func startHTTPBenchmarkService(typ string) (*benchmarkService, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/__frp_speed/ping", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("pong")) })
	mux.HandleFunc("/__frp_speed/download", func(w http.ResponseWriter, r *http.Request) {
		n, _ := strconv.ParseInt(r.URL.Query().Get("bytes"), 10, 64)
		if n <= 0 {
			n = 1024 * 1024
		}
		writePattern(w, n)
	})
	mux.HandleFunc("/__frp_speed/upload", func(w http.ResponseWriter, r *http.Request) {
		n, _ := io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		w.Header().Set("X-Frp-Speed-Bytes", strconv.FormatInt(n, 10))
		_, _ = w.Write([]byte("ok"))
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	return &benchmarkService{typ: typ, host: "127.0.0.1", port: ln.Addr().(*net.TCPAddr).Port, close: srv.Close}, nil
}

func startTCPBenchmarkService() (*benchmarkService, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	done := make(chan struct{})
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					continue
				}
			}
			go handleTCPBenchmark(conn)
		}
	}()
	return &benchmarkService{typ: "tcp", host: "127.0.0.1", port: ln.Addr().(*net.TCPAddr).Port, close: func() error { close(done); return ln.Close() }}, nil
}

func handleTCPBenchmark(conn net.Conn) {
	defer conn.Close()
	var op [1]byte
	if _, err := io.ReadFull(conn, op[:]); err != nil {
		return
	}
	switch op[0] {
	case 'p':
		_, _ = conn.Write([]byte("p"))
	case 'd':
		n := readInt64(conn)
		writePattern(conn, n)
	case 'u':
		n := readInt64(conn)
		_, _ = io.CopyN(io.Discard, conn, n)
		_, _ = conn.Write([]byte("ok"))
	}
}

func startUDPBenchmarkService() (*benchmarkService, error) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 65535)
		for {
			n, addr, err := pc.ReadFrom(buf)
			if err != nil {
				select {
				case <-done:
					return
				default:
					continue
				}
			}
			if n == 0 {
				continue
			}
			switch buf[0] {
			case 'p':
				_, _ = pc.WriteTo([]byte("p"), addr)
			case 'd':
				for i := 0; i < 64; i++ {
					payload := make([]byte, 1200)
					payload[0] = 'd'
					_, _ = pc.WriteTo(payload, addr)
				}
			case 'u':
				_, _ = pc.WriteTo([]byte("a"), addr)
			}
		}
	}()
	return &benchmarkService{typ: "udp", host: "127.0.0.1", port: pc.LocalAddr().(*net.UDPAddr).Port, close: func() error { close(done); return pc.Close() }}, nil
}

func createRemoteSpeedTestTunnel(ctx context.Context, apiBase, token, typ, localHost string, localPort int, bandwidth int) (speedTestTunnel, error) {
	body, _ := json.Marshal(map[string]any{"type": typ, "local_host": localHost, "local_port": localPort, "bandwidth_limit_kbps": bandwidth})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, trimSlash(apiBase)+"/api/speed-tests/tunnels", bytes.NewReader(body))
	if err != nil {
		return speedTestTunnel{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return speedTestTunnel{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return speedTestTunnel{}, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(b))
	}
	var env struct {
		Success bool            `json:"success"`
		Data    speedTestTunnel `json:"data"`
		Message string          `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return speedTestTunnel{}, err
	}
	if !env.Success {
		return speedTestTunnel{}, fmt.Errorf(env.Message)
	}
	return env.Data, nil
}

func finishRemoteSpeedTestTunnel(ctx context.Context, apiBase, token string, id int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/speed-tests/%d/finish", trimSlash(apiBase), id), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func reportTrafficToServer(ctx context.Context, apiBase, token string, tunnelID int64, bytesIn, bytesOut int64) error {
	body, _ := json.Marshal(map[string]any{"reports": []TrafficReport{{TunnelID: tunnelID, BytesIn: bytesIn, BytesOut: bytesOut}}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, trimSlash(apiBase)+"/api/client/traffic", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func runProtocolProbe(ctx context.Context, typ string, publicURL string, downloadBytes, uploadBytes int64) (SpeedTestResult, error) {
	switch typ {
	case "http", "https":
		return probeHTTP(ctx, publicURL, downloadBytes, uploadBytes)
	case "tcp":
		return probeTCP(ctx, publicURL, downloadBytes, uploadBytes)
	case "udp":
		return probeUDP(ctx, publicURL, downloadBytes, uploadBytes)
	default:
		return SpeedTestResult{}, fmt.Errorf("unsupported speed test type %q", typ)
	}
}

func runProtocolProbeWithRetry(ctx context.Context, typ string, publicURL string, downloadBytes, uploadBytes int64, wait time.Duration) (SpeedTestResult, error) {
	deadline := time.Now().Add(wait)
	var last error
	for {
		result, err := runProtocolProbe(ctx, typ, publicURL, downloadBytes, uploadBytes)
		if err == nil {
			return result, nil
		}
		last = err
		if time.Now().After(deadline) {
			return SpeedTestResult{}, last
		}
		select {
		case <-ctx.Done():
			return SpeedTestResult{}, ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
}

func probeHTTP(ctx context.Context, base string, downloadBytes, uploadBytes int64) (SpeedTestResult, error) {
	base = strings.TrimRight(base, "/")
	latency, err := measureHTTPLatency(ctx, base+"/__frp_speed/ping")
	if err != nil {
		return SpeedTestResult{}, err
	}
	down, downPeak, err := measureHTTPDownload(ctx, fmt.Sprintf("%s/__frp_speed/download?bytes=%d", base, downloadBytes))
	if err != nil {
		return SpeedTestResult{}, err
	}
	up, upPeak, err := measureHTTPUpload(ctx, base+"/__frp_speed/upload", uploadBytes)
	if err != nil {
		return SpeedTestResult{}, err
	}
	return SpeedTestResult{DownloadAverageKbps: down.kbps(), DownloadPeakKbps: downPeak, UploadAverageKbps: up.kbps(), UploadPeakKbps: upPeak, LatencyMs: latency, BytesIn: down.bytes, BytesOut: up.bytes}, nil
}

func probeTCP(ctx context.Context, publicURL string, downloadBytes, uploadBytes int64) (SpeedTestResult, error) {
	addr := hostPortFromPublicURL(publicURL)
	latency, err := measureTCPLatency(ctx, addr)
	if err != nil {
		return SpeedTestResult{}, err
	}
	down, downPeak, err := measureTCPDownload(ctx, addr, downloadBytes)
	if err != nil {
		return SpeedTestResult{}, err
	}
	up, upPeak, err := measureTCPUpload(ctx, addr, uploadBytes)
	if err != nil {
		return SpeedTestResult{}, err
	}
	return SpeedTestResult{DownloadAverageKbps: down.kbps(), DownloadPeakKbps: downPeak, UploadAverageKbps: up.kbps(), UploadPeakKbps: upPeak, LatencyMs: latency, BytesIn: down.bytes, BytesOut: up.bytes}, nil
}

func probeUDP(ctx context.Context, publicURL string, downloadBytes, uploadBytes int64) (SpeedTestResult, error) {
	addr := hostPortFromPublicURL(publicURL)
	latency, err := measureUDPLatency(ctx, addr)
	if err != nil {
		return SpeedTestResult{}, err
	}
	up, upPeak, err := measureUDPUpload(ctx, addr, uploadBytes)
	if err != nil {
		return SpeedTestResult{}, err
	}
	down, downPeak, err := measureUDPDownload(ctx, addr, downloadBytes)
	if err != nil {
		return SpeedTestResult{}, err
	}
	return SpeedTestResult{DownloadAverageKbps: down.kbps(), DownloadPeakKbps: downPeak, UploadAverageKbps: up.kbps(), UploadPeakKbps: upPeak, LatencyMs: latency, BytesIn: down.bytes, BytesOut: up.bytes}, nil
}

type measurement struct {
	bytes   int64
	elapsed time.Duration
}

func (m measurement) kbps() float64 {
	if m.elapsed <= 0 {
		return 0
	}
	return float64(m.bytes*8) / m.elapsed.Seconds() / 1000
}

func measureHTTPLatency(ctx context.Context, rawURL string) (float64, error) {
	start := time.Now()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return float64(time.Since(start).Microseconds()) / 1000, nil
}

func measureHTTPDownload(ctx context.Context, rawURL string) (measurement, float64, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return measurement{}, 0, err
	}
	defer resp.Body.Close()
	return copyMeasured(resp.Body)
}

func measureHTTPUpload(ctx context.Context, rawURL string, n int64) (measurement, float64, error) {
	body := io.LimitReader(patternReader{}, n)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, body)
	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return measurement{}, 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	m := measurement{bytes: n, elapsed: time.Since(start)}
	return m, m.kbps(), nil
}

func measureTCPLatency(ctx context.Context, addr string) (float64, error) {
	start := time.Now()
	conn, err := dialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	_, _ = conn.Write([]byte{'p'})
	var b [1]byte
	if _, err := io.ReadFull(conn, b[:]); err != nil {
		return 0, err
	}
	return float64(time.Since(start).Microseconds()) / 1000, nil
}

func measureTCPDownload(ctx context.Context, addr string, n int64) (measurement, float64, error) {
	conn, err := dialContext(ctx, "tcp", addr)
	if err != nil {
		return measurement{}, 0, err
	}
	defer conn.Close()
	_, _ = conn.Write([]byte{'d'})
	writeInt64(conn, n)
	return copyMeasured(conn)
}

func measureTCPUpload(ctx context.Context, addr string, n int64) (measurement, float64, error) {
	conn, err := dialContext(ctx, "tcp", addr)
	if err != nil {
		return measurement{}, 0, err
	}
	defer conn.Close()
	_, _ = conn.Write([]byte{'u'})
	writeInt64(conn, n)
	start := time.Now()
	written, err := io.CopyN(conn, patternReader{}, n)
	if err != nil {
		return measurement{}, 0, err
	}
	var ack [2]byte
	_, _ = io.ReadFull(conn, ack[:])
	elapsed := time.Since(start)
	m := measurement{bytes: written, elapsed: elapsed}
	return m, m.kbps(), nil
}

func measureUDPLatency(ctx context.Context, addr string) (float64, error) {
	_ = ctx
	conn, err := net.DialTimeout("udp", addr, 5*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	start := time.Now()
	_, _ = conn.Write([]byte{'p'})
	var b [1]byte
	if _, err := conn.Read(b[:]); err != nil {
		return 0, err
	}
	return float64(time.Since(start).Microseconds()) / 1000, nil
}

func measureUDPUpload(ctx context.Context, addr string, n int64) (measurement, float64, error) {
	_ = ctx
	conn, err := net.DialTimeout("udp", addr, 5*time.Second)
	if err != nil {
		return measurement{}, 0, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	payload := make([]byte, 1200)
	payload[0] = 'u'
	start := time.Now()
	var sent int64
	var peak peakMeter
	for sent < n {
		if n-sent < int64(len(payload)) {
			payload = payload[:n-sent]
			payload[0] = 'u'
		}
		w, err := conn.Write(payload)
		if err != nil {
			return measurement{}, 0, err
		}
		sent += int64(w)
		peak.add(int64(w))
	}
	return measurement{bytes: sent, elapsed: time.Since(start)}, peak.kbps(), nil
}

func measureUDPDownload(ctx context.Context, addr string, n int64) (measurement, float64, error) {
	_ = ctx
	conn, err := net.DialTimeout("udp", addr, 5*time.Second)
	if err != nil {
		return measurement{}, 0, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	_, _ = conn.Write([]byte{'d'})
	buf := make([]byte, 1500)
	start := time.Now()
	var got int64
	var peak peakMeter
	for got < n {
		read, err := conn.Read(buf)
		if err != nil {
			break
		}
		got += int64(read)
		peak.add(int64(read))
	}
	return measurement{bytes: got, elapsed: time.Since(start)}, peak.kbps(), nil
}

func copyMeasured(r io.Reader) (measurement, float64, error) {
	buf := make([]byte, 64*1024)
	start := time.Now()
	var total int64
	var peak peakMeter
	for {
		n, err := r.Read(buf)
		if n > 0 {
			total += int64(n)
			peak.add(int64(n))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return measurement{}, 0, err
		}
	}
	return measurement{bytes: total, elapsed: time.Since(start)}, peak.kbps(), nil
}

type peakMeter struct {
	windowStart time.Time
	windowBytes int64
	peakKbps    float64
}

func (p *peakMeter) add(n int64) {
	now := time.Now()
	if p.windowStart.IsZero() {
		p.windowStart = now
	}
	p.windowBytes += n
	elapsed := now.Sub(p.windowStart)
	if elapsed >= 250*time.Millisecond {
		kbps := float64(p.windowBytes*8) / elapsed.Seconds() / 1000
		if kbps > p.peakKbps {
			p.peakKbps = kbps
		}
		p.windowStart = now
		p.windowBytes = 0
	}
}

func (p *peakMeter) kbps() float64 {
	if p.peakKbps > 0 {
		return p.peakKbps
	}
	if p.windowBytes == 0 || p.windowStart.IsZero() {
		return 0
	}
	elapsed := time.Since(p.windowStart)
	if elapsed <= 0 {
		return 0
	}
	return float64(p.windowBytes*8) / elapsed.Seconds() / 1000
}

type patternReader struct{}

func (patternReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(i % 251)
	}
	return len(p), nil
}

func writePattern(w io.Writer, n int64) {
	_, _ = io.CopyN(w, patternReader{}, n)
}

func writeInt64(w io.Writer, n int64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(n))
	_, _ = w.Write(b[:])
}

func readInt64(r io.Reader) int64 {
	var b [8]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0
	}
	return int64(binary.BigEndian.Uint64(b[:]))
}

func dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, network, addr)
}

func hostPortFromPublicURL(raw string) string {
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil {
			return u.Host
		}
	}
	return raw
}
