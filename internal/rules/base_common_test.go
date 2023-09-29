package rules_test

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
	"github.com/D00Movenok/BounceBack/internal/rules"
	"github.com/miekg/dns"
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

func (m *MockEntity) GetQuestions() ([]dns.Question, error) {
	args := m.Called()
	//nolint: wrapcheck // mock
	return args.Get(0).([]dns.Question), args.Error(1)
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

func getTime(hourOffset time.Duration, minuteOffset time.Duration) string {
	now := time.Now().In(time.UTC).
		Add(hourOffset * time.Hour).
		Add(minuteOffset * time.Minute)
	return fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
}

func TestBase_RegexpRule(t *testing.T) {
	type args struct {
		raw       []byte
		getRawErr error
		cfg       common.RuleConfig
	}
	type want struct {
		res        bool
		createErr  bool
		prepareErr bool
		applyErr   bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"regexp true",
			args{
				raw:       []byte("test of that nice rule with two word"),
				getRawErr: nil,
				cfg: common.RuleConfig{
					Name: "test",
					Type: "regexp",
					Params: map[string]any{
						"list": "../../test/testdata/words_lists/banlist_regexp.txt",
					},
				},
			},
			want{
				res:        true,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"regexp false",
			args{
				raw:       []byte("test of that nice rule with one word"),
				getRawErr: nil,
				cfg: common.RuleConfig{
					Name: "test",
					Type: "regexp",
					Params: map[string]any{
						"list": "../../test/testdata/words_lists/banlist_regexp.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"regexp err can't open file",
			args{
				raw:       []byte("test of that nice rule with two word"),
				getRawErr: errors.New("some error"),
				cfg: common.RuleConfig{
					Name: "test",
					Type: "regexp",
					Params: map[string]any{
						"list": "../../test/testdata/words_lists/banlist_1337.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"regexp err can't parse regexp",
			args{
				raw:       []byte("test of that nice rule with two word"),
				getRawErr: errors.New("some error"),
				cfg: common.RuleConfig{
					Name: "test",
					Type: "regexp",
					Params: map[string]any{
						"list": "../../test/testdata/words_lists/broken_regexp.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"regexp err GetRaw",
			args{
				raw:       []byte("test of that nice rule with two word"),
				getRawErr: errors.New("some error"),
				cfg: common.RuleConfig{
					Name: "test",
					Type: "regexp",
					Params: map[string]any{
						"list": "../../test/testdata/words_lists/banlist_regexp.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  false,
				prepareErr: false,
				applyErr:   true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := rules.NewRegexpRule(
				nil,
				rules.RuleSet{},
				tt.args.cfg,
				common.Globals{},
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewRegexpRule() error mismatch: %s",
				err,
			)

			if !tt.want.createErr {
				e := new(MockEntity)
				e.On("GetRaw").Return(tt.args.raw, tt.args.getRawErr)

				err = rule.Prepare(e, log.Logger)
				require.Equalf(
					t,
					tt.want.prepareErr,
					err != nil,
					"Prepare() error mismatch: %s",
					err,
				)

				res, err := rule.Apply(e, log.Logger)
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

func TestBase_IPRule(t *testing.T) {
	type args struct {
		ip  string
		cfg common.RuleConfig
	}
	type want struct {
		res        bool
		createErr  bool
		prepareErr bool
		applyErr   bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"ip rule true ip v4",
			args{
				ip: "3.3.3.3",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../test/testdata/ip_lists/allowlist_1.txt",
					},
				},
			},
			want{
				res:        true,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"ip rule true ip v6",
			args{
				ip: "aaaa:aaaa:aaaa:aaaa:aaaa:aaaa:aaaa:aaaa",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../test/testdata/ip_lists/allowlist_1.txt",
					},
				},
			},
			want{
				res:        true,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"ip rule true subnet v4",
			args{
				ip: "2.2.3.4",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../test/testdata/ip_lists/allowlist_1.txt",
					},
				},
			},
			want{
				res:        true,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"ip rule true subnet v6",
			args{
				ip: "fe80:aaaa:aaaa:aaaa:aaaa:aaaa:aaaa:aaaa",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../test/testdata/ip_lists/allowlist_1.txt",
					},
				},
			},
			want{
				res:        true,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"ip rule false v4",
			args{
				ip: "5.5.5.5",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../test/testdata/ip_lists/allowlist_1.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"ip rule false v6",
			args{
				ip: "bbbb:bbbb:bbbb:bbbb:bbbb:bbbb:bbbb:bbbb",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../test/testdata/ip_lists/allowlist_1.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"ip err can't open list",
			args{
				ip: "5.5.5.5",
				cfg: common.RuleConfig{
					Name:   "test",
					Type:   "ip",
					Params: map[string]any{},
				},
			},
			want{
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"ip err can't parse ip v4 list",
			args{
				ip: "5.5.5.5",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../test/testdata/ip_lists/broken_ip_v4.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"ip err can't parse ip v6 list",
			args{
				ip: "aaaa:aaaa:aaaa:aaaa:aaaa:aaaa:aaaa",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../test/testdata/ip_lists/broken_ip_v6.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"ip err can't parse subnet ip v4 list",
			args{
				ip: "5.5.5.5",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../test/testdata/ip_lists/broken_subnet_v4.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"ip err can't parse subnet ip v6 list",
			args{
				ip: "aaaa:aaaa:aaaa:aaaa:aaaa:aaaa:aaaa",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "ip",
					Params: map[string]any{
						"list": "../../test/testdata/ip_lists/broken_subnet_v6.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := rules.NewIPRule(
				nil,
				rules.RuleSet{},
				tt.args.cfg,
				common.Globals{},
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewIPRule() error mismatch: %s",
				err,
			)

			if !tt.want.createErr {
				e := new(MockEntity)
				e.On("GetIP").Return(netip.MustParseAddr(tt.args.ip))

				err = rule.Prepare(e, log.Logger)
				require.Equalf(
					t,
					tt.want.prepareErr,
					err != nil,
					"Prepare() error mismatch: %s",
					err,
				)

				res, err := rule.Apply(e, log.Logger)
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
func TestBase_TimeRule(t *testing.T) {
	type args struct {
		cfg common.RuleConfig
	}
	type want struct {
		res        bool
		createErr  bool
		prepareErr bool
		applyErr   bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"time hour true",
			args{
				cfg: common.RuleConfig{
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
				res:        true,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"time minute true",
			args{
				cfg: common.RuleConfig{
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
				res:        true,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"time weekday true",
			args{
				cfg: common.RuleConfig{
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
				res:        true,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"time between days true",
			args{
				cfg: common.RuleConfig{
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
				res:        true,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"time hour false",
			args{
				cfg: common.RuleConfig{
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
				res:        false,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"time minute false",
			args{
				cfg: common.RuleConfig{
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
				res:        false,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"time weekday false",
			args{
				cfg: common.RuleConfig{
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
				res:        false,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"time between days false",
			args{
				cfg: common.RuleConfig{
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
				res:        false,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"time bad from",
			args{
				cfg: common.RuleConfig{
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
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"time bad to",
			args{
				cfg: common.RuleConfig{
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
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"time bad weekday",
			args{
				cfg: common.RuleConfig{
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
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"time bad timezone",
			args{
				cfg: common.RuleConfig{
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
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := rules.NewTimeRule(
				nil,
				rules.RuleSet{},
				tt.args.cfg,
				common.Globals{},
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewTimeRule() error mismatch: %s",
				err,
			)

			if !tt.want.createErr {
				err = rule.Prepare(nil, log.Logger)
				require.Equalf(
					t,
					tt.want.prepareErr,
					err != nil,
					"Prepare() error mismatch: %s",
					err,
				)

				res, err := rule.Apply(nil, log.Logger)
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

func TestBase_GeoRule(t *testing.T) {
	type args struct {
		ip  string
		cfg common.RuleConfig
	}
	type want struct {
		res        bool
		createErr  bool
		prepareErr bool
		applyErr   bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"geo geolocations array true",
			args{
				ip: "8.8.8.8",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"list": "",
						"geolocations": []map[string][]string{
							{
								"organisation": []string{"(?i)google"},
							},
						},
					},
				},
			},
			want{
				res:        true,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"geo geolocations string true",
			args{
				ip: "8.8.8.8",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"list": "",
						"geolocations": []map[string][]string{
							{
								"country": []string{"(?i)united states"},
							},
						},
					},
				},
			},
			want{
				res:        true,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"geo list array true",
			args{
				ip: "8.8.8.8",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"list": "../../test/testdata/words_lists/banlist_geo_true_array.txt",
						"geolocations": []map[string][]string{
							{
								"country": []string{"some false regexp"},
							},
						},
					},
				},
			},
			want{
				res:        true,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"geo list array true",
			args{
				ip: "8.8.8.8",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"list": "../../test/testdata/words_lists/banlist_geo_true_string.txt",
						"geolocations": []map[string][]string{
							{
								"country": []string{"some false regexp"},
							},
						},
					},
				},
			},
			want{
				res:        true,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"geo geolocations array false",
			args{
				ip: "8.8.8.8",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"list": "",
						"geolocations": []map[string][]string{
							{
								"organisation": []string{"some false org"},
							},
						},
					},
				},
			},
			want{
				res:        false,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"geo geolocations string false",
			args{
				ip: "8.8.8.8",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"list": "",
						"geolocations": []map[string][]string{
							{
								"country": []string{"some false regexp"},
							},
						},
					},
				},
			},
			want{
				res:        false,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"geo list false",
			args{
				ip: "8.8.8.8",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"list": "../../test/testdata/words_lists/banlist_geo_false.txt",
						"geolocations": []map[string][]string{
							{
								"country": []string{"some false regexp"},
							},
						},
					},
				},
			},
			want{
				res:        false,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"geo empty false",
			args{
				ip: "1.1.1.1",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"list": "",
						"geolocations": []map[string][]string{
							{
								"organisation": []string{},
								"country":      []string{},
							},
						},
					},
				},
			},
			want{
				res:        false,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"geo geolocations err bad regexp",
			args{
				ip: "1.1.1.1",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"list": "",
						"geolocations": []map[string][]string{
							{
								"organisation": []string{"(?i"},
							},
						},
					},
				},
			},
			want{
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"geo list err can't open file",
			args{
				ip: "1.1.1.1",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"list": "../../test/testdata/words_lists/banlist_1337.txt",
						"geolocations": []map[string][]string{
							{
								"organisation": []string{"(?i)google"},
							},
						},
					},
				},
			},
			want{
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"geo list err bad regexp",
			args{
				ip: "1.1.1.1",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "geo",
					Params: map[string]any{
						"list": "../../test/testdata/words_lists/broken_regexp.txt",
						"geolocations": []map[string][]string{
							{
								"organisation": []string{"(?i)google"},
							},
						},
					},
				},
			},
			want{
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := database.New("", true)
			require.NoError(t, err, "can't create db")
			rule, err := rules.NewGeolocationRule(
				db,
				rules.RuleSet{},
				tt.args.cfg,
				common.Globals{},
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewGeolocationRule() error mismatch: %s",
				err,
			)

			if !tt.want.createErr {
				const geolocationInfoCount = 2
				for i := 0; i < geolocationInfoCount; i++ {
					e := new(MockEntity)
					e.On("GetIP").Return(netip.MustParseAddr(tt.args.ip))

					err = rule.Prepare(e, log.Logger)
					require.Equalf(
						t,
						tt.want.prepareErr,
						err != nil,
						"Prepare() error mismatch: %s",
						err,
					)

					res, err := rule.Apply(e, log.Logger)
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

func TestBase_ReverseLookupRule(t *testing.T) {
	type args struct {
		ip  string
		cfg common.RuleConfig
	}
	type want struct {
		res        bool
		createErr  bool
		prepareErr bool
		applyErr   bool
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
				cfg: common.RuleConfig{
					Name: "test",
					Type: "reverse_lookup",
					Params: map[string]any{
						"dns":  "1.1.1.1:53",
						"list": "../../test/testdata/words_lists/banlist_regexp.txt",
					},
				},
			},
			want{
				res:        true,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"reverse lookup false",
			args{
				ip: "8.8.8.8",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "reverse_lookup",
					Params: map[string]any{
						"dns":  "1.1.1.1:53",
						"list": "../../test/testdata/words_lists/banlist_regexp.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"reverse lookup err can't open file",
			args{
				ip: "1.1.1.1",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "reverse_lookup",
					Params: map[string]any{
						"dns":  "1.1.1.1:53",
						"list": "../../test/testdata/words_lists/banlist_1337.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"reverse lookup err can't parse regexp",
			args{
				ip: "1.1.1.1",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "reverse_lookup",
					Params: map[string]any{
						"dns":  "1.1.1.1:53",
						"list": "../../test/testdata/words_lists/broken_regexp.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"reverse lookup err can't parse dns",
			args{
				ip: "1.1.1.1",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "reverse_lookup",
					Params: map[string]any{
						"dns":  "1.1.1.1",
						"list": "../../test/testdata/words_lists/banlist_regexp.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  true,
				prepareErr: false,
				applyErr:   false,
			},
		},
		{
			"reverse lookup err dead dns",
			args{
				ip: "1.1.1.1",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "reverse_lookup",
					Params: map[string]any{
						"dns":  "1.1.1.1:553",
						"list": "../../test/testdata/words_lists/banlist_regexp.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  false,
				prepareErr: true,
				applyErr:   true,
			},
		},
		{
			"reverse lookup err unknown ip",
			args{
				ip: "195.168.14.15",
				cfg: common.RuleConfig{
					Name: "test",
					Type: "reverse_lookup",
					Params: map[string]any{
						"dns":  "1.1.1.1:53",
						"list": "../../test/testdata/words_lists/banlist_regexp.txt",
					},
				},
			},
			want{
				res:        false,
				createErr:  false,
				prepareErr: false,
				applyErr:   false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := database.New("", true)
			require.NoError(t, err, "can't create db")
			rule, err := rules.NewReverseLookupRule(
				db,
				rules.RuleSet{},
				tt.args.cfg,
				common.Globals{},
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewReverseLookupRule() error mismatch: %s",
				err,
			)

			if !tt.want.createErr {
				e := new(MockEntity)
				e.On("GetIP").Return(netip.MustParseAddr(tt.args.ip))

				err = rule.Prepare(e, log.Logger)
				require.Equalf(
					t,
					tt.want.prepareErr,
					err != nil,
					"Prepare() error mismatch: %s",
					err,
				)

				res, err := rule.Apply(e, log.Logger)
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
