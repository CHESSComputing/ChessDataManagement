package main

// schema module
//
// Copyright (c) 2022 - Valentin Kuznetsov <vkuznet AT gmail dot com>
//

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

// skip keys
var _skipKeys = []string{"User", "Date", "Description", "SchemaName", "SchemaFile", "Schema"}

// SchemaKeys represents full collection of schema keys across all schemas
type SchemaKeys map[string]string

// schema keys map
var _schemaKeys SchemaKeys

// SchemaRenewInterval setup interal to update schema cache
var SchemaRenewInterval time.Duration

// SchemaObject holds current MetaData schema
type SchemaObject struct {
	Schema   *Schema
	LoadTime time.Time
}

// SchemaManager holds current map of MetaData schema objects
type SchemaManager struct {
	Map map[string]*SchemaObject
}

// Schema returns either cached schema map or load it from provided file
func (m *SchemaManager) String() string {
	var out string
	for k, v := range m.Map {
		out += fmt.Sprintf("\n%s %s, loaded %v\n", k, v.Schema, v.LoadTime)
	}
	return out
}

// Schema returns either cached schema map or load it from provided file
func (m *SchemaManager) Load(fname string) (*Schema, error) {
	// use full path of file name
	fname = fullPath(fname)

	// check fname in our schema map
	if sobj, ok := m.Map[fname]; ok {
		if sobj.Schema != nil && time.Since(sobj.LoadTime) < SchemaRenewInterval {
			log.Println("schema taken from cache", fname)
			return sobj.Schema, nil
		}
	}
	schema := &Schema{FileName: fname}
	err := schema.Load()
	if err != nil {
		log.Println("unable to load schema from", fname, " error", err)
		return schema, err
	}
	log.Println("renew schema:", fname)
	// reset map if it is expired
	if sobj, ok := m.Map[fname]; ok {
		if sobj.Schema != nil && time.Since(sobj.LoadTime) > SchemaRenewInterval {
			log.Println("reset schema manager")
			m.Map = nil
		}
	}
	if m.Map == nil {
		m.Map = make(map[string]*SchemaObject)
	}
	m.Map[fname] = &SchemaObject{Schema: schema, LoadTime: time.Now()}

	return schema, nil
}

// SchemaRecord provide schema record structure
type SchemaRecord struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	Optional    bool   `json:"optional"`
	Multiple    bool   `json:"multiple"`
	Section     string `json:"section"`
	Value       any    `json:"value"`
	Placeholder string `json:"placeholder"`
	Description string `json:"description"`
}

// Schema provides structure of schema file
type Schema struct {
	FileName       string                  `json:"fileName`
	Map            map[string]SchemaRecord `json:"map"`
	WebSectionKeys map[string][]string     `json:"webSectionKeys"`
}

// Load loads given schema file
func (s *Schema) String() string {
	if s.Map != nil {
		return fmt.Sprintf("<schema %s, map %d entries>", s.FileName, len(s.Map))
	}
	return fmt.Sprintf("<schema %s, map %v>", s.FileName, s.Map)
}

// Load loads given schema file
func (s *Schema) Load() error {
	fname := s.FileName
	file, err := os.Open(fname)
	if err != nil {
		msg := fmt.Sprintf("Unable to open %s, error=%v", fname, err)
		log.Printf("ERROR: %s", msg)
		return errors.New(msg)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		msg := fmt.Sprintf("Unable to read %s, error=%v", fname, err)
		log.Printf("ERROR: %s", msg)
		return errors.New(msg)
	}
	var records []SchemaRecord
	if strings.HasSuffix(fname, "json") {
		err = json.Unmarshal(data, &records)
		if err != nil {
			msg := fmt.Sprintf("fail to unmarshal json file %s, error=%v", fname, err)
			log.Printf("ERROR: %s", msg)
			return errors.New(msg)
		}
	} else if strings.HasSuffix(fname, "yaml") || strings.HasSuffix(fname, "yml") {
		var yrecords []map[interface{}]interface{}
		err = yaml.Unmarshal(data, &yrecords)
		if err != nil {
			msg := fmt.Sprintf("fail to unmarshal yaml file %s, error=%v", fname, err)
			log.Printf("ERROR: %s", msg)
			return errors.New(msg)
		}
		for _, yr := range yrecords {
			m := convertYaml(yr)
			smap := SchemaRecord{}
			for k, v := range m {
				if k == "key" {
					smap.Key = v.(string)
				} else if k == "type" {
					smap.Type = v.(string)
				} else if k == "optional" {
					smap.Optional = v.(bool)
				} else if k == "description" {
					smap.Description = v.(string)
				} else if k == "placeholder" {
					smap.Placeholder = v.(string)
				}
			}
			records = append(records, smap)
		}
	} else {
		msg := fmt.Sprintf("unsupported data format of schema file %s", fname)
		log.Printf("ERROR: %s", msg)
		return errors.New(msg)
	}
	s.FileName = fname
	smap := make(map[string]SchemaRecord)
	for _, r := range records {
		smap[r.Key] = r
	}
	// update schema map
	s.Map = smap

	// upload SchemaKeys object
	if _schemaKeys == nil {
		_schemaKeys = make(SchemaKeys)
	}
	for _, r := range smap {
		if _, ok := _schemaKeys[r.Key]; !ok {
			_schemaKeys[strings.ToLower(r.Key)] = r.Key
		}
	}

	// either load web section schema file or use default web section keys
	base := strings.Split(fname, ".")[0]
	filepath := fmt.Sprintf("%s_web.json", base)
	if _, err := os.Stat(filepath); err == nil {
		file, err := os.Open(filepath)
		if err != nil {
			log.Println("unable to open", filepath, "error", err)
			return err
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			log.Println("unable to read file, error", err)
			return err
		}
		var rec map[string][]string
		err = json.Unmarshal(data, &rec)
		if err != nil {
			log.Fatal(err)
		}
		s.WebSectionKeys = rec
	} else {
		webKeys := make(map[string][]string)
		var skeys []string
		for k, _ := range s.Map {
			skeys = append(skeys, k)
		}
		for key, values := range Config.WebSectionKeys {
			var vals []string
			for _, v := range values {
				if InList(v, skeys) {
					vals = append(vals, v)
				}
			}
			webKeys[key] = vals
		}
		s.WebSectionKeys = webKeys
	}
	return nil
}

