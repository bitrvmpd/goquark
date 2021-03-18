package cfg

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

const ConfigPath = "goquark"

type cfgNode struct {
	Index int
	Alias string
	Path  string
}

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
	m := map[string]string{name: path}
	s := len(viper.AllKeys())
	viper.Set(strconv.Itoa(s), m)
	if err := viper.WriteConfig(); err != nil {
		log.Fatalf("ERROR: %v", err)
	}
}

func RemoveFolder(idx int) {
	if !viper.IsSet(strconv.Itoa(idx)) {
		log.Println("value is not here..")
	}
	viper.Set(strconv.Itoa(idx), nil)
	if err := viper.WriteConfig(); err != nil {
		log.Fatalf("ERROR: %v", err)
	}
}

// Returns an array of maps.
// It'll always be ordered.
func ListFolders() []cfgNode {
	vKeys := viper.AllKeys()
	folders := make([]cfgNode, len(vKeys))

	for _, key := range vKeys {
		comK := strings.Split(key, ".")
		v, err := strconv.Atoi(comK[0])
		if err != nil {
			log.Printf("ERROR: %v", err)
		}
		folders[v] = cfgNode{
			Index: v,
			Alias: comK[1],
			Path:  viper.GetString(key),
		}
	}

	return folders
}
