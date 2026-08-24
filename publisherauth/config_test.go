package publisherauth

import "testing"

func TestConfigDefaultsAndBounds(t *testing.T) {
	config := (Config{}).WithDefaults()
	if config.Enabled || config.RequestSkewSeconds != 300 || config.CredentialRefreshSeconds != 30 ||
		config.CredentialMaxAgeSeconds != 120 || config.RotationMaxOverlapSeconds != 86400 {
		t.Fatalf("defaults = %#v", config)
	}
	if service, err := NewService(config, nil, nil); err != nil || service != nil {
		t.Fatalf("disabled service = %#v, %v", service, err)
	}
	for _, invalid := range []Config{
		{Enabled: true, RequestSkewSeconds: 901, CredentialRefreshSeconds: 30, CredentialMaxAgeSeconds: 120, RotationMaxOverlapSeconds: 1},
		{Enabled: true, RequestSkewSeconds: 300, CredentialRefreshSeconds: 301, CredentialMaxAgeSeconds: 301, RotationMaxOverlapSeconds: 1},
		{Enabled: true, RequestSkewSeconds: 300, CredentialRefreshSeconds: 30, CredentialMaxAgeSeconds: 29, RotationMaxOverlapSeconds: 1},
		{Enabled: true, RequestSkewSeconds: 300, CredentialRefreshSeconds: 30, CredentialMaxAgeSeconds: 120, RotationMaxOverlapSeconds: 604801},
	} {
		if err := invalid.Validate(); err == nil {
			t.Fatalf("invalid config accepted: %#v", invalid)
		}
	}
}
