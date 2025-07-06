package rules_test

import (
	"errors"
	"testing"

	"github.com/rs/zerolog/log"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/rules"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockRule struct {
	mock.Mock

	res    bool
	err    error
	called bool
}

func (m *MockRule) Prepare(
	e wrapper.Entity,
	logger zerolog.Logger,
) error {
	params := m.Called(e, logger)
	return params.Error(0) //nolint: wrapcheck // mock
}

func (m *MockRule) Apply(
	e wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	params := m.Called(e, logger)
	return params.Bool(0), params.Error(1)
}

func (m *MockRule) String() string {
	return "mock"
}

func TestComposites_CompositeAndRule(t *testing.T) {
	type args struct {
		rs  map[string]*MockRule
		cfg common.RuleConfig
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
			"and true",
			args{
				rs: map[string]*MockRule{
					"r1": {
						res:    true,
						err:    nil,
						called: true,
					},
					"r2": {
						res:    true,
						err:    nil,
						called: true,
					},
				},
				cfg: common.RuleConfig{
					Name: "test",
					Type: "and",
					Params: map[string]any{
						"rules": []string{"r1", "r2"},
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
			"and false",
			args{
				rs: map[string]*MockRule{
					"r1": {
						res:    true,
						err:    nil,
						called: true,
					},
					"r2": {
						res:    false,
						err:    nil,
						called: true,
					},
					"r3": {
						res:    false,
						err:    nil,
						called: false,
					},
				},
				cfg: common.RuleConfig{
					Name: "test",
					Type: "and",
					Params: map[string]any{
						"rules": []string{"r1", "r2", "r3"},
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
			"and rule error",
			args{
				rs: map[string]*MockRule{
					"r1": {
						res:    true,
						err:    errors.New("some error"),
						called: true,
					},
					"r2": {
						res:    true,
						err:    nil,
						called: false,
					},
				},
				cfg: common.RuleConfig{
					Name: "test",
					Type: "and",
					Params: map[string]any{
						"rules": []string{"r1", "r2"},
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
			"and unk rule error",
			args{
				rs: map[string]*MockRule{},
				cfg: common.RuleConfig{
					Name: "test",
					Type: "and",
					Params: map[string]any{
						"rules": []string{"r1", "r2"},
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
			"and not enough Params",
			args{
				rs: map[string]*MockRule{
					"r1": {
						res:    true,
						err:    nil,
						called: false,
					},
				},
				cfg: common.RuleConfig{
					Name: "test",
					Type: "and",
					Params: map[string]any{
						"rules": []string{"r1"},
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
			rs := rules.RuleSet{
				Rules: map[string]rules.Rule{},
			}
			for k, v := range tt.args.rs {
				if !tt.want.createErr {
					v.
						On("Prepare", mock.Anything, mock.Anything).
						Return(nil)
				}

				if v.called {
					v.
						On("Apply", mock.Anything, mock.Anything).
						Return(v.res, v.err)
				}
				rs.Rules[k] = v
			}

			rule, err := rules.NewCompositeAndRule(
				nil,
				rs,
				tt.args.cfg,
				common.Globals{},
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewCompositeAndRule() error mismatch: %s",
				err,
			)
			if !tt.want.createErr {
				err := rule.Prepare(nil, log.Logger)
				require.NoError(
					t,
					err,
					"Prepare() error mismatch: %s",
					err,
				)

				res, err := rule.Apply(nil, log.Logger)
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

				for _, v := range tt.args.rs {
					if v.called {
						v.AssertExpectations(t)
					}
				}
			}
		})
	}
}

func TestComposites_CompositeOrRule(t *testing.T) {
	type args struct {
		rs  map[string]*MockRule
		cfg common.RuleConfig
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
			"or true",
			args{
				rs: map[string]*MockRule{
					"r1": {
						res:    false,
						err:    nil,
						called: true,
					},
					"r2": {
						res:    true,
						err:    nil,
						called: true,
					},
					"r3": {
						res:    false,
						err:    nil,
						called: false,
					},
				},
				cfg: common.RuleConfig{
					Name: "test",
					Type: "or",
					Params: map[string]any{
						"rules": []string{"r1", "r2", "r3"},
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
			"or false",
			args{
				rs: map[string]*MockRule{
					"r1": {
						res:    false,
						err:    nil,
						called: true,
					},
					"r2": {
						res:    false,
						err:    nil,
						called: true,
					},
				},
				cfg: common.RuleConfig{
					Name: "test",
					Type: "or",
					Params: map[string]any{
						"rules": []string{"r1", "r2"},
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
			"or rule error",
			args{
				rs: map[string]*MockRule{
					"r1": {
						res:    false,
						err:    errors.New("some error"),
						called: true,
					},
					"r2": {
						res:    false,
						err:    nil,
						called: false,
					},
				},
				cfg: common.RuleConfig{
					Name: "test",
					Type: "or",
					Params: map[string]any{
						"rules": []string{"r1", "r2"},
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
			"or unk rule error",
			args{
				rs: map[string]*MockRule{},
				cfg: common.RuleConfig{
					Name: "test",
					Type: "or",
					Params: map[string]any{
						"rules": []string{"r1", "r2"},
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
			"or not enough Params",
			args{
				rs: map[string]*MockRule{
					"r1": {
						res:    true,
						err:    nil,
						called: false,
					},
				},
				cfg: common.RuleConfig{
					Name: "test",
					Type: "or",
					Params: map[string]any{
						"rules": []string{"r1"},
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
			rs := rules.RuleSet{
				Rules: map[string]rules.Rule{},
			}
			for k, v := range tt.args.rs {
				if !tt.want.createErr {
					v.
						On("Prepare", mock.Anything, mock.Anything).
						Return(nil)
				}

				if v.called {
					v.
						On("Apply", mock.Anything, mock.Anything).
						Return(v.res, v.err)
				}
				rs.Rules[k] = v
			}

			rule, err := rules.NewCompositeOrRule(nil,
				rs,
				tt.args.cfg,
				common.Globals{},
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewCompositeOrRule() error mismatch: %s",
				err,
			)
			if !tt.want.createErr {
				err := rule.Prepare(nil, log.Logger)
				require.NoError(
					t,
					err,
					"Prepare() error mismatch: %s",
					err,
				)

				res, err := rule.Apply(nil, log.Logger)
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

				for _, v := range tt.args.rs {
					if v.called {
						v.AssertExpectations(t)
					}
				}
			}
		})
	}
}

func TestComposites_CompositeNotRule(t *testing.T) {
	type args struct {
		rs  map[string]*MockRule
		cfg common.RuleConfig
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
			"not true",
			args{
				rs: map[string]*MockRule{
					"r1": {
						res:    false,
						err:    nil,
						called: true,
					},
				},
				cfg: common.RuleConfig{
					Name: "test",
					Type: "not",
					Params: map[string]any{
						"rule": "r1",
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
			"not false",
			args{
				rs: map[string]*MockRule{
					"r1": {
						res:    true,
						err:    nil,
						called: true,
					},
				},
				cfg: common.RuleConfig{
					Name: "test",
					Type: "not",
					Params: map[string]any{
						"rule": "r1",
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
			"not rule error",
			args{
				rs: map[string]*MockRule{
					"r1": {
						res:    false,
						err:    errors.New("some error"),
						called: true,
					},
				},
				cfg: common.RuleConfig{
					Name: "test",
					Type: "not",
					Params: map[string]any{
						"rule": "r1",
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
			"not unk rule error",
			args{
				rs: map[string]*MockRule{},
				cfg: common.RuleConfig{
					Name: "test",
					Type: "not",
					Params: map[string]any{
						"rule": "r1",
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
			"not not enough Params or too many arguments",
			args{
				rs: map[string]*MockRule{
					"r1": {
						res:    true,
						err:    nil,
						called: false,
					},
					"r2": {
						res:    true,
						err:    nil,
						called: false,
					},
				},
				cfg: common.RuleConfig{
					Name: "test",
					Type: "not",
					Params: map[string]any{
						"rule": []string{"r1", "r2"},
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
			rs := rules.RuleSet{
				Rules: map[string]rules.Rule{},
			}
			for k, v := range tt.args.rs {
				if !tt.want.createErr {
					v.
						On("Prepare", mock.Anything, mock.Anything).
						Return(nil)
				}

				if v.called {
					v.
						On("Apply", mock.Anything, mock.Anything).
						Return(v.res, v.err)
				}
				rs.Rules[k] = v
			}

			rule, err := rules.NewCompositeNotRule(nil,
				rs,
				tt.args.cfg,
				common.Globals{},
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewCompositeNotRule() error mismatch: %s",
				err,
			)
			if !tt.want.createErr {
				err := rule.Prepare(nil, log.Logger)
				require.NoError(
					t,
					err,
					"Prepare() error mismatch: %s",
					err,
				)

				res, err := rule.Apply(nil, log.Logger)
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

				for _, v := range tt.args.rs {
					if v.called {
						v.AssertExpectations(t)
					}
				}
			}
		})
	}
}
