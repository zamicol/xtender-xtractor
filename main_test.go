package main

import (
	"fmt"
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

	for key, value := range s {
		p, err := getPathFromId(value, c)
		if err != nil {
			t.Errorf("getPathFromId error", err)
		}
		if key != p {
			t.Errorf("Calculation mismatch. Expected: " + key + ", Got:" + p)
		}
		fmt.Println("Path calculation: ", p)
	}
}
