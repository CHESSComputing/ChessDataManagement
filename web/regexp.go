package main

import (
	"regexp"
)

// PatternInt represents an integer pattern
var PatternInt = regexp.MustCompile("(^[0-9-]$|^[0-9-][0-9]*$)")

// PatternSite represents a site name pattern
var PatternSite = regexp.MustCompile("^T[0-9]_[A-Z]+(_)[A-Z]+")

// PatternSE represents StorageElement pattern
var PatternSE = regexp.MustCompile("^[a-z]+(\\.)[a-z]+(\\.)")

// PatternUrl represents URL pattern
var PatternUrl = regexp.MustCompile("(https|http)://[-A-Za-z0-9_+&@#/%?=~_|!:,.;]*[-A-Za-z0-9+&@#/%=~_|]")

// PatternDataset represents CMS dataset
var PatternDataset = regexp.MustCompile("/[-a-zA-Z_0-9*]+/[-a-zA-Z_0-9*]+/[-a-zA-Z_0-9*]+")

// PatternBlock represents CMS block
var PatternBlock = regexp.MustCompile("/[-a-zA-Z_0-9*]+/[-a-zA-Z_0-9*]+/[-a-zA-Z_0-9*]+#.*")

// PatternFile represents CMS file
var PatternFile = regexp.MustCompile("/[a-zA-Z_0-9].*\\.root$")

// PatternRun represents CMS run
var PatternRun = regexp.MustCompile("[0-9]+")
