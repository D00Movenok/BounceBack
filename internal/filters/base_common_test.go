package filters_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"testing"
	"time"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/filters"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock is a mock entity wrapper.
type MockEntity struct {
	mock.Mock
}

func (m *MockEntity) GetIP() netip.Addr {
	args := m.Called()
	return args.Get(0).(netip.Addr)
}

func (m *MockEntity) GetRaw() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1) //nolint: wrapcheck // mock
}

func (m *MockEntity) GetBody() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1) //nolint: wrapcheck // mock
}

func (m *MockEntity) GetCookies() ([]*http.Cookie, error) {
	args := m.Called()
	//nolint: wrapcheck // mock
	return args.Get(0).([]*http.Cookie), args.Error(1)
}

func (m *MockEntity) GetHeaders() (map[string][]string, error) {
	args := m.Called()
	//nolint: wrapcheck // mock
	return args.Get(0).(map[string][]string), args.Error(1)
}

func (m *MockEntity) GetURL() (*url.URL, error) {
	args := m.Called()
	return args.Get(0).(*url.URL), args.Error(1) //nolint: wrapcheck // mock
}

func (m *MockEntity) GetMethod() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func mod(x int, y int) int {
	return (x%y + y) % y
}

func getWeekdayName(offset int) string {
	names := []string{
		"Sunday",
		"Monday",
		"Tuesday",
		"Wednesday",
		"Thursday",
		"Friday",
		"Saturday",
	}
	ls := len(names)
	nw := int(time.Now().In(time.UTC).Weekday())
	o := mod(nw+offset, ls)
	return names[o]
}

func getTime(hourOffset int, minuteOffset int) string {
	now := time.Now().In(time.UTC)
	hour := mod(now.Hour()+hourOffset, 24)
	minute := mod(now.Minute()+minuteOffset, 60)
	return fmt.Sprintf("%02d:%02d", hour, minute)
}

