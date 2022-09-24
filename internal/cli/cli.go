package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
)

type cli struct {
	debug      bool
	configFile string
	initOnce   sync.Once
	errOnce    error
	config     config
}

type config struct {
	DBFile            string              `json:"db_file"`
	SourceDir         string              `json:"source_dir"`
	DestinationDir    string              `json:"destination_dir"`
	DeviceNames       map[string][]string `json:"device_names"`
}

func (c *cli) setup(ctx context.Context) error {
	return c.init()
}

func (c *cli) init() error {
	c.initOnce.Do(func() {
		if c.errOnce = c.initContext(); c.errOnce != nil {
			return
		}
		cobra.EnableCommandSorting = false
	})
	return c.errOnce
}

func (c *cli) initContext() error {
	if c.configFile == "" {
		return fmt.Errorf("config file is not set")
	}

	var buf []byte
	var err error
	if buf, err = ioutil.ReadFile(c.configFile); err != nil {
		return err
	}

	if err := json.Unmarshal(buf, &c.config); err != nil {
		return err
	}

	if c.config.DBFile == "" {
		return fmt.Errorf("DBFile not set")
	}

	return nil
}

func (c *cli) persistConfig() error {
	if c.configFile == "" {
		return fmt.Errorf("configFile not set")
	}

	dir := filepath.Dir(c.configFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}

	if c.config.DBFile == "" {
		return fmt.Errorf("DBfile not set")
	}

	buf, err := json.MarshalIndent(c.config, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(c.configFile, buf, 0600)
}
