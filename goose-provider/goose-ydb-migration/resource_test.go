package goose_ydb_migration

import "testing"

func Test_makeDbString(t *testing.T) {
	type args struct {
		endpoint   string
		database   string
		token      string
		tlsEnabled bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test 1",
			args: args{
				endpoint:   "endpoint",
				database:   "/database",
				token:      "token",
				tlsEnabled: true,
			},
			want: "grpcs://endpoint/database?go_fake_tx=scripting&go_query_bind=declare,numeric&go_query_mode=scripting&token=token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := makeDbString(tt.args.endpoint, tt.args.database, tt.args.token, &tt.args.tlsEnabled); got != tt.want {
				t.Errorf("makeDbString() = %v, want %v", got, tt.want)
			}
		})
	}
}
