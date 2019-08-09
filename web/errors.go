package main

// errors module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//

// ServerError and others are represent different types of errors
const (
	_ = iota
	ServerError
	MongoDBError
	ProxyError
	QueryError
	ParserError
	ValidationError
)

// ServerErrorName and others provides human based definition of the error
const (
	ServerErrorName     = "Server error"
	MongoDBErrorName    = "MongoDB error"
	ProxyErrorName      = "Server proxy error"
	QueryErrorName      = "Server query error"
	ParserErrorName     = "Server parser error"
	ValidationErrorName = "Server validation error"
)
