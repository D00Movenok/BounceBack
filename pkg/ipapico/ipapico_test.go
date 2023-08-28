package ipapico_test

import (
	"context"
	"testing"

	"github.com/D00Movenok/BounceBack/pkg/ipapico"
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
		want    *ipapico.Location
	}{
		{
			"ipv4 success",
			args{
				ip: "1.1.1.1",
			},
			false,
			&ipapico.Location{
				IP:                 "1.1.1.1",
				City:               "Sydney",
				Region:             "New South Wales",
				RegionCode:         "NSW",
				Country:            "AU",
				CountryName:        "Australia",
				ContinentCode:      "OC",
				InEu:               false,
				Postal:             "2000",
				Latitude:           -33.859337,
				Longitude:          151.20363,
				Timezone:           "Australia/Sydney",
				UtcOffset:          "+1000",
				CountryCallingCode: "+61",
				Currency:           "AUD",
				Languages:          "en-AU",
				Asn:                "AS13335",
				Org:                "CLOUDFLARENET",
				IsError:            false,
				Reason:             "",
			},
		},
		{
			"ipv6 success",
			args{
				ip: "2606:4700:4700::1111",
			},
			false,
			&ipapico.Location{
				IP:                 "2606:4700:4700::1111",
				City:               "San Francisco",
				Region:             "California",
				RegionCode:         "CA",
				Country:            "US",
				CountryName:        "United States",
				ContinentCode:      "NA",
				InEu:               false,
				Postal:             "94142",
				Latitude:           37.7809,
				Longitude:          -122.4245,
				Timezone:           "America/Los_Angeles",
				UtcOffset:          "-0700",
				CountryCallingCode: "+1",
				Currency:           "USD",
				Languages:          "en-US,es-US,haw,fr",
				Asn:                "AS13335",
				Org:                "CLOUDFLARENET",
				IsError:            false,
				Reason:             "",
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
			c := ipapico.NewClient()
			ctx := context.Background()
			l, err := c.GetLocationForIP(ctx, tt.args.ip)
			require.Equalf(
				t,
				tt.wantErr,
				err != nil,
				"GetLocationForIP() get error: %s",
				err,
			)
			require.Equal(
				t,
				tt.want,
				l,
				"GetLocationForIP() ip geolocation information mismatch",
			)
		})
	}
}
