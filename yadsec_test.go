package yadsec

import (
	"os"
	"reflect"
	"testing"
)

type testCase[T comparable] struct {
	name    string
	env     map[string]string
	want    T
	wantErr bool
}

func TestYadsscSimpleStruct(t *testing.T) {
	type Struct struct {
		Str  string `env:"STR"`
		Int  int    `env:"INT"`
		Bool bool   `env:"BOL"`
	}
	tests := []testCase[Struct]{
		{
			name: "no env",
			want: Struct{},
		},
		{
			name: "string env",
			env: map[string]string{
				"STR": "Hello",
			},
			want: Struct{
				Str: "Hello",
			},
		},
		{
			name: "int env",
			env: map[string]string{
				"INT": "1",
			},
			want: Struct{
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
			want: Struct{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, performTest(tt))
	}
}

func performTest[T comparable](tc testCase[T]) func(*testing.T) {
	return func(t *testing.T) {
		for key, value := range tc.env {
			os.Setenv(key, value)
			defer os.Unsetenv(key) // Clean up environment variables after the test
		}
		var got T
		err := Load(&got)

		if tc.wantErr && err == nil {
			t.Errorf("expected an error but got nil")
		}
		if !reflect.DeepEqual(tc.want, got) {
			t.Errorf("expected %v but got %v", tc.want, got)
		}
	}
}
