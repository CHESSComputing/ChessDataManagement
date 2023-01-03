package main

import (
	"fmt"
	"testing"
)

// TestQuery1
func TestQuery1(t *testing.T) {
	user := "test"
	query := fmt.Sprintf("user:%s", user)
	res := ParseQuery(query)
	if res["user"] != user {
		t.Error("Fail TestQuery user parsing")
	}
}

// TestQuery2
func TestQuery2(t *testing.T) {
	user := "test"
	attr := "bla foo"
	query := fmt.Sprintf("user:%s attr:%s", user, attr)
	res := ParseQuery(query)
	if res["user"] != user {
		t.Error("Fail TestQuery user parsing")
	}
	if res["attr"] != attr {
		t.Error("Fail TestQuery attr parsing")
	}
}

// TestQuery3
func TestQuery3(t *testing.T) {
	user := "test"
	attr := "bla foo"
	keys := "v1 v2"
	query := fmt.Sprintf("user:%s attr:%s keys:%s", user, attr, keys)
	res := ParseQuery(query)
	if res["user"] != user {
		t.Error("Fail TestQuery user parsing")
	}
	if res["attr"] != attr {
		t.Error("Fail TestQuery attr parsing")
	}
	if res["keys"] != keys {
		t.Error("Fail TestQuery keys parsing")
	}
}
