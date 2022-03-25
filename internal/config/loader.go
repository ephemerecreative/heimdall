package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"unicode"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
)

var defaultDecodeHooks = []mapstructure.DecodeHookFunc{
	mapstructure.StringToTimeDurationHookFunc(),
	mapstructure.StringToSliceHookFunc(","),
	logLevelDecode,
	logFormatDecode,
}

// LoadConfig loads configuration into the given struct. This will take into account the following
// sources:
//
// - the given struct
// - the .env file given as "optionalConfigFile" if this argument is not nil/empty
// - a .env file in root if no file was given. This is optional
// - Overrides from Environment
//
// Per convention all fields in the struct must have lowercase "koanf" tags.
// Environment Variables will be automatically converted to lowercase. The underscore "_" serves as
// hierarchy-separator ("FOO_BAR" matches the field "bar" in the nested strut "foo".)
//
// Type Conversions for standard types are present as well as for slices and durations
func LoadConfig(config interface{}, configFile string) error {
	return LoadConfigWithDecoder(config, configFile, nil)
}

// LoadConfigWithDecoder works like "LoadConfig", but allows to use an additional DecodeHook to allow
// conversion from string values to custom types
func LoadConfigWithDecoder(config interface{}, optionalConfigFile string, additionalDecodeHook mapstructure.DecodeHookFunc) error {
	configFile := optionalConfigFile
	if len(configFile) == 0 {
		configFile = "configs/config.yaml"
	}

	err, k := koanfFromStruct(config)
	if err != nil {
		return err
	}

	loadAndMergeConfig := func(loadConfig func() (*koanf.Koanf, error)) error {
		c, err := loadConfig()
		if err != nil {
			return err
		}
		return k.Merge(c)
	}

	if _, err := os.Stat(configFile); err == nil {
		if err := loadAndMergeConfig(func() (*koanf.Koanf, error) { return koanfFromYaml(configFile) }); err != nil {
			return err
		}
	}
	if err := loadAndMergeConfig(koanfFromEnv); err != nil {
		return err
	}

	var hooks = defaultDecodeHooks
	if additionalDecodeHook != nil {
		hooks = append(hooks, additionalDecodeHook)
	}

	return k.UnmarshalWithConf("", config, koanf.UnmarshalConf{
		Tag: "koanf",
		DecoderConfig: &mapstructure.DecoderConfig{
			DecodeHook:       mapstructure.ComposeDecodeHookFunc(hooks...),
			Metadata:         nil,
			Result:           config,
			WeaklyTypedInput: true,
		},
	})
}

func koanfFromYaml(configFile string) (*koanf.Koanf, error) {
	var k = koanf.New(".")
	err := k.Load(file.Provider(configFile), yaml.Parser())
	if err != nil {
		return nil, fmt.Errorf("failed to read yaml config from %s: %w", configFile, err)
	}
	return k, nil
}

func isLower(s string) bool {
	for _, r := range s {
		if !unicode.IsLower(r) && unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func koanfFromStruct(s interface{}) (error, *koanf.Koanf) {
	var k = koanf.New(".")
	err := k.Load(structs.Provider(s, "koanf"), nil)
	if err != nil {
		return err, nil
	}

	var keys = k.Keys()
	// Assert all Keys are lowercase
	for i := 0; i < len(keys); i++ {
		if !isLower(keys[i]) {
			return errors.New(fmt.Sprintf("The Field %s in the Config Struct does not have lowercase Key. Use the `koanf` tag!", keys[i])), nil
		}
	}
	return nil, k
}

func koanfFromEnv() (*koanf.Koanf, error) {
	var k = koanf.New(".")
	err := k.Load(env.Provider("", ".", func(s string) string {
		return strings.ToLower(s)
	}), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Environment Variables to Config: %w", err)
	}

	return transformEnvFormat(k)
}

func transformEnvFormat(k *koanf.Koanf) (*koanf.Koanf, error) {
	var flattened = k.All()
	var exploded = make(map[string]interface{})
	for key, value := range flattened {
		keys := expandSlices(strings.Split(key, "_"))
		for _, newKey := range keys {
			exploded[strings.ToLower(newKey)] = value
		}
	}

	k = koanf.New(".")
	err := k.Load(confmap.Provider(exploded, "."), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse flattened Environment Variables to Config: %w", err)
	}
	return k, nil
}

func expandSlices(parts []string) []string {
	if len(parts) == 1 {
		return parts
	}

	next := expandSlices(parts[1:])
	result := make([]string, 0, len(next)*2)
	for _, k := range next {
		result = append(result, parts[0]+"."+k)
		result = append(result, parts[0]+"_"+k)
	}
	return result
}

// Decode zeroLog LogLevels from strings
func logLevelDecode(from reflect.Type, to reflect.Type, v interface{}) (interface{}, error) {
	if from.Kind() == reflect.String &&
		to.Name() == "Level" && to.PkgPath() == "github.com/rs/zerolog" {
		switch v {
		case "panic":
			return zerolog.PanicLevel, nil
		case "fatal":
			return zerolog.FatalLevel, nil
		case "error":
			return zerolog.ErrorLevel, nil
		case "warn":
			return zerolog.WarnLevel, nil
		case "debug":
			return zerolog.DebugLevel, nil
		default:
			return zerolog.InfoLevel, nil
		}
	}
	return v, nil
}