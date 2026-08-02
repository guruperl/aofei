package managementapi

import "testing"

func TestConfigRequiresDeploymentKeyOnlyWhenEnabled(t *testing.T) {
	disabled := (Config{}).WithDefaults(900)
	if err := disabled.Validate(); err != nil {
		t.Fatal(err)
	}
	if disabled.CredentialRequestsMinute != 120 || disabled.AccountRequestsMinute != 600 || disabled.CacheActivationSeconds != 900 {
		t.Fatalf("unexpected defaults: %#v", disabled)
	}
	enabled := disabled
	enabled.Enabled = true
	if err := enabled.Validate(); err == nil {
		t.Fatal("enabled API accepted a missing key environment name")
	}
	enabled.KeyEnv = "MANAGEMENT_API_HMAC_KEY"
	if err := enabled.Validate(); err != nil {
		t.Fatal(err)
	}
	enabled.KeyEnv = "MANAGEMENT-API=KEY"
	if err := enabled.Validate(); err == nil {
		t.Fatal("enabled API accepted an invalid key environment name")
	}
}
