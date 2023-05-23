package filters

import (
	"errors"
	"testing"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockFilter struct {
	mock.Mock
	res    bool
	err    error
	called bool
}

func (m MockFilter) Apply(e wrapper.Entity) (bool, error) {
	params := m.Called(e)
	return params.Bool(0), params.Error(1)
}

func (m MockFilter) String() string {
	return "mock"
}

func TestComposites_CompositeAndFilter(t *testing.T) {
	type fields struct {
		fs  map[string]*MockFilter
		cfg common.FilterConfig
	}
	type want struct {
		res       bool
		createErr bool
		applyErr  bool
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			"and true",
			fields{
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
			fields{
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
			fields{
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
			fields{
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
			fields{
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
			fs := FilterSet{
				Filters: map[string]Filter{},
			}
			for k, v := range tt.fields.fs {
				if v.called {
					v.On("Apply", mock.Anything, mock.Anything).Return(v.res, v.err)
				}
				fs.Filters[k] = v
			}

			filter, err := NewCompositeAndFilter(fs, tt.fields.cfg)
			require.Equal(t, tt.want.createErr, err != nil, "NewCompositeAndFilter() error mismatch")
			if !tt.want.createErr {
				res, err := filter.Apply(nil)
				require.Equal(t, tt.want.applyErr, err != nil, "Apply() error mismatch")
				require.Equal(t, tt.want.res, res, "Apply() result mismatch")

				for _, v := range tt.fields.fs {
					if v.called {
						v.AssertExpectations(t)
					}
				}
			}
		})
	}
}

func TestComposites_CompositeOrFilter(t *testing.T) {
	type fields struct {
		fs  map[string]*MockFilter
		cfg common.FilterConfig
	}
	type want struct {
		res       bool
		createErr bool
		applyErr  bool
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			"or true",
			fields{
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
			fields{
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
			fields{
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
			fields{
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
			fields{
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
			fs := FilterSet{
				Filters: map[string]Filter{},
			}
			for k, v := range tt.fields.fs {
				if v.called {
					v.On("Apply", mock.Anything, mock.Anything).Return(v.res, v.err)
				}
				fs.Filters[k] = v
			}

			filter, err := NewCompositeOrFilter(fs, tt.fields.cfg)
			require.Equal(t, tt.want.createErr, err != nil, "NewCompositeOrFilter() error mismatch")
			if !tt.want.createErr {
				res, err := filter.Apply(nil)
				require.Equal(t, tt.want.applyErr, err != nil, "Apply() error mismatch")
				require.Equal(t, tt.want.res, res, "Apply() result mismatch")

				for _, v := range tt.fields.fs {
					if v.called {
						v.AssertExpectations(t)
					}
				}
			}
		})
	}
}

func TestComposites_CompositeNotFilter(t *testing.T) {
	type fields struct {
		fs  map[string]*MockFilter
		cfg common.FilterConfig
	}
	type want struct {
		res       bool
		createErr bool
		applyErr  bool
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			"not true",
			fields{
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
			fields{
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
			fields{
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
			fields{
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
			fields{
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
			fs := FilterSet{
				Filters: map[string]Filter{},
			}
			for k, v := range tt.fields.fs {
				if v.called {
					v.On("Apply", mock.Anything, mock.Anything).Return(v.res, v.err)
				}
				fs.Filters[k] = v
			}

			filter, err := NewCompositeNotFilter(fs, tt.fields.cfg)
			require.Equal(t, tt.want.createErr, err != nil, "NewCompositeNotFilter() error mismatch")
			if !tt.want.createErr {
				res, err := filter.Apply(nil)
				require.Equal(t, tt.want.applyErr, err != nil, "Apply() error mismatch")
				require.Equal(t, tt.want.res, res, "Apply() result mismatch")

				for _, v := range tt.fields.fs {
					if v.called {
						v.AssertExpectations(t)
					}
				}
			}
		})
	}
}