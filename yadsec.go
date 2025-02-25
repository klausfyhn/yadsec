package yadsec

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
)

func Load(config any) error {
	var (
		val = reflect.ValueOf(config).Elem()
		typ = reflect.TypeOf(config).Elem()
		err error
	)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		envKey := field.Tag.Get("env")
		if envKey == "" {
			continue
		}

		envValue := readEnvvar(envKey)
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

func readEnvvar(key string) string {
	return os.Getenv(key)
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
