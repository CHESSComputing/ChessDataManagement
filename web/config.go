package main

// configuration module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//
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
	Verbose      int    `json:"verbose"`      // verbosity level
	Realm        string `json:"realm"`        // kerberos realm
	Keytab       string `json:"keytab"`       // kerberos keytab
	Krb5Conf     string `json:"krb5Conf"`     // kerberos krb5.conf
	ServerKey    string `json:"ckey"`         // tls.key file
	ServerCrt    string `json:"cert"`         // tls.cert file
}

// Config variable represents configuration object
var Config Configuration

// String returns string representation of server Config
func (c *Configuration) String() string {
	return fmt.Sprintf("<Config port=%d templates=%s js=%s images=%s css=%s logFormatter=%s>", c.Port, c.Templates, c.Jscripts, c.Images, c.Styles, c.LogFormatter)
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

// ChessMetaData represents input CHESS meta-data
type ChessMetaData struct {
	User       string `json:"user"`
	Name       string `json:"name"`
	Experiment string `json:"experiment"`
}

// String returns string representation of server Config
func (c *ChessMetaData) String() string {
	return fmt.Sprintf("user=%s name=%s experiment=%s", c.User, c.Name, c.Experiment)
}
