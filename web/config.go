package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	logs "github.com/sirupsen/logrus"
)

// Configuration stores server configuration parameters
type Configuration struct {
	Port         int    `json:"port"`         // server port number
	Uri          string `json:"uri"`          // server mongodb URI
	Templates    string `json:"templates"`    // location of server templates
	Jscripts     string `json:"jscripts"`     // location of server JavaScript files
	Images       string `json:"images"`       // location of server images
	Styles       string `json:"styles"`       // location of server CSS styles
	LogFormatter string `json:"logFormatter"` // LogFormatter type
}

// Config variable represents configuration object
var Config Configuration

// String returns string representation of server Config
func (c *Configuration) String() string {
	return fmt.Sprintf("<Config port=%d uri=%s templates=%s js=%s images=%s css=%s logFormatter=%s>", c.Port, c.Uri, c.Templates, c.Jscripts, c.Images, c.Styles, c.LogFormatter)
}

// ParseConfig parse given config file
func ParseConfig(configFile string) error {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		logs.WithFields(logs.Fields{"configFile": configFile}).Fatal("Unable to read", err)
		return err
	}
	err = json.Unmarshal(data, &Config)
	if err != nil {
		logs.WithFields(logs.Fields{"configFile": configFile}).Fatal("Unable to parse", err)
		return err
	}
	return nil
}
