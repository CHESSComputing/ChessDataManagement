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
	"os"
	"sort"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

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
	if sobj, ok := m.Map[fname]; ok {
		if sobj.Schema != nil && time.Since(sobj.LoadTime) < SchemaRenewInterval {
			return sobj.Schema, nil
		}
	}
	schema := &Schema{FileName: fname}
	err := schema.Load()
	if err != nil {
		return schema, err
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
	Section     string `json:"section"`
	Value       any    `json:"value"`
	Placeholder string `json:"placeholder"`
	Description string `json:"description"`
}

// Schema provides structure of schema file
type Schema struct {
	FileName string                  `json:"fileName`
	Map      map[string]SchemaRecord `json:"map"`
}

// Load loads given schema file
func (s *Schema) String() string {
	return fmt.Sprintf("<schema %s, map %d entries>", s.FileName, len(s.Map))
}

// Load loads given schema file
func (s *Schema) Load() error {
	fname := s.FileName
	file, err := os.Open(fname)
	if err != nil {
		msg := fmt.Sprintf("Unable to open %s, error=%v", fname, err)
		return errors.New(msg)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		msg := fmt.Sprintf("Unable to read %s, error=%v", fname, err)
		return errors.New(msg)
	}
	var records []SchemaRecord
	if strings.HasSuffix(fname, "json") {
		err = json.Unmarshal(data, &records)
		if err != nil {
			msg := fmt.Sprintf("fail to unmarshal json file %s, error=%v", fname, err)
			return errors.New(msg)
		}
	} else if strings.HasSuffix(fname, "yaml") || strings.HasSuffix(fname, "yml") {
		var yrecords []map[interface{}]interface{}
		err = yaml.Unmarshal(data, &yrecords)
		if err != nil {
			msg := fmt.Sprintf("fail to unmarshal yaml file %s, error=%v", fname, err)
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
		return errors.New(msg)
	}
	s.FileName = fname
	smap := make(map[string]SchemaRecord)
	for _, r := range records {
		smap[r.Key] = r
	}
	s.Map = smap
	return nil
}

// Validate validates given record against schema
func (s *Schema) Validate(rec Record) error {
	if err := s.Load(); err != nil {
		return err
	}
	keys, err := s.Keys()
	if err != nil {
		return err
	}
	var mkeys []string
	for k, v := range rec {
		// skip user key
		if k == "user" {
			continue
		}
		// check if our record key belong to the schema keys
		if !InList(k, keys) {
			msg := fmt.Sprintf("record key '%s' is not known", k)
			return errors.New(msg)
		}

		if m, ok := s.Map[k]; ok {
			// check key name
			if m.Key != k {
				msg := fmt.Sprintf("invalid key=%s", k)
				return errors.New(msg)
			}
			// check data type
			if !validSchemaType(m.Type, v) {
				msg := fmt.Sprintf("invalid data type for key=%s, value=%v, type=%T, expect=%s", k, v, v, m.Type)
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
		msg := fmt.Sprintf("Unable to collect all mandatory keys %v, found %v", smkeys, mkeys)
		return errors.New(msg)
	}
	return nil
}

// Keys provide list of keys of the schema
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

// OptionalKeys provide list of optional keys of the schema
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

// MandatoryKeys provide list of madatory keys of the schema
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

// Sections provide list of schema sections
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
		// we will return sections according to logical SchemaSection order
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

// helper function to validate schema type of given value with respect to schema
func validSchemaType(stype string, v interface{}) bool {
	var etype string
	switch v.(type) {
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
		etype = "[]string"
	case []any:
		etype = "list"
	case []int:
		etype = "[]int"
	case []float64:
		etype = "[]float64"
	case []float32:
		etype = "[]float32"
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
