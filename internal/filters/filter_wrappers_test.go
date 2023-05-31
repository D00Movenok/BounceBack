package filters

import (
	"testing"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWrappers_NotFilter(t *testing.T) {
	type fields struct {
		filter *MockFilter
	}
	type want struct {
		res      bool
		applyErr bool
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			"not true",
			fields{
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
			fields{
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
			if tt.fields.filter.called {
				tt.fields.filter.On("Apply", mock.Anything, mock.Anything).Return(tt.fields.filter.res, tt.fields.filter.err)
			}
			filter := NewNotWrapper(tt.fields.filter, common.FilterConfig{})
			res, err := filter.Apply(nil, log.Logger)
			require.Equal(t, tt.want.applyErr, err != nil, "Apply() error mismatch")
			require.Equal(t, tt.want.res, res, "Apply() result mismatch")
			tt.fields.filter.AssertExpectations(t)
		})
	}
}
