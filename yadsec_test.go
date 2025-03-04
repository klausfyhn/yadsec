package yadsec

import (
	"os"
	"reflect"
	"testing"
	"testing/fstest"
)

type TestCase[T comparable] struct {
	name    string
	env     map[string]string
	fs      fstest.MapFS
	want    T
	wantErr bool
}

func Test_LoadSimpleStruct(t *testing.T) {
	type SimpleStruct struct {
		Str  string `env:"STR"`
		Int  int    `env:"INT"`
		Bool bool   `env:"BOL"`
	}
	tests := []TestCase[SimpleStruct]{
		{
			name: "no env variables set",
			want: SimpleStruct{},
		},
		{
			name: "string env variable",
			env: map[string]string{
				"STR": "Hello",
			},
			want: SimpleStruct{
				Str: "Hello",
			},
		},
		{
			name: "int env variable",
			env: map[string]string{
				"INT": "1",
			},
			want: SimpleStruct{
				Int: 1,
			},
		},
		{
			name: "invalid int env variable",
			env: map[string]string{
				"INT": "invalid",
			},
			wantErr: true,
		},
		{
			name: "bool env variable",
			env: map[string]string{
				"BOL": "true",
			},
			want: SimpleStruct{
				Bool: true,
			},
		},
		{
			name: "invalid bool env variable",
			env: map[string]string{
				"BOL": "invalid",
			},
			wantErr: true,
		},
		{
			name: "all env variables together",
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
			name: "mutually exclusive env variables",
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
		{
			name: "empty string env variable",
			env: map[string]string{
				"STR": "",
			},
			want: SimpleStruct{},
		},
		{
			name: "large int env variable",
			env: map[string]string{
				"INT": "999999999",
			},
			want: SimpleStruct{
				Int: 999999999,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, performTest(tt))
	}
}

func TestRequiredFields(t *testing.T) {
	type xyz struct {
		Required string `env:"REQUIRED,required"`
	}
	tests := []TestCase[xyz]{
		{
			name:    "required field missing",
			wantErr: true,
		},
		{
			name: "required field present",
			env: map[string]string{
				"REQUIRED": "present",
			},
			want: xyz{
				Required: "present",
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
		name   string
		values []bool
		want   bool
	}{
		{
			name:   "one true value",
			values: []bool{true},
			want:   true,
		},
		{
			name:   "one true value among falses",
			values: []bool{true, false, false},
			want:   true,
		},
		{
			name:   "multiple true values",
			values: []bool{true, false, false, false, true},
			want:   false,
		},
		{
			name:   "all false values",
			values: []bool{false, false, false},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mutuallyExclusive(tt.values...)
			if tt.want != got {
				t.Errorf("expected %v but got %v", tt.want, got)
			}
		})
	}
}
