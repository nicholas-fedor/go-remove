/*
Copyright © 2025 Nicholas Fedor <nick@nickfedor.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/

package logger

import (
	"reflect"
	"testing"

	"go.uber.org/zap"
)

// TestNewZapLogger verifies the NewZapLogger function’s logger creation.
func TestNewZapLogger(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful creation",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewZapLogger()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewZapLogger() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if got == nil {
				t.Errorf("NewZapLogger() returned nil logger")

				return
			}

			// Verify the returned logger is a valid *ZapLogger.
			zapLogger, ok := got.(*ZapLogger)
			if !ok || zapLogger.Logger == nil {
				t.Errorf("NewZapLogger() = %v, expected non-nil *ZapLogger", got)
			}
		})
	}
}

// TestZapLogger_Sync verifies the Sync method’s behavior.
func TestZapLogger_Sync(t *testing.T) {
	tests := []struct {
		name    string
		z       *ZapLogger
		wantErr bool
	}{
		{
			name: "sync with valid logger",
			z: func() *ZapLogger {
				logger, _ := zap.NewProduction()

				return &ZapLogger{logger}
			}(),
			wantErr: false,
		},
		{
			name:    "sync with nil logger",
			z:       &ZapLogger{nil},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.z.Sync()
			if (err != nil) != tt.wantErr {
				t.Errorf("ZapLogger.Sync() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestZapLogger_Sugar verifies the Sugar method’s sugared logger output.
func TestZapLogger_Sugar(t *testing.T) {
	tests := []struct {
		name string
		z    *ZapLogger
	}{
		{
			name: "sugar with valid logger",
			z: func() *ZapLogger {
				logger, _ := zap.NewProduction()

				return &ZapLogger{logger}
			}(),
		},
		{
			name: "sugar with nil logger",
			z:    &ZapLogger{nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.z.Sugar()
			if tt.z.Logger == nil {
				if got != nil {
					t.Errorf("ZapLogger.Sugar() = %v, want nil for nil logger", got)
				}
			} else {
				if got == nil {
					t.Errorf("ZapLogger.Sugar() = nil, want non-nil *zap.SugaredLogger")
				} else if reflect.TypeOf(got) != reflect.TypeOf(zap.NewNop().Sugar()) {
					t.Errorf("ZapLogger.Sugar() = %v, want *zap.SugaredLogger", got)
				}
			}
		})
	}
}
