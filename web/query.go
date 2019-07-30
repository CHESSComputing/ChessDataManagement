package main

// query module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"fmt"
	"strings"

	"gopkg.in/mgo.v2/bson"
)

// ParseQuery function provides basic parser for user queries and return
// results in bson dictionary
func ParseQuery(inputQuery []string) bson.M {
	query := strings.Join(inputQuery, " ")
	if strings.TrimSpace(query) == "" {
		return nil
	}
	spec := make(bson.M)
	for _, item := range strings.Split(query, " ") {
		val := strings.Split(strings.TrimSpace(item), ":")
		if len(val) == 2 {
			spec[val[0]] = val[1]
		} else {
			if v, ok := spec["free"]; ok {
				spec["free"] = fmt.Sprintf("%s %s", v, val)
			} else {
				spec["free"] = val
			}
		}
	}
	return spec
}
