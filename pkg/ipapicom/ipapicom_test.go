package ipapicom_test

import (
	"context"
	"testing"

	"github.com/D00Movenok/BounceBack/pkg/ipapicom"
	"github.com/stretchr/testify/require"
)

func TestGetLocation(t *testing.T) {
	type args struct {
		ip string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *ipapicom.Location
	}{
		{
			"ipv4 success",
			args{
				ip: "1.1.1.1",
			},
			false,
			&ipapicom.Location{
				As:      "AS13335 Cloudflare, Inc.",
				Isp:     "Cloudflare, Inc",
				Message: "",
				Org:     "APNIC and Cloudflare DNS Resolver project",
				Query:   "1.1.1.1",
				Status:  "success",
			},
		},
		{
			"ipv6 success",
			args{
				ip: "2606:4700:4700::1111",
			},
			false,
			&ipapicom.Location{
				As:      "AS13335 Cloudflare, Inc.",
				Isp:     "Cloudflare, Inc.",
				Message: "",
				Org:     "Cloudflare, Inc.",
				Query:   "2606:4700:4700::1111",
				Status:  "success",
			},
		},
		{
			"bad ip fail",
			args{
				ip: "1.2.3.4.5",
			},
			true,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ipapicom.NewClient()
			ctx := context.Background()
			l, err := c.GetLocationForIP(ctx, tt.args.ip)
			require.Equalf(
				t,
				tt.wantErr,
				err != nil,
				"GetLocationForIP() get error: %s",
				err,
			)

			// partially compare fields, because other often changes
			if tt.want != nil {
				require.Equal(
					t,
					tt.want.As,
					l.As,
					"GetLocationForIP() ip AS mismatch",
				)
				require.Equal(
					t,
					tt.want.Isp,
					l.Isp,
					"GetLocationForIP() ip Isp mismatch",
				)
				require.Equal(
					t,
					tt.want.Message,
					l.Message,
					"GetLocationForIP() ip Message mismatch",
				)
				require.Equal(
					t,
					tt.want.Org,
					l.Org,
					"GetLocationForIP() ip Org mismatch",
				)
				require.Equal(
					t,
					tt.want.Query,
					l.Query,
					"GetLocationForIP() ip Query mismatch",
				)
				require.Equal(
					t,
					tt.want.Status,
					l.Status,
					"GetLocationForIP() ip Status mismatch",
				)
			}
		})
	}
}
