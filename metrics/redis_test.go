package metrics_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// redisImage provides the RedisBloom commands Tracker uses.
const redisImage = "redis/redis-stack:6.2.6-v17"

var (
	redisAddr string
	redisErr  error
)

// TestMain uses the Redis at REDIS_ADDR, or else starts redisImage with
// docker. Tests that need Redis are skipped when neither is available.
func TestMain(m *testing.M) {
	stop := func() {}
	if redisAddr = os.Getenv("REDIS_ADDR"); redisAddr == "" {
		redisAddr, stop, redisErr = startRedis()
	}
	code := m.Run()
	stop()
	os.Exit(code)
}

func startRedis() (addr string, stop func(), err error) {
	out, err := exec.Command("docker", "run", "-d", "--rm", "-p", "127.0.0.1::6379", redisImage).Output()
	if err != nil {
		return "", func() {}, fmt.Errorf("docker run: %w", err)
	}
	id := strings.TrimSpace(string(out))
	stop = func() { _ = exec.Command("docker", "stop", id).Run() }

	out, err = exec.Command("docker", "port", id, "6379/tcp").Output()
	if err != nil {
		stop()
		return "", func() {}, fmt.Errorf("docker port: %w", err)
	}
	addr = strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])

	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()
	for deadline := time.Now().Add(30 * time.Second); ; time.Sleep(100 * time.Millisecond) {
		if err = client.Ping(context.Background()).Err(); err == nil {
			return addr, stop, nil
		}
		if time.Now().After(deadline) {
			stop()
			return "", func() {}, fmt.Errorf("redis not ready: %w", err)
		}
	}
}

// redisClient returns a client for the test Redis, or skips the test.
func redisClient(t *testing.T) *redis.Client {
	t.Helper()
	if redisErr != nil {
		t.Skipf("redis unavailable: %v", redisErr)
	}
	client := redis.NewClient(&redis.Options{Addr: redisAddr})
	t.Cleanup(func() { _ = client.Close() })
	return client
}
