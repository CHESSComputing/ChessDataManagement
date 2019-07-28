package main

import (
	"strings"

	"gopkg.in/mgo.v2/bson"
)

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
			spec["free"] = val
		}
	}
	return spec
}
