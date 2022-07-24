package core

import (
	log "github.com/livelace/logrus"
	"github.com/spf13/viper"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

func initAppConfig() (string, error) {
	configFile := "config.toml"

	// Get current user info.
	// Ignore user absence.
	userDir := ""
	userAccount, err := user.Current()
	if err == nil {
		userDir = filepath.Join(userAccount.HomeDir, ".gosquito")
	}

	// Possible config paths.
	// Path priority order.
	// 1. /etc/gosquito/
	// 2. ~/.gosquito
	// 3. .gosquito
    if _, err := IsFile(filepath.Join(DEFAULT_ETC_PATH, configFile)); err == nil {
		return DEFAULT_ETC_PATH, nil

    } else if _, err := IsFile(filepath.Join(userDir, configFile)); err == nil {
		return userDir, nil

    } else if _, err := IsFile(filepath.Join(DEFAULT_CURRENT_PATH, configFile)); err == nil {
		return DEFAULT_CURRENT_PATH, nil
	}

	// Write config sample if config not found.
	if err := CreateDirIfNotExist(userDir); err != nil {
		return userDir, err
	}

	if err := WriteStringToFile(userDir, configFile, CONFIG_SAMPLE); err != nil {
		return userDir, err
	}

	return userDir, nil
}

func GetAppConfig() *viper.Viper {
	configPath, err := initAppConfig()

	if err != nil {
		log.WithFields(log.Fields{
			"path":  configPath,
			"error": err,
		}).Error(LOG_CONFIG_ERROR)
		os.Exit(1)
	}

	// Show user config path.
	if configPath != "" {
		log.WithFields(log.Fields{
			"path": configPath,
		}).Info(LOG_CONFIG_APPLY)
	}

	// Read generated/existed configuration.
	v := viper.New()
	v.SetConfigName("config.toml")
	v.SetConfigType("toml")
	v.AddConfigPath(configPath)

	if err := v.ReadInConfig(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error(LOG_CONFIG_ERROR)
		os.Exit(1)
	}

	// Set defaults.
	v.SetDefault(VIPER_DEFAULT_EXPIRE_ACTION, make([]string, 0))
	v.SetDefault(VIPER_DEFAULT_EXPIRE_ACTION_DELAY, DEFAULT_EXPIRE_ACTION_DELAY)
	v.SetDefault(VIPER_DEFAULT_EXPIRE_ACTION_TIMEOUT, DEFAULT_EXPIRE_ACTION_TIMEOUT)
	v.SetDefault(VIPER_DEFAULT_EXPIRE_INTERVAL, DEFAULT_EXPIRE_INTERVAL)
	v.SetDefault(VIPER_DEFAULT_EXPORTER_LISTEN, DEFAULT_EXPORTER_LISTEN)
	v.SetDefault(VIPER_DEFAULT_FLOW_CLEANUP, DEFAULT_FLOW_CLEANUP)
	v.SetDefault(VIPER_DEFAULT_FLOW_CONF, filepath.Join(configPath, DEFAULT_FLOW_CONF_DIR))
	v.SetDefault(VIPER_DEFAULT_FLOW_DATA, filepath.Join(configPath, DEFAULT_FLOW_DATA_DIR))
	v.SetDefault(VIPER_DEFAULT_FLOW_ENABLE, make([]string, 0))
	v.SetDefault(VIPER_DEFAULT_FLOW_INSTANCE, DEFAULT_FLOW_INSTANCE)
	v.SetDefault(VIPER_DEFAULT_FLOW_INTERVAL, DEFAULT_FLOW_INTERVAL)
	v.SetDefault(VIPER_DEFAULT_FLOW_LIMIT, DEFAULT_FLOW_LIMIT)
	v.SetDefault(VIPER_DEFAULT_LOG_LEVEL, DEFAULT_LOG_LEVEL)
	v.SetDefault(VIPER_DEFAULT_PLUGIN_INCLUDE, DEFAULT_PLUGIN_INCLUDE)
	v.SetDefault(VIPER_DEFAULT_PLUGIN_TIMEOUT, DEFAULT_PLUGIN_TIMEOUT)
	v.SetDefault(VIPER_DEFAULT_PROC_NUM, runtime.GOMAXPROCS(0))
	v.SetDefault(VIPER_DEFAULT_TIME_FORMAT, DEFAULT_TIME_FORMAT)
	v.SetDefault(VIPER_DEFAULT_TIME_ZONE, DEFAULT_TIME_ZONE)
	v.SetDefault(VIPER_DEFAULT_USER_AGENT, DEFAULT_USER_AGENT)

	// Directories must exist for proper work.
	workDirs := []string{
		v.GetString(VIPER_DEFAULT_FLOW_CONF),
		v.GetString(VIPER_DEFAULT_FLOW_DATA),
	}

	for _, workDir := range workDirs {
		if err := CreateDirIfNotExist(workDir); err != nil {
			log.WithFields(log.Fields{
				"path":  workDir,
				"error": err,
			}).Error(LOG_CONFIG_ERROR)
			os.Exit(1)
		}
	}

	return v
}
