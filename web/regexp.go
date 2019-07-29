package main

// regexp module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"regexp"
)

// PatternInt represents an integer pattern
var PatternInt = regexp.MustCompile("(^[0-9-]$|^[0-9-][0-9]*$)")

// PatternUrl represents URL pattern
var PatternUrl = regexp.MustCompile("(https|http)://[-A-Za-z0-9_+&@#/%?=~_|!:,.;]*[-A-Za-z0-9+&@#/%=~_|]")

// PatternDataset represents CHESS dataset
var PatternDataset = regexp.MustCompile("/[-a-zA-Z_0-9*]+/[-a-zA-Z_0-9*]+/[-a-zA-Z_0-9*]+")

// PatternFile represents CHESS file
var PatternFile = regexp.MustCompile("/[a-zA-Z_0-9].*\\.root$")

// PatternRun represents CHESS run
var PatternRun = regexp.MustCompile("[0-9]+")
