package cfg

import (
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const ConfigPath = "goquark.yaml"

type cfgRoot struct {
	Nodes []cfgNode `yaml:"nodes"`
}
type cfgNode struct {
	Alias string `yaml:"alias"`
	Path  string `yaml:"path"`
	index int
}

var cfg cfgRoot = cfgRoot{}

func init() {
	loadConfig()
}

func loadConfig() {
	userDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("ERROR: %v", err)
	}

	// Open or Create if not found.
	f, err := os.OpenFile(filepath.Join(userDir, ConfigPath), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("ERROR: %v", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Printf("ERROR: %v", err)
	}

	//Fill current index
	for i := 0; i < len(cfg.Nodes); i++ {
		cfg.Nodes[i].index = i
	}
}

func writeConfig() {
	userDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("ERROR: %v", err)
	}

	// Delete the file, to prevent weird bugs.
	os.Remove(filepath.Join(userDir, ConfigPath))

	// Open or Create if not found.
	f, err := os.OpenFile(filepath.Join(userDir, ConfigPath), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("ERROR: %v", err)
	}
	defer f.Close()
	encoder := yaml.NewEncoder(f)
	err = encoder.Encode(cfg)
	if err != nil {
		log.Printf("ERROR: %v", err)
	}
}

func Size() uint32 {
	return uint32(len(cfg.Nodes))
}

func AddFolder(name string, path string) {
	cfg.Nodes = append(cfg.Nodes, cfgNode{name, path, len(cfg.Nodes)})
	writeConfig()
}

func RemoveFolder(idx int) {
	// Not as efficient, but it works.
	cfg.Nodes = append(cfg.Nodes[:idx], cfg.Nodes[idx+1:]...)
	// Update index
	for i := 0; i < len(cfg.Nodes); i++ {
		cfg.Nodes[i].index = i
	}
	writeConfig()
}

func ListFolders() []cfgNode {
	return cfg.Nodes
}
