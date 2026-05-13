package ledger

import (
	"net/url"
	"testing"
)

func TestIsAdvertiserMiddlemanReportRequiresAdvertiserRoleArgs(t *testing.T) {
	tests := []struct {
		name string
		args url.Values
		want bool
	}{
		{
			name: "advertiser",
			args: url.Values{"adv_id": {"7"}},
			want: true,
		},
		{
			name: "admin with adv filter",
			args: url.Values{"admin_id": {"1"}, "adv_id": {"7"}},
			want: false,
		},
		{
			name: "admin",
			args: url.Values{"admin_id": {"1"}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &Model{}
			model.ARGS = tt.args
			if got := model.isAdvertiserMiddlemanReport(); got != tt.want {
				t.Fatalf("isAdvertiserMiddlemanReport() = %v, want %v", got, tt.want)
			}
		})
	}
}
