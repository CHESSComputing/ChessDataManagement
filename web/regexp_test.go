package main

import "testing"

// TestRegexp
func TestRegexp(t *testing.T) {
	v := "1"
	if PatternInt.MatchString(v) != true {
		t.Errorf("Unable to match '%s' as int", v)
	}
	o := "bla"
	if PatternInt.MatchString(o) == true {
		t.Errorf("match '%s' as int", o)
	}

	v = "1.1"
	if PatternFloat.MatchString(v) != true {
		t.Errorf("Unable to match '%s' as float", v)
	}
	if PatternFloat.MatchString(o) == true {
		t.Errorf("match '%s' as float", v)
	}

	v = "http://abc.com"
	if PatternURL.MatchString(v) != true {
		t.Errorf("Unable to match '%s' as url", v)
	}
	if PatternURL.MatchString(o) == true {
		t.Errorf("match '%s' as url", v)
	}

	v = "/a/b/c"
	if PatternDataset.MatchString(v) != true {
		t.Errorf("Unable to match '%s' as dataset", v)
	}
	if PatternDataset.MatchString(o) == true {
		t.Errorf("match '%s' as dataset", v)
	}

	v = "/tmp/file.root"
	if PatternFile.MatchString(v) != true {
		t.Errorf("Unable to match '%s' as file", v)
	}
	if PatternFile.MatchString(o) == true {
		t.Errorf("match '%s' as file", v)
	}

	v = "123"
	if PatternRun.MatchString(v) != true {
		t.Errorf("Unable to match '%s' as run", v)
	}
	if PatternRun.MatchString(o) == true {
		t.Errorf("match '%s' as run", v)
	}
}