// Validate validates given record against schema
func (s *Schema) Validate(rec Record) error {
	if err := s.Load(); err != nil {
		return err
	}
	log.Println("INFO: ", s.String())
	keys, err := s.Keys()
	if err != nil {
		return err
	}
	// hidden mandatory keys we add to each form
	var mkeys []string
	for k, v := range rec {
		// skip user key
		if InList(k, _skipKeys) {
			continue
		}
		// check if our record key belong to the schema keys
		if !InList(k, keys) {
			msg := fmt.Sprintf("record key '%s' is not known", k)
			log.Printf("ERROR: %s, schema file %s, schema map %+v", msg, s.FileName, s.Map)
			return errors.New(msg)
		}

		if m, ok := s.Map[k]; ok {
			// check key name
			if m.Key != k {
				msg := fmt.Sprintf("invalid key=%s", k)
				log.Printf("ERROR: %s", msg)
				return errors.New(msg)
			}
			// check data type
			if !validSchemaType(m.Type, v) {
				// check if provided data type can be converted to m.Type
				msg := fmt.Sprintf("invalid data type for key=%s, value=%v, type=%T, expect=%s", k, v, v, m.Type)
				log.Printf("ERROR: %s", msg)
				return errors.New(msg)
			}
			// check data value
			if !validDataValue(m, v) {
				// check if provided data type can be converted to m.Type
				msg := fmt.Sprintf("invalid data value for key=%s, type=%s, multiple=%v, value=%v", k, m.Type, m.Multiple, v)
				log.Printf("ERROR: %s", msg)
				return errors.New(msg)
			}
			// collect mandatory keys
			if !m.Optional {
				mkeys = append(mkeys, k)
			}
		}
	}

	// check that we collected all mandatory keys
	smkeys, err := s.MandatoryKeys()
	if err != nil {
		return err
	}
	if len(mkeys) != len(smkeys) {
		sort.Sort(StringList(mkeys))
		var missing []string
		for _, k := range smkeys {
			if !InList(k, mkeys) {
				missing = append(missing, k)
			}
		}
		msg := fmt.Sprintf("Schema %s, mandatory keys %v, record keys %v, missing keys %v", s.FileName, smkeys, mkeys, missing)
		log.Printf("ERROR: %s", msg)
		return errors.New(msg)
	}
	return nil
}

// Keys provides list of keys of the schema
func (s *Schema) Keys() ([]string, error) {
	var keys []string
	if err := s.Load(); err != nil {
		return keys, err
	}
	for k, _ := range s.Map {
		keys = append(keys, k)
	}
	sort.Sort(StringList(keys))
	return keys, nil
}

// OptionalKeys provides list of optional keys of the schema
func (s *Schema) OptionalKeys() ([]string, error) {
	var keys []string
	if err := s.Load(); err != nil {
		return keys, err
	}
	for k, _ := range s.Map {
		if m, ok := s.Map[k]; ok {
			if m.Optional {
				keys = append(keys, k)
			}
		}
	}
	sort.Sort(StringList(keys))
	return keys, nil
}

// MandatoryKeys provides list of madatory keys of the schema
func (s *Schema) MandatoryKeys() ([]string, error) {
	var keys []string
	if err := s.Load(); err != nil {
		return keys, err
	}
	for k, _ := range s.Map {
		if m, ok := s.Map[k]; ok {
			if !m.Optional {
				keys = append(keys, k)
			}
		}
	}
	sort.Sort(StringList(keys))
	return keys, nil
}

// Sections provides list of schema sections
func (s *Schema) Sections() ([]string, error) {
	var sections []string
	if err := s.Load(); err != nil {
		return sections, err
	}
	for k, _ := range s.Map {
		if m, ok := s.Map[k]; ok {
			if m.Section != "" {
				sections = append(sections, m.Section)
			}
		}
	}
	if len(Config.SchemaSections) > 0 {
		// we will return sections according to order in SchemaSections
		var out []string
		out = Config.SchemaSections
		// add other section to the output
		sort.Sort(StringList(sections))
		for _, s := range sections {
			if !InList(s, out) {
				out = append(out, s)
			}
		}
		return out, nil
	}
	return sections, nil
}

