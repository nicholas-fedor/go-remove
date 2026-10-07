/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package flags

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateBinary verifies blank names are rejected and others pass
// through unchanged.
func TestValidateBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr error
		name    string
		input   string
	}{
		{name: "plain name", input: "vhs", wantErr: nil},
		{name: "surrounding space is kept", input: " vhs ", wantErr: nil},
		{name: "empty", input: "", wantErr: ErrEmptyBinaryName},
		{name: "whitespace", input: " \t\n", wantErr: ErrEmptyBinaryName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ValidateBinary(tt.input)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, got)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.input, got)
		})
	}
}
