package djinnlog

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMultiHandler_FanOut(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	h1 := slog.NewTextHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelDebug})
	h2 := slog.NewTextHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelDebug})
	multi := NewMultiHandler(h1, h2)
	log := slog.New(multi)

	log.Info("hello", "key", "value")

	if !strings.Contains(buf1.String(), "hello") {
		t.Fatalf("handler 1 missing message: %s", buf1.String())
	}
	if !strings.Contains(buf2.String(), "hello") {
		t.Fatalf("handler 2 missing message: %s", buf2.String())
	}
}

func TestMultiHandler_LevelFiltering(t *testing.T) {
	var debugBuf, infoBuf bytes.Buffer
	debugH := slog.NewTextHandler(&debugBuf, &slog.HandlerOptions{Level: slog.LevelDebug})
	infoH := slog.NewTextHandler(&infoBuf, &slog.HandlerOptions{Level: slog.LevelInfo})
	multi := NewMultiHandler(debugH, infoH)
	log := slog.New(multi)

	log.Debug("debug message")

	if !strings.Contains(debugBuf.String(), "debug message") {
		t.Fatal("debug handler should have debug message")
	}
	if strings.Contains(infoBuf.String(), "debug message") {
		t.Fatal("info handler should NOT have debug message")
	}
}

func TestMultiHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	multi := NewMultiHandler(h)
	log := slog.New(multi.WithAttrs([]slog.Attr{slog.String("component", "test")}))

	log.Info("msg")

	if !strings.Contains(buf.String(), "component=test") {
		t.Fatalf("missing component attr: %s", buf.String())
	}
}

func TestRingHandler_Basic(t *testing.T) {
	ring := NewRingHandler(10)
	log := slog.New(ring)

	log.Info("hello")
	log.Warn("warning")

	entries := ring.Entries()
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0].Message != "hello" {
		t.Fatalf("first = %q", entries[0].Message)
	}
	if entries[1].Level != slog.LevelWarn {
		t.Fatalf("second level = %v", entries[1].Level)
	}
}

func TestRingHandler_Capacity(t *testing.T) {
	ring := NewRingHandler(3)
	log := slog.New(ring)

	for i := range 5 {
		log.Info("msg", "i", i)
	}

	entries := ring.Entries()
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}
	// Oldest should be i=2 (0 and 1 evicted)
	if entries[0].Attrs["i"] != int64(2) {
		t.Fatalf("oldest entry i = %v, want 2", entries[0].Attrs["i"])
	}
}

func TestRingHandler_Filter(t *testing.T) {
	ring := NewRingHandler(10)
	log := slog.New(ring)

	log.Debug("debug")
	log.Info("info")
	log.Warn("warn")
	log.Error("error")

	errors := ring.Filter(slog.LevelError)
	if len(errors) != 1 {
		t.Fatalf("errors = %d, want 1", len(errors))
	}

	warns := ring.Filter(slog.LevelWarn)
	if len(warns) != 2 { // warn + error
		t.Fatalf("warns+ = %d, want 2", len(warns))
	}
}

func TestRingHandler_Component(t *testing.T) {
	ring := NewRingHandler(10)
	log := slog.New(ring).With("component", "driver")

	log.Info("test")

	entries := ring.Entries()
	if len(entries) != 1 {
		t.Fatal("expected 1 entry")
	}
	if entries[0].Component != "driver" {
		t.Fatalf("component = %q, want driver", entries[0].Component)
	}
}

func TestRingHandler_Concurrent(t *testing.T) {
	ring := NewRingHandler(100)
	log := slog.New(ring)

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := range 10 {
				log.Info("msg", "goroutine", n, "iter", j)
			}
		}(i)
	}
	wg.Wait()

	if ring.Len() != 100 {
		t.Fatalf("len = %d, want 100", ring.Len())
	}
}

func TestFor(t *testing.T) {
	var buf bytes.Buffer
	parent := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	child := For(parent, "agent")

	child.Info("test")

	if !strings.Contains(buf.String(), "component=agent") {
		t.Fatalf("missing component: %s", buf.String())
	}
}

func TestFor_NilParent(t *testing.T) {
	log := For(nil, "test")
	// Should not panic
	log.Info("test")
}

func TestNop(t *testing.T) {
	log := Nop()
	// Should not panic or produce output
	log.Debug("ignored")
	log.Error("also ignored")
}

func TestSetup_NoVerbose(t *testing.T) {
	dir := t.TempDir()
	result := Setup(Options{
		LogFile: filepath.Join(dir, "test.log"),
	})
	if result.Logger == nil {
		t.Fatal("logger should not be nil")
	}
	if result.Ring == nil {
		t.Fatal("ring should not be nil")
	}
}

func TestSetup_WithVerbose(t *testing.T) {
	dir := t.TempDir()
	result := Setup(Options{
		Verbose: true,
		LogFile: filepath.Join(dir, "test.log"),
	})
	result.Logger.Info("test message")
	// Should not panic
}

func TestPerf_TTFT(t *testing.T) {
	attr := TTFT(100 * time.Millisecond)
	if attr.Key != "ttft" {
		t.Fatalf("key = %q", attr.Key)
	}
}

func TestPerf_RTT(t *testing.T) {
	attr := RTT(2 * time.Second)
	if attr.Key != "rtt" {
		t.Fatalf("key = %q", attr.Key)
	}
}

func TestPerf_Throughput(t *testing.T) {
	attr := Throughput(100, 2*time.Second)
	if attr.Key != "throughput" {
		t.Fatalf("key = %q", attr.Key)
	}
	tps := attr.Value.Float64()
	if tps < 49 || tps > 51 {
		t.Fatalf("throughput = %f, want ~50", tps)
	}
}

func TestPerf_Throughput_ZeroDuration(t *testing.T) {
	attr := Throughput(100, 0)
	if attr.Value.Float64() != 0 {
		t.Fatal("zero duration should return 0 throughput")
	}
}

func TestPerf_ContextPct(t *testing.T) {
	attr := ContextPct(80000, 200000)
	if attr.Key != "context_pct" {
		t.Fatalf("key = %q", attr.Key)
	}
	pct := attr.Value.Float64()
	if pct < 0.39 || pct > 0.41 {
		t.Fatalf("pct = %f, want ~0.4", pct)
	}
}

func TestPerf_ContextPct_ZeroTotal(t *testing.T) {
	attr := ContextPct(100, 0)
	if attr.Value.Float64() != 0 {
		t.Fatal("zero total should return 0")
	}
}
