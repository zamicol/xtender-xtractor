package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

//Configuration stores settings from config.json
//See README for description of each variable.
type Configuration struct {
	//In
	InFlatFile string
	InDir      string
	InFileExt  string

	//Out
	OutDir string
	OutLog string
	//Lines
	OutLinesName          string
	OutLinesErrorName     string
	OutLinesColomns       string
	OutLinesRowOffset     int64
	OutLinesDuplicateName string
	//Files
	OutFileExt             string
	OutFileRenameInt       bool
	OutFileRenameIntOffset int64
	OutXtenderStructure    bool
	OutZipped              bool
	OutZippedDeleteSource  bool
	//AutoBatch
	OutAutoBatch        bool
	OutAutoBatchCount   int64
	OutAutoBatchName    string
	OutAutoBatchZeroPad int

	//Missing
	Missing      bool
	MissingIn    string
	MissingCount int64

	//Global
	//Applies to both in and out
	DirDepth        int
	FolderSize      int
	Delimiter       string
	ComputeChecksum bool

	//Columns
	ColObjectID   int
	ColFileName   int
	ColFileExtIn  int
	ColFileExtOut int
}

//Parse parses config file into memory
func (c *Configuration) Parse(configFile string) {
	var err error
	//Open config
	file, err := os.Open(configFile)
	if err != nil {
		panic("Unable to open config file: " + configFile +
			". This probably means it doesn't exist or the program doesn't have read permissions." + err.Error())
	}
	defer file.Close()
	//Config is json.
	decoder := json.NewDecoder(file)

	err = decoder.Decode(c)
	if err != nil {
		panic("Unable to process config file. " + configFile +
			". This probably means that the file isn't valid json." + err.Error())
	}

	//Sanitze values
	c.clean()
}

//clean sanitization for the config values
func (c *Configuration) clean() {
	//Get the OS
	fmt.Println(runtime.GOOS)
	//Fix path if Windows
	var err error
	c.OutDir, err = filepath.Abs(c.OutDir)
	if err != nil {
		panic("Unable to get absolute out directory path.")
	}
}
