package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func main() {
	var schema string
	flag.StringVar(&schema, "schema", "", "schema file")
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	validate(schema)
}

// SchemaRecord provide schema record structure
type SchemaRecord struct {
	Key         string `json:"key" validate:"required"`
	Type        string `json:"type" validate:"required"`
	Optional    bool   `json:"optional"`
	Multiple    bool   `json:"multiple"`
	Section     string `json:"section" validate:"required"`
	Value       any    `json:"value"`
	Placeholder string `json:"placeholder"`
	Description string `json:"description"`
}

// Types represents allowed types
var Types = []string{
	"int", "int32", "int64", "uint8", "uint16", "uint32",
	"float", "float32", "float64",
	"string", "bool",
	"list_str", "list_int", "list_float",
}

// Keys represents allowed keys in schemarecord
// var Keys = []string{
//     "key", "type", "optional", "multiple", "section", "value", "placeholder", "description",
// }

func validate(fname string) {
	file, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	var records []SchemaRecord
	//     err = json.Unmarshal(bytes, &records)
	//     if err != nil {
	//         log.Fatal(err)
	//     }
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&records); err != nil {
		log.Fatalf("Unable to decode schema records, error: %v", err)
	}

	for _, rec := range records {
		if !InList(rec.Type, Types) {
			log.Fatalf("Unknown schema type %s, should be one of %v", rec.Type, Types)
		}
		// check type with provided values
		val := rec.Value
		switch vvv := val.(type) {
		case []any:
			if strings.HasPrefix(rec.Type, "list_") {
				// check individual values
				rtype := strings.Replace(rec.Type, "list_", "", -1)
				for _, v := range vvv {
					if !checkTypeValues(rtype, v) {
						log.Fatalf(
							"Provided type %s does not match values %v in record\n%+v",
							rec.Type, val, repr(rec))
					}
				}
			} else {
				// provided values are in a list but rec.Type is not a list
				// ["", "true", "false"] vs bool
				for _, v := range vvv {
					if v == "" {
						// default empty string we may omit
						continue
					}
					// check if values are in list type
					if rec.Type == "bool" {
						vb := false
						if v == "true" {
							vb = true
						}
						if !checkTypeValues(rec.Type, vb) {
							log.Fatalf(
								"Provided type '%s' does not match values '%v' from list '%v' in record\n%+v",
								rec.Type, v, val, repr(rec))
						}
					} else {
						if !checkTypeValues(rec.Type, v) {
							log.Fatalf(
								"Provided type '%s' does not match values '%v' from list '%v' in record\n%+v",
								rec.Type, v, val, repr(rec))
						}
					}
				}
			}
		default:
			if !checkTypeValues(rec.Type, val) {
				log.Fatalf(
					"Provided type %s does not match values %v in record\n%+v",
					rec.Type, val, repr(rec))
			}
		}
	}
}

func checkTypeValues(rtype string, v any) bool {
	if v == nil { // nothing to be checked
		return true
	}
	if v == 0 || v == 0. {
		if strings.Contains(rtype, "int") || strings.Contains(rtype, "float") {
			return true
		}
	}
	if v == "" {
		// for empty values we simply return
		return true
	}
	if rtype == "str" {
		rtype = "string"
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
	if rtype == "int64" && vtype == "float64" && !strings.Contains(sv, ".") {
		return true
	}
	if rtype == "list_float" && vtype == "[]interface {}" {
		return true
	}
	if rtype != etype {
		return false
	}
	return true
}

// InList helper function to check item in a list
func InList(a string, list []string) bool {
	check := 0
	for _, b := range list {
		if b == a {
			check++
		}
	}
	if check != 0 {
		return true
	}
	return false
}

func repr(rec SchemaRecord) string {
	data, err := json.MarshalIndent(rec, "", "    ")
	if err == nil {
		return string(data)
	}
	return fmt.Sprintf("%+v", rec)
}
