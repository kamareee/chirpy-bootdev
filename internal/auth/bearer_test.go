package auth

import (
	"net/http"
	"testing"
)

func TestGetBearerToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		authorization string
		wantToken     string
		wantErr       bool
	}{
		{
			name:          "valid bearer token",
			authorization: "Bearer abc123token",
			wantToken:     "abc123token",
			wantErr:       false,
		},
		{
			name:          "valid bearer token with special characters",
			authorization: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			wantToken:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			wantErr:       false,
		},
		{
			name:          "missing Bearer prefix",
			authorization: "abc123token",
			wantToken:     "",
			wantErr:       true,
		},
		{
			name:          "wrong prefix",
			authorization: "Basic abc123token",
			wantToken:     "",
			wantErr:       true,
		},
		{
			name:          "empty authorization header",
			authorization: "",
			wantToken:     "",
			wantErr:       true,
		},
		{
			name:          "only Bearer without token",
			authorization: "Bearer",
			wantToken:     "",
			wantErr:       true,
		},
		{
			name:          "Bearer with empty token",
			authorization: "Bearer ",
			wantToken:     "",
			wantErr:       false,
		},
		{
			name:          "lowercase bearer",
			authorization: "bearer abc123token",
			wantToken:     "",
			wantErr:       true,
		},
		// {
		// 	name:          "extra spaces",
		// 	authorization: "Bearer  abc123token",
		// 	wantToken:     "",
		// 	wantErr:       true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			headers := http.Header{}
			if tt.authorization != "" {
				headers.Set("Authorization", tt.authorization)
			}

			token, err := GetBearerToken(headers)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetBearerToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if token != tt.wantToken {
				t.Errorf("GetBearerToken() token = %v, want %v", token, tt.wantToken)
			}
		})
	}
}
