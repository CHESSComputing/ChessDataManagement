package main

// query module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"gopkg.in/mgo.v2/bson"
)

// separator defines our query separator
var separator = ":"

func convertType(val interface{}) interface{} {
	switch v := val.(type) {
	case []interface{}:
		return v
	case string:
		if IsInt(fmt.Sprintf("%v", v)) {
			v, e := strconv.Atoi(v)
			if e == nil {
				return v
			}
		}
		if IsFloat(fmt.Sprintf("%v", v)) {
			v, e := strconv.ParseFloat(v, 64)
			if e == nil {
				return v
			}
		}
		if strings.Contains(v, ",") {
			var out []string
			for _, vvv := range strings.Split(v, ",") {
				out = append(out, strings.Trim(vvv, " "))
			}
			return out
		}
		return val
	default:
		return val
	}
}

// ParseQuery function provides basic parser for user queries and return
// results in bson dictionary
func ParseQuery(query string) bson.M {
	spec := make(bson.M)
	if strings.TrimSpace(query) == "" {
		log.Println("WARNING: empty query string")
		return nil
	}
	// support MongoDB specs
	if strings.Contains(query, "{") {
		if err := json.Unmarshal([]byte(query), &spec); err == nil {
			if Config.Verbose > 0 {
				log.Printf("found bson spec %+v", spec)
			}
			return spec
		}
	}

	// query as key:value
	if strings.Contains(query, separator) {
		arr := strings.Split(query, separator)
		var vals []string
		key := arr[0]
		last := arr[len(arr)-1]
		for i := 0; i < len(arr); i++ {
			if len(arr) > i+1 {
				vals = strings.Split(arr[i+1], " ")
				if arr[i+1] == last {
					spec[key] = last
					break
				}
				if len(vals) > 0 {
					values := strings.Join(vals[:len(vals)-1], " ")
					spec[key] = values
					key = vals[len(vals)-1]
				} else {
					spec[key] = vals[0]
					break
				}
			} else {
				vals = arr[i:]
				values := strings.Join(vals, " ")
				spec[key] = values
				break
			}
		}
	} else {
		// or, query as free text
		spec["$text"] = bson.M{"$search": query}
	}
	return spec
}
