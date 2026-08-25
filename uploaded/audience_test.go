package uploaded

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/mediocregopher/radix/v4"
	"github.com/prebid/openrtb/v20/openrtb2"
)

type countingRedisClient struct {
	radix.Client
	calls int
}

func (c *countingRedisClient) Addr() net.Addr { return c.Client.Addr() }

func (c *countingRedisClient) Do(ctx context.Context, action radix.Action) error {
	c.calls++
	return c.Client.Do(ctx, action)
}

func TestUploadManyWithTTLWritesAtomicallyAndPreservesLongerRetention(t *testing.T) {
	server := miniredis.RunT(t)
	client, err := (radix.PoolConfig{Size: 1}).New(context.Background(), "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx := context.Background()

	if err := UploadManyWithTTL(ctx, client, 7, "user", []string{"one", "two"}, 30*time.Second); err != nil {
		t.Fatal(err)
	}
	key := uploadName(7, "userid")
	if members, err := server.Members(key); err != nil || len(members) != 2 {
		t.Fatalf("members = %v, err = %v", members, err)
	}
	if ttl := server.TTL(key); ttl != 30*time.Second {
		t.Fatalf("new-key TTL = %s, want 30s", ttl)
	}

	server.SetTTL(key, time.Minute)
	if err := UploadSingleWithTTL(ctx, client, 7, "user", "three", 30*time.Second); err != nil {
		t.Fatal(err)
	}
	if ttl := server.TTL(key); ttl != time.Minute {
		t.Fatalf("longer TTL = %s, want 1m preserved", ttl)
	}

	server.SetTTL(key, 0)
	if err := UploadSingleWithTTL(ctx, client, 7, "user", "four", 30*time.Second); err != nil {
		t.Fatal(err)
	}
	if ttl := server.TTL(key); ttl != 30*time.Second {
		t.Fatalf("persistent-key TTL = %s, want 30s", ttl)
	}
}

func TestUploadAudienceTTLConfigurationAndValidation(t *testing.T) {
	original := DefaultAudienceTTL()
	defer func() {
		if err := SetDefaultAudienceTTL(original); err != nil {
			t.Fatal(err)
		}
	}()
	if err := SetDefaultAudienceTTL(90 * time.Second); err != nil {
		t.Fatal(err)
	}
	if got := DefaultAudienceTTL(); got != 90*time.Second {
		t.Fatalf("default TTL = %s, want 90s", got)
	}
	if err := SetDefaultAudienceTTL(0); err == nil {
		t.Fatal("zero TTL should fail")
	}
	if err := UploadManyWithTTL(context.Background(), nil, 1, "user", []string{"one"}, time.Minute); err == nil {
		t.Fatal("nil Redis client should fail")
	}
}

func TestDeleteAudienceDataIsScopedAndDoesNotExposeContents(t *testing.T) {
	server := miniredis.RunT(t)
	client, err := (radix.PoolConfig{Size: 1}).New(context.Background(), "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx := context.Background()
	if err := UploadManyWithTTL(ctx, client, 7, "user", []string{"one", "two"}, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := UploadSingleWithTTL(ctx, client, 8, "user", "other-advertiser", time.Minute); err != nil {
		t.Fatal(err)
	}
	removed, err := DeleteAudienceIdentifier(ctx, client, 7, "user", "one")
	if err != nil || !removed {
		t.Fatalf("identifier delete = %t, err = %v", removed, err)
	}
	one, err := server.SIsMember(uploadName(7, "userid"), "one")
	if err != nil {
		t.Fatal(err)
	}
	two, err := server.SIsMember(uploadName(7, "userid"), "two")
	if err != nil {
		t.Fatal(err)
	}
	if one || !two {
		members, _ := server.Members(uploadName(7, "userid"))
		t.Fatalf("advertiser 7 members = %v", members)
	}
	removed, err = DeleteAudience(ctx, client, 7, "user")
	if err != nil || !removed {
		t.Fatalf("audience delete = %t, err = %v", removed, err)
	}
	if server.Exists(uploadName(7, "userid")) || !server.Exists(uploadName(8, "userid")) {
		t.Fatal("audience deletion escaped advertiser/marker scope")
	}
}

func TestAudienceMarkersCanonicalizeAndReadLegacyAliases(t *testing.T) {
	server := miniredis.RunT(t)
	client, err := (radix.PoolConfig{Size: 1}).New(context.Background(), "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx := context.Background()

	if err := UploadSingleWithTTL(ctx, client, 7, " BuyerID ", "canonical", time.Minute); err != nil {
		t.Fatal(err)
	}
	if !server.Exists(uploadName(7, "buyeruid")) || server.Exists(uploadName(7, "buyerid")) {
		t.Fatalf("canonical/legacy keys = %t/%t", server.Exists(uploadName(7, "buyeruid")), server.Exists(uploadName(7, "buyerid")))
	}
	if err := UploadSingleWithTTL(ctx, client, 7, "arbitrary", "bad", time.Minute); err == nil {
		t.Fatal("unsupported marker was accepted")
	}

	server.SAdd(uploadName(7, "buyerid"), "legacy")
	audience := &UploadAudience{Uploads: uint32(1 << UploadBuyerUID)}
	matched, err := audience.Has(ctx, client, &openrtb2.BidRequest{User: &openrtb2.User{BuyerUID: "legacy"}}, 7)
	if err != nil || !matched {
		t.Fatalf("legacy buyerid match = %t, err = %v", matched, err)
	}
	removed, err := DeleteAudience(ctx, client, 7, "buyeruid")
	if err != nil || !removed || server.Exists(uploadName(7, "buyeruid")) || server.Exists(uploadName(7, "buyerid")) {
		t.Fatalf("alias family delete = %t, err = %v", removed, err)
	}
}

func TestAudienceAliasMembershipUsesOneRedisCommand(t *testing.T) {
	server := miniredis.RunT(t)
	client, err := (radix.PoolConfig{Size: 1}).New(context.Background(), "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx := context.Background()
	audience := &UploadAudience{}
	server.SAdd(uploadName(7, "buyeruid"), "canonical")
	server.SAdd(uploadName(7, "buyerid"), "legacy")

	for name, target := range map[string]string{
		"canonical hit": "canonical",
		"legacy hit":    "legacy",
		"miss":          "missing",
	} {
		t.Run(name, func(t *testing.T) {
			counting := &countingRedisClient{Client: client}
			matched, err := audience.findUploaded(ctx, counting, 7, "buyeruid", target)
			if err != nil {
				t.Fatal(err)
			}
			if want := target != "missing"; matched != want {
				t.Fatalf("matched = %t, want %t", matched, want)
			}
			if counting.calls != 1 {
				t.Fatalf("Redis calls = %d, want 1", counting.calls)
			}
		})
	}
}
