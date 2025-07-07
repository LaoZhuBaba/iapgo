package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/LaoZhuBaba/iapgo/v2/internal/constants"
	"github.com/stretchr/testify/assert"
)

func TestGetConfig(t *testing.T) {
	type args struct {
		ctx          context.Context
		yamlFileName string
		cfgSection   string
		logger       *slog.Logger
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr error
	}{
		{
			name: "GetConfig_valid_config",
			args: args{
				ctx:    context.Background(),
				logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})),
			},
			wantErr: nil,
			want: &Config{
				ProjectID:          "project_id",
				Zone:               "zone",
				Instance:           "instance",
				RemotePort:         200,
				LocalPort:          100,
				RemoteNic:          "nic0",
				TerminateAfterExec: false,
			},
		},
		{
			name: "GetConfig_unknown_field",
			args: args{
				ctx:    context.Background(),
				logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})),
			},
			wantErr: constants.ErrFailedToUnmarshalYaml,
			want:    nil,
		},
		{
			name: "GetConfig_empty_config_file",
			args: args{
				ctx:    context.Background(),
				logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})),
			},
			wantErr: constants.ErrFailedToUnmarshalYaml,
			want:    nil,
		},
		{
			name: "GetConfig_no_config_file",
			args: args{
				ctx:    context.Background(),
				logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})),
			},
			wantErr: constants.ErrFailedToReadYaml,
			want:    nil,
		},
		{
			name: "GetConfig_invalid_yaml_file",
			args: args{
				ctx:    context.Background(),
				logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})),
			},
			wantErr: constants.ErrFailedToUnmarshalYaml,
			want:    nil,
		},
		{
			name: "GetConfig_no_section_match",
			args: args{
				ctx:    context.Background(),
				logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})),
			},
			wantErr: constants.ErrConfigSectionNotFound,
			want:    nil,
		},
		{
			name: "GetConfig_ssh_tunnel_to_no_value",
			args: args{
				ctx:    context.Background(),
				logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})),
			},
			wantErr: constants.ErrSshTunnelToNoValue,
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetConfig(tt.args.ctx, fmt.Sprintf("testdata/%s.yaml", tt.name), tt.name, tt.args.logger)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !assert.EqualValues(t, got, tt.want) {
				t.Errorf("GetConfig() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}