func TestBase_RegexpFilter(t *testing.T) {
	type args struct {
		raw       []byte
		getRawErr error
		cfg       common.FilterConfig
	}
	type want struct {
		res       bool
		createErr bool
		applyErr  bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"regexp true",
			args{
				raw:       []byte("test of that nice filter with two word"),
				getRawErr: nil,
				cfg: common.FilterConfig{
					Name: "test",
					Type: "regexp",
					Params: map[string]any{
						"list": "../../testdata/words_lists/banlist_1.txt",
					},
				},
			},
			want{
				res:       true,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"regexp false",
			args{
				raw:       []byte("test of that nice filter with one word"),
				getRawErr: nil,
				cfg: common.FilterConfig{
					Name: "test",
					Type: "regexp",
					Params: map[string]any{
						"list": "../../testdata/words_lists/banlist_1.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"regexp err can't open file",
			args{
				raw:       []byte("test of that nice filter with two word"),
				getRawErr: errors.New("some error"),
				cfg: common.FilterConfig{
					Name: "test",
					Type: "regexp",
					Params: map[string]any{
						"list": "../../testdata/words_lists/banlist_1337.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: true,
				applyErr:  false,
			},
		},
		{
			"regexp err can't parse regexp",
			args{
				raw:       []byte("test of that nice filter with two word"),
				getRawErr: errors.New("some error"),
				cfg: common.FilterConfig{
					Name: "test",
					Type: "regexp",
					Params: map[string]any{
						"list": "../../testdata/words_lists/broken_regexp.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: true,
				applyErr:  false,
			},
		},
		{
			"regexp err GetRaw",
			args{
				raw:       []byte("test of that nice filter with two word"),
				getRawErr: errors.New("some error"),
				cfg: common.FilterConfig{
					Name: "test",
					Type: "regexp",
					Params: map[string]any{
						"list": "../../testdata/words_lists/banlist_1.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: false,
				applyErr:  true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := filters.NewRegexFilter(
				nil,
				filters.FilterSet{},
				tt.args.cfg,
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewRegexFilter() error mismatch: %s",
				err,
			)

			if !tt.want.createErr {
				e := new(MockEntity)
				e.On("GetRaw").Return(tt.args.raw, tt.args.getRawErr)

				res, err := filter.Apply(e, log.Logger)
				require.Equalf(
					t,
					tt.want.applyErr,
					err != nil,
					"Apply() error mismatch: %s",
					err,
				)
				require.Equal(
					t,
					tt.want.res,
					res,
					"Apply() result mismatch",
				)
				e.AssertExpectations(t)
			}
		})
	}
}

func TestBase_IPFilter(t *testing.T) {
	type args struct {
		ip  string
		cfg common.FilterConfig
	}
	type want struct {
		res       bool
		createErr bool
		applyErr  bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"ip filter true ip v4",
			args{
				ip: "3.3.3.3",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../testdata/ip_lists/allowlist_1.txt",
					},
				},
			},
			want{
				res:       true,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"ip filter true ip v6",
			args{
				ip: "aaaa:aaaa:aaaa:aaaa:aaaa:aaaa:aaaa:aaaa",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../testdata/ip_lists/allowlist_1.txt",
					},
				},
			},
			want{
				res:       true,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"ip filter true subnet v4",
			args{
				ip: "2.2.3.4",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../testdata/ip_lists/allowlist_1.txt",
					},
				},
			},
			want{
				res:       true,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"ip filter true subnet v6",
			args{
				ip: "fe80:aaaa:aaaa:aaaa:aaaa:aaaa:aaaa:aaaa",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../testdata/ip_lists/allowlist_1.txt",
					},
				},
			},
			want{
				res:       true,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"ip filter false v4",
			args{
				ip: "5.5.5.5",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../testdata/ip_lists/allowlist_1.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"ip filter false v6",
			args{
				ip: "bbbb:bbbb:bbbb:bbbb:bbbb:bbbb:bbbb:bbbb",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../testdata/ip_lists/allowlist_1.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"ip err can't open list",
			args{
				ip: "5.5.5.5",
				cfg: common.FilterConfig{
					Name:   "test",
					Type:   "ip",
					Params: map[string]any{},
				},
			},
			want{
				res:       false,
				createErr: true,
				applyErr:  false,
			},
		},
		{
			"ip err can't parse ip v4 list",
			args{
				ip: "5.5.5.5",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../testdata/ip_lists/broken_ip_v4.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: true,
				applyErr:  false,
			},
		},
		{
			"ip err can't parse ip v6 list",
			args{
				ip: "aaaa:aaaa:aaaa:aaaa:aaaa:aaaa:aaaa",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../testdata/ip_lists/broken_ip_v6.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: true,
				applyErr:  false,
			},
		},
		{
			"ip err can't parse subnet ip v4 list",
			args{
				ip: "5.5.5.5",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../testdata/ip_lists/broken_subnet_v4.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: true,
				applyErr:  false,
			},
		},
		{
			"ip err can't parse subnet ip v6 list",
			args{
				ip: "aaaa:aaaa:aaaa:aaaa:aaaa:aaaa:aaaa",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../testdata/ip_lists/broken_subnet_v6.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: true,
				applyErr:  false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := filters.NewIPFilter(
				nil,
				filters.FilterSet{},
				tt.args.cfg,
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewIPFilter() error mismatch: %s",
				err,
			)

			if !tt.want.createErr {
				e := new(MockEntity)
				e.On("GetIP").Return(netip.MustParseAddr(tt.args.ip))

				res, err := filter.Apply(e, log.Logger)
				require.Equalf(
					t,
					tt.want.applyErr,
					err != nil,
					"Apply() error mismatch: %s",
					err,
				)
				require.Equal(
					t,
					tt.want.res,
					res,
					"Apply() result mismatch",
				)
				e.AssertExpectations(t)
			}
		})
	}
}

// test ignores timezone.
func TestBase_TimeFilter(t *testing.T) {
	type args struct {
		cfg common.FilterConfig
	}
	type want struct {
		res       bool
		createErr bool
		applyErr  bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"time hour true",
			args{
				cfg: common.FilterConfig{
					Name: "test",
					Type: "time",
					Params: map[string]any{
						"from":     getTime(-1, 0),
						"to":       getTime(1, 0),
						"weekdays": []string{},
					},
				},
			},
			want{
				res:       true,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"time minute true",
			args{
				cfg: common.FilterConfig{
					Name: "test",
					Type: "time",
					Params: map[string]any{
						"from":     getTime(0, -10),
						"to":       getTime(0, 10),
						"weekdays": []string{},
					},
				},
			},
			want{
				res:       true,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"time weekday true",
			args{
				cfg: common.FilterConfig{
					Name: "test",
					Type: "time",
					Params: map[string]any{
						"from": getTime(0, -10),
						"to":   getTime(0, 10),
						"weekdays": []string{
							getWeekdayName(0),
						},
					},
				},
			},
			want{
				res:       true,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"time between days true",
			args{
				cfg: common.FilterConfig{
					Name: "test",
					Type: "time",
					Params: map[string]any{
						"from": getTime(0, -1),
						"to":   getTime(23, 59),
						"weekdays": []string{
							getWeekdayName(-1),
							getWeekdayName(0),
							getWeekdayName(1),
						},
					},
				},
			},
			want{
				res:       true,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"time hour false",
			args{
				cfg: common.FilterConfig{
					Name: "test",
					Type: "time",
					Params: map[string]any{
						"from":     getTime(-2, 0),
						"to":       getTime(-1, 0),
						"weekdays": []string{},
					},
				},
			},
			want{
				res:       false,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"time minute false",
			args{
				cfg: common.FilterConfig{
					Name: "test",
					Type: "time",
					Params: map[string]any{
						"from":     getTime(0, -20),
						"to":       getTime(0, -10),
						"weekdays": []string{},
					},
				},
			},
			want{
				res:       false,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"time weekday false",
			args{
				cfg: common.FilterConfig{
					Name: "test",
					Type: "time",
					Params: map[string]any{
						"from": getTime(0, -10),
						"to":   getTime(0, 10),
						"weekdays": []string{
							getWeekdayName(1),
						},
					},
				},
			},
			want{
				res:       false,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"time between days false",
			args{
				cfg: common.FilterConfig{
					Name: "test",
					Type: "time",
					Params: map[string]any{
						"from": getTime(-23, -59),
						"to":   getTime(0, 1),
						"weekdays": []string{
							getWeekdayName(-1),
						},
					},
				},
			},
			want{
				res:       false,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"time bad from",
			args{
				cfg: common.FilterConfig{
					Name: "test",
					Type: "time",
					Params: map[string]any{
						"from": "",
						"to":   getTime(0, 10),
						"weekdays": []string{
							getWeekdayName(1),
						},
					},
				},
			},
			want{
				res:       false,
				createErr: true,
				applyErr:  false,
			},
		},
		{
			"time bad to",
			args{
				cfg: common.FilterConfig{
					Name: "test",
					Type: "time",
					Params: map[string]any{
						"from": getTime(0, -10),
						"to":   "",
						"weekdays": []string{
							getWeekdayName(1),
						},
					},
				},
			},
			want{
				res:       false,
				createErr: true,
				applyErr:  false,
			},
		},
		{
			"time bad weekday",
			args{
				cfg: common.FilterConfig{
					Name: "test",
					Type: "time",
					Params: map[string]any{
						"from": getTime(0, -10),
						"to":   getTime(0, 10),
						"weekdays": []string{
							"some bad weekday",
						},
					},
				},
			},
			want{
				res:       false,
				createErr: true,
				applyErr:  false,
			},
		},
		{
			"time bad timezone",
			args{
				cfg: common.FilterConfig{
					Name: "test",
					Type: "time",
					Params: map[string]any{
						"from":     getTime(0, -10),
						"to":       getTime(0, 10),
						"weekdays": []string{},
						"timezone": "some bad timezone",
					},
				},
			},
			want{
				res:       false,
				createErr: true,
				applyErr:  false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := filters.NewTimeFilter(
				nil,
				filters.FilterSet{},
				tt.args.cfg,
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewTimeFilter() error mismatch: %s",
				err,
			)

			if !tt.want.createErr {
				res, err := filter.Apply(nil, log.Logger)
				require.Equalf(t,
					tt.want.applyErr,
					err != nil,
					"Apply() error mismatch: %s",
					err,
				)
				require.Equal(t,
					tt.want.res,
					res,
					"Apply() result mismatch",
				)
			}
		})
	}
}

func TestBase_GeoFilter(t *testing.T) {
	type args struct {
		ip  string
		cfg common.FilterConfig
	}
	type want struct {
		res       bool
		createErr bool
		applyErr  bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"geo array true",
			args{
				ip: "8.8.8.8",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"geolocations": []map[string][]string{
							{
								"organisation": []string{"(?i)google"},
							},
						},
					},
				},
			},
			want{
				res:       true,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"geo string true",
			args{
				ip: "8.8.8.8",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"geolocations": []map[string][]string{
							{
								"country": []string{"(?i)united states"},
							},
						},
					},
				},
			},
			want{
				res:       true,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"geo array false",
			args{
				ip: "1.1.1.1",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"geolocations": []map[string][]string{
							{
								"organisation": []string{"(?i)google"},
							},
						},
					},
				},
			},
			want{
				res:       false,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"geo string false",
			args{
				ip: "1.1.1.1",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"geolocations": []map[string][]string{
							{
								"organisation": []string{"(?i)google"},
							},
						},
					},
				},
			},
			want{
				res:       false,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"geo empty false",
			args{
				ip: "1.1.1.1",
				cfg: common.FilterConfig{
					Name:   "test",
					Type:   "geo",
					Params: map[string]any{},
				},
			},
			want{
				res:       false,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"geo err bad regexp",
			args{
				ip: "1.1.1.1",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"geolocations": []map[string][]string{
							{
								"organisation": []string{"(?i"},
							},
						},
					},
				},
			},
			want{
				res:       false,
				createErr: true,
				applyErr:  false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := database.New("", true)
			require.NoError(t, err, "can't create db")
			filter, err := filters.NewGeolocationFilter(
				db,
				filters.FilterSet{},
				tt.args.cfg,
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewGeolocationFilter() error mismatch: %s",
				err,
			)

			if !tt.want.createErr {
				const geolocationInfoCount = 2
				for i := 0; i < geolocationInfoCount; i++ {
					e := new(MockEntity)
					e.On("GetIP").Return(netip.MustParseAddr(tt.args.ip))

					res, err := filter.Apply(e, log.Logger)
					require.Equalf(
						t,
						tt.want.applyErr,
						err != nil,
						"Apply() error mismatch: %s",
						err,
					)
					require.Equal(
						t,
						tt.want.res,
						res,
						"Apply() result mismatch",
					)
					e.AssertExpectations(t)

					// clear db so all geo getters will be tested
					err = db.DB.DropAll()
					require.NoError(t, err, "Can't clear db")
				}
			}
		})
	}
}

func TestBase_ReverseLookupFilter(t *testing.T) {
	type args struct {
		ip  string
		cfg common.FilterConfig
	}
	type want struct {
		res       bool
		createErr bool
		applyErr  bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"reverse lookup true",
			args{
				ip: "1.1.1.1",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "reverse_lookup",
					Params: map[string]any{
						"dns":  "1.1.1.1:53",
						"list": "../../testdata/words_lists/banlist_1.txt",
					},
				},
			},
			want{
				res:       true,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"reverse lookup false",
			args{
				ip: "8.8.8.8",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "reverse_lookup",
					Params: map[string]any{
						"dns":  "1.1.1.1:53",
						"list": "../../testdata/words_lists/banlist_1.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: false,
				applyErr:  false,
			},
		},
		{
			"reverse lookup err can't open file",
			args{
				ip: "1.1.1.1",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "reverse_lookup",
					Params: map[string]any{
						"dns":  "1.1.1.1:53",
						"list": "../../testdata/words_lists/banlist_1337.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: true,
				applyErr:  false,
			},
		},
		{
			"reverse lookup err can't parse regexp",
			args{
				ip: "1.1.1.1",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "reverse_lookup",
					Params: map[string]any{
						"dns":  "1.1.1.1:53",
						"list": "../../testdata/words_lists/broken_regexp.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: true,
				applyErr:  false,
			},
		},
		{
			"reverse lookup err can't parse dns",
			args{
				ip: "1.1.1.1",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "reverse_lookup",
					Params: map[string]any{
						"dns":  "1.1.1.1",
						"list": "../../testdata/words_lists/banlist_1.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: true,
				applyErr:  false,
			},
		},
		{
			"reverse lookup err dead dns",
			args{
				ip: "1.1.1.1",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "reverse_lookup",
					Params: map[string]any{
						"dns":  "1.1.1.1:553",
						"list": "../../testdata/words_lists/banlist_1.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: false,
				applyErr:  true,
			},
		},
		{
			"reverse lookup err unknown ip",
			args{
				ip: "195.168.14.15",
				cfg: common.FilterConfig{
					Name: "test",
					Type: "reverse_lookup",
					Params: map[string]any{
						"dns":  "1.1.1.1:53",
						"list": "../../testdata/words_lists/banlist_1.txt",
					},
				},
			},
			want{
				res:       false,
				createErr: false,
				applyErr:  true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := database.New("", true)
			require.NoError(t, err, "can't create db")
			filter, err := filters.NewReverseLookupFilter(
				db,
				filters.FilterSet{},
				tt.args.cfg,
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewReverseLookupFilter() error mismatch: %s",
				err,
			)

			if !tt.want.createErr {
				e := new(MockEntity)
				e.On("GetIP").Return(netip.MustParseAddr(tt.args.ip))

				res, err := filter.Apply(e, log.Logger)
				require.Equalf(
					t,
					tt.want.applyErr,
					err != nil,
					"Apply() error mismatch: %s",
					err,
				)
				require.Equal(
					t,
					tt.want.res,
					res,
					"Apply() result mismatch",
				)
				e.AssertExpectations(t)

				// clear db so all geo getters will be tested
				err = db.DB.DropAll()
				require.NoError(t, err, "Can't clear db")
			}
		})
	}
}
