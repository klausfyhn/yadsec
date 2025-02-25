package yadsec

import (
	"os"
	"reflect"
	"testing"
	"testing/fstest"
)

type SimpleStruct struct {
	Str  string `env:"STR"`
	Int  int    `env:"INT"`
	Bool bool   `env:"BOL"`
}

type TestCase[T comparable] struct {
	name    string
	env     map[string]string
	fs      fstest.MapFS
	want    T
	wantErr bool
}

func Test_LoadSimpleStruct(t *testing.T) {
	tests := []TestCase[SimpleStruct]{
		{
			name: "no env",
			want: SimpleStruct{},
		},
		{
			name: "string env",
			env: map[string]string{
				"STR": "Hello",
			},
			want: SimpleStruct{
				Str: "Hello",
			},
		},
		{
			name: "int env",
			env: map[string]string{
				"INT": "1",
			},
			want: SimpleStruct{
				Int: 1,
			},
		},
		{
			name: "invalid int",
			env: map[string]string{
				"INT": "invalid",
			},
			wantErr: true,
		},
		{
			name: "bool env",
			env: map[string]string{
				"BOL": "true",
			},
			want: SimpleStruct{
				Bool: true,
			},
		},
		{
			name: "invalid bool",
			env: map[string]string{
				"BOL": "invalid",
			},
			wantErr: true,
		},
		{
			name: "all of them together",
			env: map[string]string{
				"STR": "YADSEC",
				"BOL": "1",
				"INT": "512",
			},
			want: SimpleStruct{
				Str:  "YADSEC",
				Int:  512,
				Bool: true,
			},
		},
		{
			name: "VAR, VAR__FILE and VAR__SECRET is mutually exlucive",
			env: map[string]string{
				"STR":         "hello",
				"STR__FILE":   "hello",
				"STR__SECRET": "",
			},
			fs: fstest.MapFS{
				"/hello":            {Data: []byte("hello")},
				"somewhere" + "STR": {Data: []byte("hello")},
			},
			wantErr: true,
		},
		{
			name: "string file should fail if file not there",
			env: map[string]string{
				"STR__FILE": "notthere",
			},
			wantErr: true,
		},
		{
			name: "string from file",
			env: map[string]string{
				"STR__FILE": "hello",
			},
			fs: fstest.MapFS{
				"hello": {Data: []byte("hello")},
			},
			want: SimpleStruct{
				Str: "hello",
			},
		},
		{
			name: "unspecified secret string",
			env: map[string]string{
				"STR__SECRET": "",
			},
			fs: fstest.MapFS{
				"secrets/STR": {Data: []byte("hello")},
			},
			want: SimpleStruct{
				Str: "hello",
			},
		},
		{
			name: "alternate secret name string",
			env: map[string]string{
				"STR__SECRET": "hello",
			},
			fs: fstest.MapFS{
				"secrets/hello": {Data: []byte("hello")},
			},
			want: SimpleStruct{
				Str: "hello",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, performTest(tt))
	}
}

func performTest[T comparable](tc TestCase[T]) func(*testing.T) {
	return func(t *testing.T) {
		for key, value := range tc.env {
			os.Setenv(key, value)
			defer os.Unsetenv(key) // Clean up environment variables after the test
		}
		var got T
		yadsec := Yadsec{
			fs:         tc.fs,
			secretsDir: "secrets/",
		}
		err := yadsec.load(&got)

		if tc.wantErr && err == nil {
			t.Errorf("expected an error but got nil")
		}
		if !tc.wantErr && err != nil {
			t.Errorf("did not expect an error, but got one %v", err)
		}
		if !reflect.DeepEqual(tc.want, got) {
			t.Errorf("expected %v but got %v", tc.want, got)
		}
	}
}

func Test_mutuallyExclusive(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		values []bool
		want   bool
	}{
		{
			name:   "one",
			values: []bool{true},
			want:   true,
		},
		{
			name:   "good",
			values: []bool{true, false, false},
			want:   true,
		},
		{
			name:   "bad",
			values: []bool{true, false, false, false, true},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mutuallyExclusive(tt.values...)
			if tt.want != got {
				t.Errorf("mutuallyExclusive() = %v, want %v", got, tt.want)
			}
		})
	}
}
