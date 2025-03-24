package analyzer

type Config struct {
	IgnoreFiles []string
	Verbose     bool
}

var globalConfig *Config

func SetConfig(config *Config) {
	globalConfig = config
}

func GetConfig() *Config {
	if globalConfig == nil {
		// Default configuration
		return &Config{
			IgnoreFiles: nil,
			Verbose:     false,
		}
	}

	return globalConfig
}