// SectionKeys provides map of section keys
func (s *Schema) SectionKeys() (map[string][]string, error) {
	smap := make(map[string][]string)
	sections, err := s.Sections()
	if err != nil {
		return smap, err
	}
	allKeys, err := s.Keys()
	if err != nil {
		return smap, err
	}
	// populate section map with keys defined in webSectionKeys
	if s.WebSectionKeys != nil {
		for k, v := range s.WebSectionKeys {
			smap[k] = v
		}
	}
	// loop over all sections and add section keys to the map
	for _, sect := range sections {
		for _, k := range allKeys {
			if r, ok := s.Map[k]; ok {
				if r.Section == sect {
					if skeys, ok := smap[sect]; ok {
						if !InList(k, skeys) {
							skeys = append(skeys, k)
							smap[sect] = skeys
						}
					} else {
						smap[sect] = []string{k}
					}
				}
			}
		}
	}
	return smap, nil
}

// helper function to validate given value with respect to schema one
// only valid for value of list type
func validDataValue(rec SchemaRecord, v any) bool {
	if strings.HasPrefix(rec.Type, "list") {
		var values []string
		if rec.Value == nil {
			return true
		}
		for _, v := range rec.Value.([]any) {
			values = append(values, strings.Trim(fmt.Sprintf("%v", v), " "))
		}
		matched := false
		if Config.Verbose > 0 {
			log.Printf("checking %v of type %T against %+v", v, v, rec)
		}
		vtype := fmt.Sprintf("%T", v)
		if strings.HasPrefix(vtype, "[]") {
			// our input value is a list data-type and we should check all its values
			var matchArr []bool
			var rvalues []string
			switch rvals := v.(type) {
			case []string:
				for _, rv := range rvals {
					for _, vvv := range strings.Split(rv, " ") {
						rvalues = append(rvalues, vvv)
					}
				}
			case []any:
				for _, rv := range rvals {
					vvv := fmt.Sprintf("%v", rv)
					rvalues = append(rvalues, vvv)
				}
			}
			for _, rv := range rvalues {
				for _, vvv := range values {
					//                     if rv == vvv || (vvv != "" && strings.Contains(rv, vvv)) {
					if rv == vvv {
						matchArr = append(matchArr, true)
					}
				}
			}
			if Config.Verbose > 0 {
				log.Println("values list", values, len(values))
				log.Println("matched list", matchArr, len(matchArr))
				log.Println("expected list", rvalues, len(rvalues))
			}
			// all matched values
			if len(matchArr) == len(rvalues) {
				matched = true
			}
		} else {
			for _, val := range values {
				vvv := fmt.Sprintf("%v", val)
				if v == vvv {
					matched = true
				}
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// helper function to validate schema type of given value with respect to schema
func validSchemaType(stype string, v interface{}) bool {
	// on web form 0 will be int type, but we can allow it for any int's float's
	if v == 0 || v == 0. {
		if strings.Contains(stype, "int") || strings.Contains(stype, "float") {
			return true
		}
	}
	// check actual value type and compare it to given schema type
	var etype string
	switch v.(type) {
	case bool:
		etype = "bool"
	case int:
		etype = "int"
	case int8:
		etype = "int8"
	case int16:
		etype = "int16"
	case int32:
		etype = "int32"
	case int64:
		etype = "int64"
	case uint16:
		etype = "uint16"
	case uint32:
		etype = "uint32"
	case uint64:
		etype = "uint64"
	case float32:
		etype = "float"
	case float64:
		etype = "float64"
	case string:
		etype = "string"
	case []string:
		etype = "list_str"
	case []any:
		etype = "list_str"
	case []int:
		etype = "list_int"
	case []float64:
		etype = "list_float"
	case []float32:
		etype = "list_float"
	}
	sv := fmt.Sprintf("%v", v)
	vtype := fmt.Sprintf("%T", v)
	if Config.Verbose > 1 {
		log.Printf("### validSchemaType schema type=%v value type=%T value=%v", stype, v, sv)
	}
	if stype == "int64" && vtype == "float64" && !strings.Contains(sv, ".") {
		return true
	}
	if stype == "list_float" && vtype == "[]interface {}" {
		return true
	}
	// empty list of floats
	if stype == "list_float" && vtype == "[]string" && sv == "[]" {
		return true
	}
	// check if we can reduce data-type
	if sv == "" && etype != "string" {
		return false
	}
	if stype != etype {
		return false
	}
	return true
}

// helper function to convert yaml map to json map interface
func convertYaml(m map[interface{}]interface{}) map[string]interface{} {
	res := map[string]interface{}{}
	for k, v := range m {
		switch v2 := v.(type) {
		case map[interface{}]interface{}:
			res[fmt.Sprint(k)] = convertYaml(v2)
		default:
			res[fmt.Sprint(k)] = v
		}
	}
	return res
}
