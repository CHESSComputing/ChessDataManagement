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
func ParseQuery(query string) bson.M {
	if strings.TrimSpace(query) == "" {
		return nil
	}
	spec := make(bson.M)
	var description string
	for _, item := range strings.Split(query, " ") {
		val := strings.Split(strings.TrimSpace(item), ":")
		if len(val) == 2 {
			spec[val[0]] = val[1]
		} else {
			description = fmt.Sprintf("%s %s", description, val)
		}
	}
	if description != "" {
		spec["$text"] = bson.M{"$search": description}
	}
	return spec
}
