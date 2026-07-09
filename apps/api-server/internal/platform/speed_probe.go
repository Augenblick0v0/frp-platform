package platform

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func runSpeedProbe(ctx context.Context, typ string, publicURL string, downloadBytes, uploadBytes int64) (SpeedTestProbeMetrics, error) {
	typ = strings.ToLower(strings.TrimSpace(typ))
	if downloadBytes <= 0 {
		downloadBytes = 8 * 1024 * 1024
	}
	if uploadBytes <= 0 {
		uploadBytes = 8 * 1024 * 1024
	}
	switch typ {
	case "http", "https":
		return probeHTTPThroughTunnel(ctx, publicURL, downloadBytes, uploadBytes)
	case "tcp":
		return probeTCPThroughTunnel(ctx, publicURL, downloadBytes, uploadBytes)
	case "udp":
		return probeUDPThroughTunnel(ctx, publicURL, downloadBytes, uploadBytes)
	default:
		return SpeedTestProbeMetrics{}, fmt.Errorf("unsupported speed test type %q", typ)
	}
}

func probeHTTPThroughTunnel(ctx context.Context, base string, downloadBytes, uploadBytes int64) (SpeedTestProbeMetrics, error) {
	base = strings.TrimRight(base, "/")
	latency, err := measureHTTPLatency(ctx, base+"/__frp_speed/ping")
	if err != nil {
		return SpeedTestProbeMetrics{}, err
	}
	down, downPeak, err := measureHTTPDownload(ctx, fmt.Sprintf("%s/__frp_speed/download?bytes=%d", base, downloadBytes))
	if err != nil {
		return SpeedTestProbeMetrics{}, err
	}
	up, upPeak, err := measureHTTPUpload(ctx, base+"/__frp_speed/upload", uploadBytes)
	if err != nil {
		return SpeedTestProbeMetrics{}, err
	}
	return SpeedTestProbeMetrics{DownloadAverageKbps: down.kbps(), DownloadPeakKbps: downPeak, UploadAverageKbps: up.kbps(), UploadPeakKbps: upPeak, LatencyMs: latency, BytesIn: down.bytes, BytesOut: up.bytes}, nil
}

func probeTCPThroughTunnel(ctx context.Context, publicURL string, downloadBytes, uploadBytes int64) (SpeedTestProbeMetrics, error) {
	addr := speedHostPort(publicURL)
	latency, err := measureTCPLatency(ctx, addr)
	if err != nil {
		return SpeedTestProbeMetrics{}, err
	}
	down, downPeak, err := measureTCPDownload(ctx, addr, downloadBytes)
	if err != nil {
		return SpeedTestProbeMetrics{}, err
	}
	up, upPeak, err := measureTCPUpload(ctx, addr, uploadBytes)
	if err != nil {
		return SpeedTestProbeMetrics{}, err
	}
	return SpeedTestProbeMetrics{DownloadAverageKbps: down.kbps(), DownloadPeakKbps: downPeak, UploadAverageKbps: up.kbps(), UploadPeakKbps: upPeak, LatencyMs: latency, BytesIn: down.bytes, BytesOut: up.bytes}, nil
}

func probeUDPThroughTunnel(ctx context.Context, publicURL string, downloadBytes, uploadBytes int64) (SpeedTestProbeMetrics, error) {
	addr := speedHostPort(publicURL)
	latency, err := measureUDPLatency(ctx, addr)
	if err != nil {
		return SpeedTestProbeMetrics{}, err
	}
	up, upPeak, err := measureUDPUpload(ctx, addr, uploadBytes)
	if err != nil {
		return SpeedTestProbeMetrics{}, err
	}
	down, downPeak, err := measureUDPDownload(ctx, addr, downloadBytes)
	if err != nil {
		return SpeedTestProbeMetrics{}, err
	}
	return SpeedTestProbeMetrics{DownloadAverageKbps: down.kbps(), DownloadPeakKbps: downPeak, UploadAverageKbps: up.kbps(), UploadPeakKbps: upPeak, LatencyMs: latency, BytesIn: down.bytes, BytesOut: up.bytes}, nil
}

type speedMeasurement struct {
	bytes   int64
	elapsed time.Duration
}

