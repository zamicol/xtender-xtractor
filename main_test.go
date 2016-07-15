package main

import (
	"fmt"
	"strconv"
	"testing"
)

func TestPathCalc(t *testing.T) {
	//Golden case
	s := map[string]string{
		"2/811": "2927782",
		"7/603": "7957574",
	}

	c := new(Configuration)
	c.DirDepth = 2
	c.FolderSize = 1024

	l := new(Line)
	l.Configuration = c

	for key, value := range s {
		l.ID, _ = strconv.ParseInt(value, 10, 64)
		p, err := l.GetPathFromID()
		if err != nil {
			t.Error("GetPathFromID error", err)
		}
		if key != p {
			t.Errorf("Calculation mismatch. Expected: " + key + ", Got:" + p)
		}
		fmt.Println("Path calculation: ", p)
	}
}
