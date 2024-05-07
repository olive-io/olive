/*
   Copyright 2023 The olive Authors

   This program is offered under a commercial and under the AGPL license.
   For AGPL licensing, see below.

   AGPL licensing:
   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package pagation

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
)

func encodeContinueOrDie(apiVersion string, resourceVersion int64, nextKey string) string {
	out, err := json.Marshal(&continueToken{APIVersion: apiVersion, ResourceVersion: resourceVersion, StartKey: nextKey})
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(out)
}

func Test_decodeContinue(t *testing.T) {
	type args struct {
		continueValue string
		keyPrefix     string
	}
	tests := []struct {
		name        string
		args        args
		wantFromKey string
		wantRv      int64
		wantErr     error
	}{
		{
			name:        "valid",
			args:        args{continueValue: encodeContinueOrDie("olive.io/v1", 1, "key"), keyPrefix: "/test/"},
			wantRv:      1,
			wantFromKey: "/test/key",
		},
		{
			name:        "root path",
			args:        args{continueValue: encodeContinueOrDie("olive.io/v1", 1, "/"), keyPrefix: "/test/"},
			wantRv:      1,
			wantFromKey: "/test/",
		},
		{
			name:    "empty version",
			args:    args{continueValue: encodeContinueOrDie("", 1, "key"), keyPrefix: "/test/"},
			wantErr: ErrUnrecognizedEncodedVersion,
		},
		{
			name:    "invalid version",
			args:    args{continueValue: encodeContinueOrDie("v1", 1, "key"), keyPrefix: "/test/"},
			wantErr: ErrUnrecognizedEncodedVersion,
		},
		{
			name:    "invalid RV",
			args:    args{continueValue: encodeContinueOrDie("olive.io/v1", 0, "key"), keyPrefix: "/test/"},
			wantErr: ErrInvalidStartRV,
		},
		{
			name:    "no start Key",
			args:    args{continueValue: encodeContinueOrDie("olive.io/v1", 1, ""), keyPrefix: "/test/"},
			wantErr: ErrEmptyStartKey,
		},
		{
			name:    "path traversal - parent",
			args:    args{continueValue: encodeContinueOrDie("olive.io/v1", 1, "../key"), keyPrefix: "/test/"},
			wantErr: ErrGenericInvalidKey,
		},
		{
			name:    "path traversal - local",
			args:    args{continueValue: encodeContinueOrDie("olive.io/v1", 1, "./key"), keyPrefix: "/test/"},
			wantErr: ErrGenericInvalidKey,
		},
		{
			name:    "path traversal - double parent",
			args:    args{continueValue: encodeContinueOrDie("olive.io/v1", 1, "./../key"), keyPrefix: "/test/"},
			wantErr: ErrGenericInvalidKey,
		},
		{
			name:    "path traversal - after parent",
			args:    args{continueValue: encodeContinueOrDie("olive.io/v1", 1, "key/../.."), keyPrefix: "/test/"},
			wantErr: ErrGenericInvalidKey,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFromKey, gotRv, err := DecodeContinue(tt.args.continueValue, tt.args.keyPrefix)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("decodeContinue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotFromKey != tt.wantFromKey {
				t.Errorf("decodeContinue() gotFromKey = %v, want %v", gotFromKey, tt.wantFromKey)
			}
			if gotRv != tt.wantRv {
				t.Errorf("decodeContinue() gotRv = %v, want %v", gotRv, tt.wantRv)
			}
		})
	}
}
