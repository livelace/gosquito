package core

import (
	log "github.com/livelace/logrus"
	"github.com/spf13/viper"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

const (
	FLOW_DIR   = "flow"
	PLUGIN_DIR = "plugin"
	STATE_DIR  = "state"
	TEMP_DIR   = "temp"
)

/*
	Get and load configuration file, set defaults.
*/
func GetConfig() *viper.Viper {
	// 1. Create configuration directory.
	// 2. Put sample configuration.
	// 3. Put flow examples.
	// dir == "", if configuration directories already exist.
	dir, err := initConfig()

	if err != nil {
		log.WithFields(log.Fields{
			"path":  dir,
			"error": err,
		}).Error(ERROR_CONFIG_INIT)
		os.Exit(1)
	}

	if dir != "" {
		log.WithFields(log.Fields{
			"path": dir,
		}).Info(LOG_CONFIG_INIT)
	}

	// Read generated/existed configuration.
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(DEFAULT_ETC_PATH)
	v.AddConfigPath("$HOME/.gosquito")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error(ERROR_CONFIG_READ)
		os.Exit(1)
	}

	// Set defaults.
	v.SetDefault(VIPER_DEFAULT_EXPIRE_ACTION, make([]string, 0))
	v.SetDefault(VIPER_DEFAULT_EXPIRE_ACTION_TIMEOUT, DEFAULT_EXPIRE_ACTION_TIMEOUT)
	v.SetDefault(VIPER_DEFAULT_EXPIRE_DELAY, DEFAULT_EXPIRE_DELAY)
	v.SetDefault(VIPER_DEFAULT_EXPIRE_INTERVAL, DEFAULT_EXPIRE_INTERVAL)
	v.SetDefault(VIPER_DEFAULT_FLOW_DIR, filepath.Join(filepath.Dir(v.ConfigFileUsed()), FLOW_DIR))
	v.SetDefault(VIPER_DEFAULT_FLOW_ENABLE, make([]string, 0))
	v.SetDefault(VIPER_DEFAULT_FLOW_INTERVAL, DEFAULT_FLOW_INTERVAL)
	v.SetDefault(VIPER_DEFAULT_FLOW_NUMBER, DEFAULT_FLOW_NUMBER)
	v.SetDefault(VIPER_DEFAULT_LOG_LEVEL, DEFAULT_LOG_LEVEL)
	v.SetDefault(VIPER_DEFAULT_EXPORTER_LISTEN, DEFAULT_EXPORTER_LISTEN)
	v.SetDefault(VIPER_DEFAULT_PLUGIN_DIR, filepath.Join(filepath.Dir(v.ConfigFileUsed()), PLUGIN_DIR))
	v.SetDefault(VIPER_DEFAULT_PLUGIN_INCLUDE, DEFAULT_PLUGIN_INCLUDE)
	v.SetDefault(VIPER_DEFAULT_PLUGIN_TIMEOUT, DEFAULT_PLUGIN_TIMEOUT)
	v.SetDefault(VIPER_DEFAULT_PROC_MAX, runtime.GOMAXPROCS(0))
	v.SetDefault(VIPER_DEFAULT_STATE_DIR, filepath.Join(filepath.Dir(v.ConfigFileUsed()), STATE_DIR))
	v.SetDefault(VIPER_DEFAULT_TEMP_DIR, filepath.Join(filepath.Dir(v.ConfigFileUsed()), TEMP_DIR))
	v.SetDefault(VIPER_DEFAULT_TIME_FORMAT, DEFAULT_TIME_FORMAT)
	v.SetDefault(VIPER_DEFAULT_TIME_ZONE, DEFAULT_TIME_ZONE)
	v.SetDefault(VIPER_DEFAULT_USER_AGENT, DEFAULT_USER_AGENT)

	// Directories must exist for proper work.
	configDirs := []string{
		v.GetString(VIPER_DEFAULT_FLOW_DIR),
		v.GetString(VIPER_DEFAULT_PLUGIN_DIR),
		v.GetString(VIPER_DEFAULT_STATE_DIR),
		v.GetString(VIPER_DEFAULT_TEMP_DIR),
	}

	for _, dir := range configDirs {
		if err := CreateDirIfNotExist(dir); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error(ERROR_CONFIG_INIT)
			os.Exit(1)
		}
	}

	return v
}

func initConfig() (string, error) {
	// Get current user info.
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	// Possible config paths.
	userDir := filepath.Join(usr.HomeDir, ".gosquito")
	configFile := "config.toml"

	// Exit, if config exists.
	if IsFile(userDir, configFile) || IsFile(DEFAULT_ETC_PATH, configFile) {
		return "", nil
	}

	// Initialize basic directories structure.
	if err := CreateDirIfNotExist(userDir); err != nil {
		return "", err
	}

	for _, dir := range []string{FLOW_DIR, STATE_DIR, TEMP_DIR} {
		if err := CreateDirIfNotExist(filepath.Join(userDir, dir)); err != nil {
			return "", err
		}
	}

	// Write config sample.
	if err := WriteStringToFile(userDir, configFile, CONFIG_SAMPLE); err != nil {
		return "", err
	}

	return userDir, nil
}
