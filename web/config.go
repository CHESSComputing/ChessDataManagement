package main

// configuration module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"

	logs "github.com/sirupsen/logrus"
)

// Configuration stores server configuration parameters
type Configuration struct {
	Port            int      `json:"port"`                 // server port number
	Uri             string   `json:"uri"`                  // server mongodb URI
	DBName          string   `json:"dbname"`               // mongo db name
	DBColl          string   `json:"dbcoll"`               // mongo db name
	FilesDBUri      string   `json:"filesdburi"`           // server FilesDB URI
	Templates       string   `json:"templates"`            // location of server templates
	Jscripts        string   `json:"jscripts"`             // location of server JavaScript files
	Images          string   `json:"images"`               // location of server images
	Styles          string   `json:"styles"`               // location of server CSS styles
	LogFormatter    string   `json:"logFormatter"`         // LogFormatter type
	Verbose         int      `json:"verbose"`              // verbosity level
	Realm           string   `json:"realm"`                // kerberos realm
	Keytab          string   `json:"keytab"`               // kerberos keytab
	Krb5Conf        string   `json:"krb5Conf"`             // kerberos krb5.conf
	ServerKey       string   `json:"ckey"`                 // tls.key file
	ServerCrt       string   `json:"cert"`                 // tls.cert file
	MandatoryAttrs  []string `json:"mandatoryAttributes"`  // list of madatory attributes
	AdjustableAttrs []string `json:"adjustableAttributes"` // list of adjustable attributes
}

// Config variable represents configuration object
var Config Configuration

// String returns string representation of server Config
func (c *Configuration) String() string {
	data, _ := json.Marshal(c)
	return fmt.Sprintf(string(data))
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
	sort.Sort(StringList(Config.MandatoryAttrs))
	sort.Sort(StringList(Config.AdjustableAttrs))
	return nil
}

// ChessMetaData represents input CHESS meta-data
type ChessMetaData struct {
	User        string `json:"user"`
	Name        string `json:"name"`
	Experiment  string `json:"experiment"`
	Path        string `json:"path"`
	Processing  string `json:"processing"`
	Tier        string `json:"tier"`
	Description string `json:"description"`
}

// String returns string representation of server Config
func (c *ChessMetaData) String() string {
	data, _ := json.Marshal(c)
	return fmt.Sprintf(string(data))
}

// ToRecord convert ChessMetaData structure into json record
func (c *ChessMetaData) ToRecord() Record {
	rec := make(Record)
	data, _ := json.Marshal(c)
	json.Unmarshal(data, &rec)
	return rec
}
