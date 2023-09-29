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
				IP:      "1.1.1.1",
				Asn:     "AS13335",
				Org:     "CLOUDFLARENET",
				IsError: false,
				Reason:  "",
			},
		},
		{
			"ipv6 success",
			args{
				ip: "2606:4700:4700::1111",
			},
			false,
			&ipapico.Location{
				IP:      "2606:4700:4700::1111",
				Asn:     "AS13335",
				Org:     "CLOUDFLARENET",
				IsError: false,
				Reason:  "",
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

			// partially compare fields, because other often changes
			if tt.want != nil {
				require.Equal(
					t,
					tt.want.Asn,
					l.Asn,
					"GetLocationForIP() ip Asn mismatch",
				)
				require.Equal(
					t,
					tt.want.Org,
					l.Org,
					"GetLocationForIP() ip Org mismatch",
				)
				require.Equal(
					t,
					tt.want.IsError,
					l.IsError,
					"GetLocationForIP() ip IsError mismatch",
				)
				require.Equal(
					t,
					tt.want.Reason,
					l.Reason,
					"GetLocationForIP() ip Reason mismatch",
				)
			}
		})
	}
}
