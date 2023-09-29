package filters_test

import (
	"errors"
	"testing"

	"github.com/rs/zerolog/log"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/filters"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockFilter struct {
	mock.Mock
	res    bool
	err    error
	called bool
}

func (m *MockFilter) Prepare(
	e wrapper.Entity,
	logger zerolog.Logger,
) error {
	params := m.Called(e, logger)
	return params.Error(0) //nolint: wrapcheck // mock
}

func (m *MockFilter) Apply(
	e wrapper.Entity,
	logger zerolog.Logger,
) (bool, error) {
	params := m.Called(e, logger)
	return params.Bool(0), params.Error(1)
}

func (m *MockFilter) String() string {
	return "mock"
}

func TestComposites_CompositeAndFilter(t *testing.T) {
	type args struct {
		fs  map[string]*MockFilter
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
			"and true",
			args{
				fs: map[string]*MockFilter{
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
				cfg: common.FilterConfig{
					Name: "test",
					Type: "and",
					Params: map[string]any{
						"filters": []string{"r1", "r2"},
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
				fs: map[string]*MockFilter{
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
				cfg: common.FilterConfig{
					Name: "test",
					Type: "and",
					Params: map[string]any{
						"filters": []string{"r1", "r2", "r3"},
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
			"and filter error",
			args{
				fs: map[string]*MockFilter{
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
				cfg: common.FilterConfig{
					Name: "test",
					Type: "and",
					Params: map[string]any{
						"filters": []string{"r1", "r2"},
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
			"and unk filter error",
			args{
				fs: map[string]*MockFilter{},
				cfg: common.FilterConfig{
					Name: "test",
					Type: "and",
					Params: map[string]any{
						"filters": []string{"r1", "r2"},
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
				fs: map[string]*MockFilter{
					"r1": {
						res:    true,
						err:    nil,
						called: false,
					},
				},
				cfg: common.FilterConfig{
					Name: "test",
					Type: "and",
					Params: map[string]any{
						"filters": []string{"r1"},
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
			fs := filters.FilterSet{
				Filters: map[string]filters.Filter{},
			}
			for k, v := range tt.args.fs {
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
				fs.Filters[k] = v
			}

			filter, err := filters.NewCompositeAndFilter(
				nil,
				fs,
				tt.args.cfg,
				common.Globals{},
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewCompositeAndFilter() error mismatch: %s",
				err,
			)
			if !tt.want.createErr {
				err := filter.Prepare(nil, log.Logger)
				require.NoError(
					t,
					err,
					"Prepare() error mismatch: %s",
					err,
				)

				res, err := filter.Apply(nil, log.Logger)
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

				for _, v := range tt.args.fs {
					if v.called {
						v.AssertExpectations(t)
					}
				}
			}
		})
	}
}

func TestComposites_CompositeOrFilter(t *testing.T) {
	type args struct {
		fs  map[string]*MockFilter
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
			"or true",
			args{
				fs: map[string]*MockFilter{
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
				cfg: common.FilterConfig{
					Name: "test",
					Type: "or",
					Params: map[string]any{
						"filters": []string{"r1", "r2", "r3"},
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
				fs: map[string]*MockFilter{
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
				cfg: common.FilterConfig{
					Name: "test",
					Type: "or",
					Params: map[string]any{
						"filters": []string{"r1", "r2"},
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
			"or filter error",
			args{
				fs: map[string]*MockFilter{
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
				cfg: common.FilterConfig{
					Name: "test",
					Type: "or",
					Params: map[string]any{
						"filters": []string{"r1", "r2"},
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
			"or unk filter error",
			args{
				fs: map[string]*MockFilter{},
				cfg: common.FilterConfig{
					Name: "test",
					Type: "or",
					Params: map[string]any{
						"filters": []string{"r1", "r2"},
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
				fs: map[string]*MockFilter{
					"r1": {
						res:    true,
						err:    nil,
						called: false,
					},
				},
				cfg: common.FilterConfig{
					Name: "test",
					Type: "or",
					Params: map[string]any{
						"filters": []string{"r1"},
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
			fs := filters.FilterSet{
				Filters: map[string]filters.Filter{},
			}
			for k, v := range tt.args.fs {
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
				fs.Filters[k] = v
			}

			filter, err := filters.NewCompositeOrFilter(nil,
				fs,
				tt.args.cfg,
				common.Globals{},
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewCompositeOrFilter() error mismatch: %s",
				err,
			)
			if !tt.want.createErr {
				err := filter.Prepare(nil, log.Logger)
				require.NoError(
					t,
					err,
					"Prepare() error mismatch: %s",
					err,
				)

				res, err := filter.Apply(nil, log.Logger)
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

				for _, v := range tt.args.fs {
					if v.called {
						v.AssertExpectations(t)
					}
				}
			}
		})
	}
}

func TestComposites_CompositeNotFilter(t *testing.T) {
	type args struct {
		fs  map[string]*MockFilter
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
			"not true",
			args{
				fs: map[string]*MockFilter{
					"r1": {
						res:    false,
						err:    nil,
						called: true,
					},
				},
				cfg: common.FilterConfig{
					Name: "test",
					Type: "not",
					Params: map[string]any{
						"filter": "r1",
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
				fs: map[string]*MockFilter{
					"r1": {
						res:    true,
						err:    nil,
						called: true,
					},
				},
				cfg: common.FilterConfig{
					Name: "test",
					Type: "not",
					Params: map[string]any{
						"filter": "r1",
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
			"not filter error",
			args{
				fs: map[string]*MockFilter{
					"r1": {
						res:    false,
						err:    errors.New("some error"),
						called: true,
					},
				},
				cfg: common.FilterConfig{
					Name: "test",
					Type: "not",
					Params: map[string]any{
						"filter": "r1",
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
			"not unk filter error",
			args{
				fs: map[string]*MockFilter{},
				cfg: common.FilterConfig{
					Name: "test",
					Type: "not",
					Params: map[string]any{
						"filter": "r1",
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
				fs: map[string]*MockFilter{
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
				cfg: common.FilterConfig{
					Name: "test",
					Type: "not",
					Params: map[string]any{
						"filter": []string{"r1", "r2"},
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
			fs := filters.FilterSet{
				Filters: map[string]filters.Filter{},
			}
			for k, v := range tt.args.fs {
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
				fs.Filters[k] = v
			}

			filter, err := filters.NewCompositeNotFilter(nil,
				fs,
				tt.args.cfg,
				common.Globals{},
			)
			require.Equalf(
				t,
				tt.want.createErr,
				err != nil,
				"NewCompositeNotFilter() error mismatch: %s",
				err,
			)
			if !tt.want.createErr {
				err := filter.Prepare(nil, log.Logger)
				require.NoError(
					t,
					err,
					"Prepare() error mismatch: %s",
					err,
				)

				res, err := filter.Apply(nil, log.Logger)
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

				for _, v := range tt.args.fs {
					if v.called {
						v.AssertExpectations(t)
					}
				}
			}
		})
	}
}