func (m speedMeasurement) kbps() float64 {
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

func measureHTTPDownload(ctx context.Context, rawURL string) (speedMeasurement, float64, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return speedMeasurement{}, 0, err
	}
	defer resp.Body.Close()
	return copySpeedMeasured(resp.Body)
}

func measureHTTPUpload(ctx context.Context, rawURL string, n int64) (speedMeasurement, float64, error) {
	body := io.LimitReader(speedPatternReader{}, n)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, body)
	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return speedMeasurement{}, 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	m := speedMeasurement{bytes: n, elapsed: time.Since(start)}
	return m, m.kbps(), nil
}

func measureTCPLatency(ctx context.Context, addr string) (float64, error) {
	start := time.Now()
	conn, err := dialSpeedContext(ctx, "tcp", addr)
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

func measureTCPDownload(ctx context.Context, addr string, n int64) (speedMeasurement, float64, error) {
	conn, err := dialSpeedContext(ctx, "tcp", addr)
	if err != nil {
		return speedMeasurement{}, 0, err
	}
	defer conn.Close()
	_, _ = conn.Write([]byte{'d'})
	writeSpeedInt64(conn, n)
	return copySpeedMeasured(conn)
}

func measureTCPUpload(ctx context.Context, addr string, n int64) (speedMeasurement, float64, error) {
	conn, err := dialSpeedContext(ctx, "tcp", addr)
	if err != nil {
		return speedMeasurement{}, 0, err
	}
	defer conn.Close()
	_, _ = conn.Write([]byte{'u'})
	writeSpeedInt64(conn, n)
	start := time.Now()
	written, err := io.CopyN(conn, speedPatternReader{}, n)
	if err != nil {
		return speedMeasurement{}, 0, err
	}
	var ack [2]byte
	_, _ = io.ReadFull(conn, ack[:])
	m := speedMeasurement{bytes: written, elapsed: time.Since(start)}
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

func measureUDPUpload(ctx context.Context, addr string, n int64) (speedMeasurement, float64, error) {
	_ = ctx
	conn, err := net.DialTimeout("udp", addr, 5*time.Second)
	if err != nil {
		return speedMeasurement{}, 0, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	payload := make([]byte, 1200)
	payload[0] = 'u'
	start := time.Now()
	var sent int64
	var peak speedPeakMeter
	for sent < n {
		if n-sent < int64(len(payload)) {
			payload = payload[:n-sent]
			payload[0] = 'u'
		}
		w, err := conn.Write(payload)
		if err != nil {
			return speedMeasurement{}, 0, err
		}
		sent += int64(w)
		peak.add(int64(w))
	}
	return speedMeasurement{bytes: sent, elapsed: time.Since(start)}, peak.kbps(), nil
}

func measureUDPDownload(ctx context.Context, addr string, n int64) (speedMeasurement, float64, error) {
	_ = ctx
	conn, err := net.DialTimeout("udp", addr, 5*time.Second)
	if err != nil {
		return speedMeasurement{}, 0, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	_, _ = conn.Write([]byte{'d'})
	buf := make([]byte, 1500)
	start := time.Now()
	var got int64
	var peak speedPeakMeter
	for got < n {
		read, err := conn.Read(buf)
		if err != nil {
			break
		}
		got += int64(read)
		peak.add(int64(read))
	}
	return speedMeasurement{bytes: got, elapsed: time.Since(start)}, peak.kbps(), nil
}

func copySpeedMeasured(r io.Reader) (speedMeasurement, float64, error) {
	buf := make([]byte, 64*1024)
	start := time.Now()
	var total int64
	var peak speedPeakMeter
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
			return speedMeasurement{}, 0, err
		}
	}
	return speedMeasurement{bytes: total, elapsed: time.Since(start)}, peak.kbps(), nil
}

type speedPeakMeter struct {
	windowStart time.Time
	windowBytes int64
	peakKbps    float64
}

func (p *speedPeakMeter) add(n int64) {
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

func (p *speedPeakMeter) kbps() float64 {
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

type speedPatternReader struct{}

func (speedPatternReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(i % 251)
	}
	return len(p), nil
}

func writeSpeedInt64(w io.Writer, n int64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(n))
	_, _ = w.Write(b[:])
}

func dialSpeedContext(ctx context.Context, network, addr string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, network, addr)
}

func speedHostPort(raw string) string {
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil {
			return u.Host
		}
	}
	return raw
}
