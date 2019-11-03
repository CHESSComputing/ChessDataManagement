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
	"strings"

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
	RootCA          string   `json:"rootCA"`               // RootCA file
	MandatoryAttrs  []string `json:"mandatoryAttributes"`  // list of madatory attributes
	AdjustableAttrs []string `json:"adjustableAttributes"` // list of adjustable attributes
	TestMode        bool     `json:"testMode"`             // test mode to bypass auth
}

// Config variable represents configuration object
var Config Configuration

// String returns string representation of server Config
func (c *Configuration) String() string {
	dbAttrs := strings.Split(c.FilesDBUri, "@")
	var cc Configuration
	cc = *c
	cc.FilesDBUri = dbAttrs[1]
	data, _ := json.Marshal(cc)
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

// MetaData provides details about CHESS experiment
type MetaData struct {
	Date             int64  `json:"Date"`
	PI               string `json:"PI"`
	Affliation       string `json:"Affiliation"`
	Email            string `json:"Email"`
	Proposal         string `json:"Proposal"`
	BTR              int64  `json:"BTR"`
	RawDataDirectory string `json:"RawDataDirectory"`
	AuxDataDirectory string `json:"AuxDataDirectory"`
}

// Sample defines details of used sample in CHESS experiment
type Material struct {
	SpecName            string   `json:"SpecName"`
	CalibrationSample   bool     `json:"CalibrationSample"`
	MaterialClass       string   `json:"MaterialClass"`
	CommonName          string   `json:"CommonName"`
	AbbreviatedName     string   `json:"AbbreviatedName"`
	ConstituentElements []string `json:"ConstituentElements"`
	Phases              []string `json:"Phases"`
	Processing          string   `json:"Processing"`
}

// Experiment provides Meta-Data attributes about CHESS experiment
type Experiment struct {
	ExperimentType            []string `json:"ExperimentType"`
	XrayModality              string   `json:"XrayModality"`
	XrayTechnique             []string `json:"XrawTechnique"`
	SupplementaryMeasurements []string `json:"SupplementaryMeasurements"`
	Furnance                  []string `json:"Furnance"`
	LoadFrame                 []string `json:"LoadFrame"`
	Detectors                 []string `json:"Detectors"`
}

// ChessMetaData represents input CHESS meta-data
type ChessMetaData struct {
	User        string     `json:"user"`
	MetaData    MetaData   `json:"MetaData"`
	Material    Material   `json:"Material"`
	Experiment  Experiment `json:"Experiment"`
	Description string     `json:"description"`
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
