package rules_test

import (
	"testing"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/rules"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWrappers_NotRule(t *testing.T) {
	type args struct {
		rule *MockRule
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
				rule: &MockRule{
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
				rule: &MockRule{
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
			if tt.args.rule.called {
				tt.args.rule.
					On("Apply", mock.Anything, mock.Anything).
					Return(tt.args.rule.res, tt.args.rule.err)
			}
			rule := rules.NewNotWrapper(
				tt.args.rule,
				common.RuleConfig{},
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
			tt.args.rule.AssertExpectations(t)
		})
	}
}
