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
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

// SchemaRenewInterval setup interal to update schema cache
var SchemaRenewInterval time.Duration

// SchemaManager holds current MetaData schema
type SchemaManager struct {
	Schema   *Schema
	LoadTime time.Time
}

// Schema returns either cached schema map or load it from provided file
func (m *SchemaManager) String() string {
	return fmt.Sprintf("%s, loaded %v", m.Schema, m.LoadTime)
}

// Schema returns either cached schema map or load it from provided file
func (m *SchemaManager) Load(fname string) (*Schema, error) {
	if m.Schema == nil || time.Since(m.LoadTime) > SchemaRenewInterval {
		m.Schema = &Schema{FileName: fname}
		err := m.Schema.Load()
		if err != nil {
			return m.Schema, err
		}
		m.LoadTime = time.Now()
	}
	return m.Schema, nil
}

// SchemaRecord provide schema record structure
type SchemaRecord struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	Optional    bool   `json:"optional"`
	Section     string `json:"section"`
	Value       any    `json:"value"`
	Placeholder any    `json:"placeholder"`
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
	for k, v := range rec {
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
		}
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
	return keys, nil
}

// Sections provide list of schema sections
func (s *Schema) Sections() ([]string, error) {
	var keys []string
	if err := s.Load(); err != nil {
		return keys, err
	}
	for k, _ := range s.Map {
		if m, ok := s.Map[k]; ok {
			if m.Section != "" {
				keys = append(keys, m.Section)
			}
		}
	}
	return keys, nil
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
