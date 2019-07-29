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
	DBName       string `json:"dbname"`       // mongo db name
	DBColl       string `json:"dbcoll"`       // mongo db name
	FilesDBUri   string `json:"filesdburi"`   // server FilesDB URI
	Templates    string `json:"templates"`    // location of server templates
	Jscripts     string `json:"jscripts"`     // location of server JavaScript files
	Images       string `json:"images"`       // location of server images
	Styles       string `json:"styles"`       // location of server CSS styles
	LogFormatter string `json:"logFormatter"` // LogFormatter type
	ClientID     string `json:"clientId"`     // clientID for OAuth
	ClientSecret string `json:"clientSecret"` // clientSercret for OAuth
	Redirect     string `json:"redirect"`     // redirect URI on Github app
	StoreSecret  string `json:"storeSecret"`  // web server store secret string
}

// Config variable represents configuration object
var Config Configuration

// String returns string representation of server Config
func (c *Configuration) String() string {
	return fmt.Sprintf("<Config port=%d uri=%s dbname=%s dbcoll=%s templates=%s js=%s images=%s css=%s logFormatter=%s>", c.Port, c.Uri, c.DBName, c.DBColl, c.Templates, c.Jscripts, c.Images, c.Styles, c.LogFormatter)
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
