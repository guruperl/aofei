package dsp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guruperl/aofei/maxmind"
	"github.com/mediocregopher/radix/v4"
	"github.com/prebid/openrtb/v20/openrtb2"
	"go.uber.org/zap"
)

const smokePublisherDomain = "exchange.example.test"

func TestServeBidSmoke(t *testing.T) {
	controller := newSmokeController(t)
	defer controller.Close()

	body, request := smokeBidRequestBody(t)
	rr := serveSmokeBid(t, controller, smokePublisherDomain, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("ServeBid status = %d, want %d", rr.Code, http.StatusOK)
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.ID != request.ID {
		t.Fatalf("response.ID = %q, want %q", response.ID, request.ID)
	}
	if len(response.SeatBid) != 1 || len(response.SeatBid[0].Bid) != 1 {
		t.Fatalf("response has invalid seatbid shape: %+v", response.SeatBid)
	}
	bid := response.SeatBid[0].Bid[0]
	if bid.ImpID != request.Imp[0].ID {
		t.Fatalf("bid.ImpID = %q, want %q", bid.ImpID, request.Imp[0].ID)
	}
	if bid.AdM == "" {
		t.Fatal("bid.AdM is empty")
	}
	if bid.Price <= 0 {
		t.Fatalf("bid.Price = %f, want positive", bid.Price)
	}
}

func TestServeBidUnknownPublisherSmoke(t *testing.T) {
	controller := newSmokeController(t)
	defer controller.Close()

	body, _ := smokeBidRequestBody(t)
	rr := serveSmokeBid(t, controller, "missing.example.invalid", body)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("ServeBid status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestServeBidMalformedJSONSmoke(t *testing.T) {
	controller := newSmokeController(t)
	defer controller.Close()

	rr := serveSmokeBid(t, controller, smokePublisherDomain, []byte("{"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("ServeBid status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestServeBidOversizedBodySmoke(t *testing.T) {
	controller := newSmokeController(t)
	defer controller.Close()

	body := strings.Repeat(" ", maxBidRequestBodyBytes+1)
	rr := serveSmokeBid(t, controller, smokePublisherDomain, []byte(body))
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("ServeBid status = %d, want %d", rr.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestUnpackURLStringNoBidSmoke(t *testing.T) {
	_, err := UnpackURLString("https://example.test/win?auction_id=${AUCTION_ID}", &openrtb2.BidResponse{ID: "no-bid"})
	if err == nil {
		t.Fatal("expected no-bid response to return an error")
	}
}

func serveSmokeBid(t *testing.T, controller *Controller, domain string, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/bid/"+domain, bytes.NewReader(body))
	req.SetPathValue("domain", domain)
	rr := httptest.NewRecorder()
	controller.ServeBid(rr, req)
	return rr
}

func smokeBidRequestBody(t *testing.T) ([]byte, *openrtb2.BidRequest) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "etc", "samples", "sample_bid.json"))
	if err != nil {
		t.Fatal(err)
	}
	var bid openrtb2.BidRequest
	if err := json.Unmarshal(data, &bid); err != nil {
		t.Fatal(err)
	}
	if bid.Device != nil {
		bid.Device.IP = ""
	}
	data, err = json.Marshal(&bid)
	if err != nil {
		t.Fatal(err)
	}
	return data, &bid
}

func newSmokeController(t *testing.T) *Controller {
	t.Helper()

	configPath, ok := os.LookupEnv("AOFEI")
	if !ok || configPath == "" {
		t.Skip("AOFEI is unset; run ./scripts/aofei-local.sh reset-sample first")
	}
	if _, err := os.Stat(configPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("AOFEI config %s is missing; run ./scripts/aofei-local.sh up", configPath)
		}
		t.Fatal(err)
	}
	config, err := NewConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	red := config.Redis
	poolConfig := radix.PoolConfig{
		Dialer: radix.Dialer{
			AuthUser: red.User,
			AuthPass: red.Pass,
		},
	}
	if red.Size != 0 {
		poolConfig.Size = red.Size
	}
	redis, err := poolConfig.New(context.Background(), red.Network, red.Addr)
	if err != nil {
		t.Fatal(err)
	}
	ips := loadSmokeIPSearch(t, config.Ips)
	return &Controller{
		C:      config,
		Ips:    ips,
		Redis:  redis,
		Logger: zap.NewNop(),
	}
}

func loadSmokeIPSearch(t *testing.T, path string) *maxmind.IPSearch {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var ips maxmind.IPSearch
	if err := json.Unmarshal(data, &ips); err != nil {
		t.Fatal(err)
	}
	return &ips
}
