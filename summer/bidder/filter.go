package bidder

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/genelet/winter/summer"
)

type Filter struct {
	summer.Filter
}

const (
	defaultTimeoutMS = 100
	maxTimeoutMS     = 5000
)

func (self *Filter) Preset() error {
	if err := self.Filter.Preset(); err != nil {
		return err
	}

	if self.RoleValue != "admin" {
		ARGS := self.R.Form
		ARGS.Del("synthetic_campaign_id")
		ARGS.Del("synthetic_item_id")
		ARGS.Del("synthetic_creative_id")
		ARGS.Del("credential_ref")
		ARGS.Del("credential_status")
		ARGS.Del("active")
	}
	return validateEndpointFields(self.R.Form, self.Action)
}

func validateEndpointFields(form url.Values, action string) error {
	if action != "insert" && action != "update" {
		return nil
	}
	if err := validateEndpointURL(form, action); err != nil {
		return err
	}
	return normalizeTimeout(form, action)
}

func validateEndpointURL(form url.Values, action string) error {
	raw, ok := form["endpoint_url"]
	if !ok {
		return nil
	}
	endpoint := ""
	if len(raw) > 0 {
		endpoint = raw[0]
	}
	if endpoint == "" {
		if action == "insert" {
			return fmt.Errorf("endpoint_url is required")
		}
		return nil
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" {
		return fmt.Errorf("endpoint_url must be an absolute http or https URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("endpoint_url must use http or https")
	}
	if parsed.User != nil {
		return fmt.Errorf("endpoint_url must not contain user info")
	}
	if host := parsed.Hostname(); host == "" {
		return fmt.Errorf("endpoint_url must include a host")
	} else if ip := net.ParseIP(host); ip == nil && !validEndpointHostname(host) {
		return fmt.Errorf("endpoint_url host is invalid")
	}
	return nil
}

func validEndpointHostname(host string) bool {
	for _, r := range host {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func normalizeTimeout(form url.Values, action string) error {
	raw, ok := form["timeout_ms"]
	if !ok {
		if action == "insert" {
			form.Set("timeout_ms", strconv.Itoa(defaultTimeoutMS))
		}
		return nil
	}
	timeout := ""
	if len(raw) > 0 {
		timeout = raw[0]
	}
	if timeout == "" {
		if action == "insert" {
			form.Set("timeout_ms", strconv.Itoa(defaultTimeoutMS))
		}
		return nil
	}
	n, err := strconv.Atoi(timeout)
	if err != nil || n <= 0 || n > maxTimeoutMS {
		return fmt.Errorf("timeout_ms must be between 1 and %d", maxTimeoutMS)
	}
	form.Set("timeout_ms", strconv.Itoa(n))
	return nil
}

func (self *Filter) Before(model *Model, extra url.Values, nextextra url.Values) error {
	if err := self.Filter.Before(&model.Model, extra, nextextra); err != nil {
		return err
	}

	if self.RoleValue == "adv" {
		advID := self.R.Form.Get("adv_id")
		extra.Set("adv_id", advID)
		if self.Action == "insert" {
			self.R.Form.Set("adv_id", advID)
			self.R.Form.Set("credential_status", "Missing")
			self.R.Form.Set("active", "No")
		}
	}
	return nil
}

func (self *Filter) After(model *Model) error {
	if err := self.Filter.After(&model.Model); err != nil {
		return err
	}
	if self.RoleValue != "admin" {
		for _, item := range *model.LISTS {
			for _, field := range operatorFields {
				delete(item, field)
			}
		}
	}
	return nil
}

var operatorFields = []string{
	"synthetic_campaign_id",
	"synthetic_item_id",
	"synthetic_creative_id",
	"credential_ref",
}
