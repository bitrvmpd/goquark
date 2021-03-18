package cfg

import (
	"fmt"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

const ConfigPath = "goquark"

func init() {
	viper.SetConfigName(ConfigPath) // name of config file (without extension
	viper.SetConfigType("yaml")
	userDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("ERROR: %v", err)
	}
	viper.AddConfigPath(userDir) // optionally look for config in the working directory

	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil {
		// Handle errors reading the config file
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			viper.AllowEmptyEnv(true)
			err = viper.SafeWriteConfig() // writes current config to predefined path set by 'viper.AddConfigPath()' and 'viper.SetConfigName'
			if err != nil {
				log.Fatalf("Couldn't write config file: %v \n", err)
			}
		} else {
			// Config file was found but another error was produced
			log.Fatalf("Fatal error config file: %s \n", err)

		}
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})
}

func Size() uint32 {
	return uint32(len(viper.AllKeys()))
}

func AddFolder(name string, path string) {
	viper.Set(name, path)
	if err := viper.WriteConfig(); err != nil {
		log.Fatalf("ERROR: %v", err)
	}
}

func RemoveFolder(name string) {
	if !viper.IsSet(name) {
		log.Println("value is not here..")
	}
	viper.Set(name, nil)
	if err := viper.WriteConfig(); err != nil {
		log.Fatalf("ERROR: %v", err)
	}
}

func ListFolders() map[string]string {
	vKeys := viper.AllKeys()
	folders := make(map[string]string, len(vKeys))
	for _, key := range vKeys {
		folders[key] = viper.GetString(key)
	}
	return folders
}
