package main

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
	ServerErrorName     = "DAS error"
	MongoDBErrorName    = "MongoDB error"
	ProxyErrorName      = "DAS proxy error"
	QueryErrorName      = "DAS query error"
	ParserErrorName     = "DAS parser error"
	ValidationErrorName = "DAS validation error"
)
