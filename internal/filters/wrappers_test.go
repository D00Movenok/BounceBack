package filters_test

import (
	"testing"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/filters"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWrappers_NotFilter(t *testing.T) {
	type args struct {
		filter *MockFilter
	}
	type want struct {
		res      bool
		applyErr bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"not true",
			args{
				filter: &MockFilter{
					res:    true,
					err:    nil,
					called: true,
				},
			},
			want{
				res:      false,
				applyErr: false,
			},
		},
		{
			"not false",
			args{
				filter: &MockFilter{
					res:    false,
					err:    nil,
					called: true,
				},
			},
			want{
				res:      true,
				applyErr: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.filter.called {
				tt.args.filter.
					On("Apply", mock.Anything, mock.Anything).
					Return(tt.args.filter.res, tt.args.filter.err)
			}
			filter := filters.NewNotWrapper(
				tt.args.filter,
				common.FilterConfig{},
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
			tt.args.filter.AssertExpectations(t)
		})
	}
}
