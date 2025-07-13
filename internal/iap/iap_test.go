package iap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"testing"

	"github.com/LaoZhuBaba/iapgo/v2/internal/config"
	"github.com/LaoZhuBaba/iapgo/v2/internal/constants"
)

type fakeTunnelServer struct {
	serveErr  error
	errorsErr error
	readyErr  interface{}
	t         *testing.T
}

func (f fakeTunnelServer) Serve(ctx context.Context, lis net.Listener) error {
	//f.t.Logf("in fake Serve!!!!!!!!!!")
	return nil
}

func (f fakeTunnelServer) Errors() <-chan error {
	ch := make(chan error, 1)
	if f.errorsErr != nil {
		ch <- f.errorsErr
	}
	return ch
}

func (f fakeTunnelServer) Ready() <-chan struct{} {
	//f.t.Logf("in Ready()")
	ch := make(chan struct{}, 1)
	if f.readyErr != nil {
		ch <- struct{}{}
	}
	return ch
}

func TestIapTunnel_Start(t *testing.T) {
	var logLevel slog.LevelVar
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: &logLevel,
	}))
	slog.SetDefault(logger)
	logLevel.Set(slog.LevelDebug)

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 0))
	if err != nil {
		t.Errorf("failed to listen (listener): %v", err)
	}

	type fields struct {
		config    *config.Config
		listener  net.Listener
		logger    *slog.Logger
		tunnelMgr TunnelServer
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "success",
			fields: fields{
				config: &config.Config{
					ProjectID:          "testProjectID",
					Zone:               "testZone",
					Instance:           "testInstance",
					RemotePort:         2000,
					LocalPort:          3000,
					RemoteNic:          "testRemoteNic",
					Exec:               nil,
					TerminateAfterExec: true,
					SshTunnel:          nil,
				},
				listener: listener,
				logger:   logger,
				tunnelMgr: fakeTunnelServer{
					serveErr:  nil,
					readyErr:  struct{}{},
					errorsErr: nil,
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: nil,
		},
		{
			name: "timeout",
			fields: fields{
				config: &config.Config{
					ProjectID:          "testProjectID",
					Zone:               "testZone",
					Instance:           "testInstance",
					RemotePort:         2000,
					LocalPort:          3000,
					RemoteNic:          "testRemoteNic",
					Exec:               nil,
					TerminateAfterExec: true,
					SshTunnel:          nil,
				},
				listener: listener,
				logger:   logger,
				tunnelMgr: fakeTunnelServer{
					serveErr:  nil,
					readyErr:  nil,
					errorsErr: nil,
					t:         t,
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: constants.ErrTunnelReadyTimeout,
		},
		{
			name: "error",
			fields: fields{
				config: &config.Config{
					ProjectID:          "testProjectID",
					Zone:               "testZone",
					Instance:           "testInstance",
					RemotePort:         2000,
					LocalPort:          3000,
					RemoteNic:          "testRemoteNic",
					Exec:               nil,
					TerminateAfterExec: true,
					SshTunnel:          nil,
				},
				listener: listener,
				logger:   logger,
				tunnelMgr: fakeTunnelServer{
					serveErr:  nil,
					readyErr:  nil,
					errorsErr: constants.ErrFailedToListen,
					t:         t,
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: constants.ErrFailedToListen,
		},
		{
			name: "bad_listener",
			fields: fields{
				config: &config.Config{
					ProjectID:          "testProjectID",
					Zone:               "testZone",
					Instance:           "testInstance",
					RemotePort:         2000,
					LocalPort:          3000,
					RemoteNic:          "testRemoteNic",
					Exec:               nil,
					TerminateAfterExec: true,
					SshTunnel:          nil,
				},
				listener: nil,
				logger:   logger,
				tunnelMgr: fakeTunnelServer{
					serveErr:  nil,
					readyErr:  nil,
					errorsErr: constants.ErrFailedToListen,
					t:         t,
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: constants.ErrNotATcpListener,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t1 *testing.T) {
			t := &IapTunnel{
				config:    tt.fields.config,
				listener:  tt.fields.listener,
				logger:    tt.fields.logger,
				tunnelMgr: tt.fields.tunnelMgr,
			}
			if err := t.Start(tt.args.ctx); !errors.Is(err, tt.wantErr) {
				t1.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
