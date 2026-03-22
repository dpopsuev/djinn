package acceptance

import (
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/djinnlog"
)

func TestLog_SetupVerboseThreeHandlers(t *testing.T) {
	result := djinnlog.Setup(djinnlog.Options{
		Verbose: true,
		LogFile: filepath.Join(t.TempDir(), "test.log"),
	})
	if result.Logger == nil {
		t.Fatal("logger nil")
	}
	if result.Ring == nil {
		t.Fatal("ring nil")
	}
	// Verbose mode should produce terminal output (can't verify stderr easily,
	// but the logger should accept Info level)
	result.Logger.Info("test")
}

func TestLog_SetupQuietNoTerminal(t *testing.T) {
	result := djinnlog.Setup(djinnlog.Options{
		LogFile: filepath.Join(t.TempDir(), "test.log"),
	})
	result.Logger.Info("quiet test")
	// Ring should capture it
	if result.Ring.Len() == 0 {
		t.Fatal("ring should capture even in quiet mode")
	}
}

func TestLog_RedactionStripsSecrets(t *testing.T) {
	result := djinnlog.Setup(djinnlog.Options{
		RingSize: 10,
	})

	result.Logger.Info("auth", "api_key", "sk-secret-123", "model", "claude")

	entries := result.Ring.Entries()
	for _, e := range entries {
		for k, v := range e.Attrs {
			if strings.Contains(strings.ToLower(k), "api_key") {
				if str, ok := v.(string); ok && str != "[REDACTED]" {
					t.Fatalf("api_key should be redacted, got %q", str)
				}
			}
		}
	}
}

func TestLog_RingBufferCapacity(t *testing.T) {
	ring := djinnlog.NewRingHandler(5)
	log := slog.New(ring)

	for i := range 10 {
		log.Info("msg", "i", i)
	}

	if ring.Len() != 5 {
		t.Fatalf("len = %d, want 5 (capacity)", ring.Len())
	}
}

func TestLog_FilterByLevel(t *testing.T) {
	ring := djinnlog.NewRingHandler(20)
	log := slog.New(ring)

	log.Debug("debug")
	log.Info("info")
	log.Warn("warn")
	log.Error("error")

	errors := ring.Filter(slog.LevelError)
	if len(errors) != 1 {
		t.Fatalf("errors = %d, want 1", len(errors))
	}
}

func TestLog_PerfAttributes(t *testing.T) {
	rtt := djinnlog.RTT(2 * time.Second)
	ttft := djinnlog.TTFT(100 * time.Millisecond)
	throughput := djinnlog.Throughput(100, 2*time.Second)

	if rtt.Key != "rtt" {
		t.Fatalf("rtt key = %q", rtt.Key)
	}
	if ttft.Key != "ttft" {
		t.Fatalf("ttft key = %q", ttft.Key)
	}
	tps := throughput.Value.Float64()
	if tps < 49 || tps > 51 {
		t.Fatalf("throughput = %f, want ~50", tps)
	}
}
