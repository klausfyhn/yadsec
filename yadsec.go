package yadsec

import (
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"strconv"
	"strings"
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

	for i := range typ.NumField() {
		field := typ.Field(i)
		rawKey := field.Tag.Get("env")
		if rawKey == "" {
			continue
		}
		keys := strings.Split(rawKey, ",")

		envKey := keys[0]

		envValue, err := y.readEnvvar(envKey)
		if err != nil {
			return fmt.Errorf("failed to read variable %s: %v", envKey, err)
		}
		if envValue == "" {
			if contains("required", keys) {
				return fmt.Errorf("%s is required", envKey)
			} else {
				continue
			}
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
		s := os.Getenv(secret)
		var secretName string
		if s == "" {
			secretName = key
		} else {
			secretName = s
		}

		os.Setenv(file, y.secretsDir+secretName)
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
	var fd fs.FS
	if y.fs != nil {
		fd = y.fs
	} else {
		fd = os.DirFS("/")
	}

	path = strings.TrimPrefix(path, "/")

	if !fs.ValidPath(path) {
		return "", fmt.Errorf("invalid path %s", path)
	}

	f, err := fd.Open(path)
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

	str := string(b)
	str = strings.TrimSpace(str)

	return str, nil
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

func contains[T comparable](elem T, slice []T) bool {
	for _, v := range slice {
		if elem == v {
			return true
		}
	}
	return false
}
