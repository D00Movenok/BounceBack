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
				As:          "AS13335 Cloudflare, Inc.",
				City:        "South Brisbane",
				Country:     "Australia",
				CountryCode: "AU",
				Isp:         "Cloudflare, Inc",
				Lat:         -27.4766,
				Lon:         153.0166,
				Message:     "",
				Org:         "APNIC and Cloudflare DNS Resolver project",
				Query:       "1.1.1.1",
				Region:      "QLD",
				RegionName:  "Queensland",
				Status:      "success",
				Timezone:    "Australia/Brisbane",
				Zip:         "4101",
			},
		},
		{
			"ipv6 success",
			args{
				ip: "2606:4700:4700::1111",
			},
			false,
			&ipapicom.Location{
				As:          "AS13335 Cloudflare, Inc.",
				City:        "San Francisco",
				Country:     "United States",
				CountryCode: "US",
				Isp:         "Cloudflare, Inc.",
				Lat:         37.7803,
				Lon:         -122.39,
				Message:     "",
				Org:         "Cloudflare, Inc.",
				Query:       "2606:4700:4700::1111",
				Region:      "CA",
				RegionName:  "California",
				Status:      "success",
				Timezone:    "America/Los_Angeles",
				Zip:         "94107",
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
			require.Equal(t,
				tt.want,
				l,
				"GetLocationForIP() ip geolocation information mismatch",
			)
		})
	}
}
