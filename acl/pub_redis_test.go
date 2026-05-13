package acl

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/mediocregopher/radix/v4"
)

type redisCommandRecorder struct {
	commands []string
}

func (r *redisCommandRecorder) Addr() net.Addr {
	return redisRecorderAddr("redis-recorder")
}

func (r *redisCommandRecorder) Do(_ context.Context, action radix.Action) error {
	r.commands = append(r.commands, fmt.Sprint(action))
	return nil
}

func (r *redisCommandRecorder) Close() error {
	return nil
}

type redisRecorderAddr string

func (a redisRecorderAddr) Network() string { return "test" }
func (a redisRecorderAddr) String() string  { return string(a) }

func TestPubToRedisUpdatesPubmapAndDirectPubByID(t *testing.T) {
	pub := &Pub{
		PubID:  42,
		Active: true,
		Sites:  map[string]uint32{"example.com": 7},
		Slots:  map[uint32]map[string]uint32{7: {"leaderboard": 99}},
	}
	redis := &redisCommandRecorder{}

	if err := pub.ToRedis(context.Background(), redis, "pub.example"); err != nil {
		t.Fatal(err)
	}
	if len(redis.commands) != 2 {
		t.Fatalf("commands = %#v, want 2 commands", redis.commands)
	}
	if !strings.Contains(redis.commands[0], `"HSET" "pubmap" "pub.example"`) {
		t.Fatalf("pubmap command = %s", redis.commands[0])
	}
	if !strings.Contains(redis.commands[1], `"HSET" "pubmap:by-id" "42"`) {
		t.Fatalf("direct by-id command = %s", redis.commands[1])
	}
}

func TestPubToRedisDeletesPubmapAndDirectPubByID(t *testing.T) {
	tests := []struct {
		name string
		pub  *Pub
	}{
		{
			name: "inactive",
			pub:  &Pub{PubID: 42, Active: false},
		},
		{
			name: "limited",
			pub:  &Pub{PubID: 43, Active: true, LimitImps: 10, CurrentImps: 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redis := &redisCommandRecorder{}
			if err := tt.pub.ToRedis(context.Background(), redis, "pub.example"); err != nil {
				t.Fatal(err)
			}
			if len(redis.commands) != 2 {
				t.Fatalf("commands = %#v, want 2 commands", redis.commands)
			}
			if !strings.Contains(redis.commands[0], `"HDEL" "pubmap" "pub.example"`) {
				t.Fatalf("pubmap delete command = %s", redis.commands[0])
			}
			wantByID := fmt.Sprintf(`"HDEL" "pubmap:by-id" "%d"`, tt.pub.PubID)
			if !strings.Contains(redis.commands[1], wantByID) {
				t.Fatalf("direct by-id delete command = %s, want %s", redis.commands[1], wantByID)
			}
		})
	}
}
