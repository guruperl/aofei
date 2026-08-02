package hostedpayment

import "testing"

func TestConfigDisabledIsInert(t *testing.T) {
	if err := (Config{}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestConfigEnabledRequiresSeparateSecretReferencesAndHTTPS(t *testing.T) {
	base := Config{
		Enabled: true, Provider: ProviderStripe, APIBaseURL: "https://api.stripe.com",
		PublicBaseURL: "https://www.w8m.com", APIKeyEnv: "STRIPE_API_KEY",
		WebhookSecretEnv: "STRIPE_WEBHOOK_SECRET",
	}.WithDefaults()
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := base
	bad.WebhookSecretEnv = bad.APIKeyEnv
	if err := bad.Validate(); err == nil {
		t.Fatal("shared API/webhook secret reference succeeded")
	}
	bad = base
	bad.PublicBaseURL = "http://w8m.com"
	if err := bad.Validate(); err == nil {
		t.Fatal("cleartext public URL succeeded")
	}
	bad = base
	bad.APIBaseURL = "https://payments.example.com"
	if err := bad.Validate(); err == nil {
		t.Fatal("non-Stripe remote API origin succeeded")
	}
	bad = base
	bad.WebhookPreviousSecretEnv = bad.WebhookSecretEnv
	if err := bad.Validate(); err == nil {
		t.Fatal("reused previous webhook secret reference succeeded")
	}
}
