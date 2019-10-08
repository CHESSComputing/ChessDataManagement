package main

// query module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/mgo.v2/bson"
)

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
		return nil
	}
	if strings.TrimSpace(query) == "__all__" {
		return spec
	}
	var description string
	for _, item := range strings.Split(query, " ") {
		val := strings.Split(strings.TrimSpace(item), ":")
		if len(val) == 2 {
			spec[val[0]] = convertType(val[1])
		} else {
			description = fmt.Sprintf("%s %s", description, val)
		}
	}
	if description != "" {
		spec["$text"] = bson.M{"$search": description}
	}
	return spec
}
