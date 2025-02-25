package yadsec

import (
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"strconv"
)

const (
	fileVarSuffix   = "__FILE"
	secretVarSuffix = "__SECRET"
)

type Yadsec struct {
	fs         fs.FS
	secretsDir string
}

func Load(config any) error {
	y := new(Yadsec)
	y.secretsDir = "/run/secrets/"
	return y.load(config)
}

func (y Yadsec) load(config any) error {
	var (
		val = reflect.ValueOf(config).Elem()
		typ = reflect.TypeOf(config).Elem()
	)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		envKey := field.Tag.Get("env")
		if envKey == "" {
			continue
		}

		envValue, err := y.readEnvvar(envKey)
		if err != nil {
			return fmt.Errorf("failed to read variable %s: %v", envKey, err)
		}
		if envValue == "" {
			continue
		}

		fieldVal := val.Field(i)

		err = parseEnvvar(fieldVal, envValue, envKey)
		if err != nil {
			return fmt.Errorf("failed to parse variable: %v", err)
		}
	}
	return nil
}

func (y Yadsec) readEnvvar(key string) (string, error) {
	var (
		file   = fileEnvvar(key)
		secret = secretEnvvar(key)
	)

	if !mutuallyExclusive(isEnvSet(key), isEnvSet(file), isEnvSet(secret)) {
		return "", fmt.Errorf("%s, %s and %s are mutually exclusive", key, file, secret)
	}

	if isEnvSet(key) {
		return os.Getenv(key), nil
	}

	if isEnvSet(secret) {
		defer os.Unsetenv(secret)
		os.Setenv(file, y.secretsDir+key)
	}

	if isEnvSet(file) {
		defer os.Unsetenv(file)
		path := os.Getenv(file)
		value, err := y.readFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %v", file, err)
		}
		if value == "" {
			return "", fmt.Errorf("content is empty %s", file)
		}
		return value, err
	}

	return "", nil
}

func fileEnvvar(key string) string {
	return key + fileVarSuffix
}

func (y Yadsec) readFile(path string) (string, error) {
	f, err := y.fs.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open %s with error: %v", path, err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat %s with error: %v", path, err)
	}

	b := make([]byte, stat.Size())
	_, err = f.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to read %s with error: %v", path, err)
	}
	return string(b), nil
}

func secretEnvvar(key string) string {
	return key + secretVarSuffix
}

func parseEnvvar(fieldVal reflect.Value, envValue string, envTag string) error {
	switch fieldVal.Kind() {
	case reflect.String:
		fieldVal.SetString(envValue)
		return nil
	case reflect.Int:
		intVal, err := strconv.Atoi(envValue)
		if err != nil {
			return fmt.Errorf("invalid integer value for %s: %v", envTag, err)
		}
		fieldVal.SetInt(int64(intVal))
		return nil
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(envValue)
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %v", envTag, err)
		}
		fieldVal.SetBool(boolVal)
		return nil
	default:
		return fmt.Errorf("unsupported field type for %s", envTag)
	}
}

func isEnvSet(key string) bool {
	_, set := os.LookupEnv(key)
	return set
}

func mutuallyExclusive(values ...bool) bool {
	count := 0
	for _, v := range values {
		if v {
			count++
		}
		if count > 1 {
			return false
		}
	}
	return count <= 1
}
