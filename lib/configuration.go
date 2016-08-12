package lib

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

//Configuration stores settings from config.json
//And internally calculated configs
//See README for description of each variable.
type Configuration struct {
	////////////////////
	//Config variables from config file
	////////////////////
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
	OutLinesColumns       string
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

	////////////////////
	//Config variables **not** from config file
	////////////////////
	//Files
	logFile    *os.File //Log file
	outDups    *os.File //Lines that are duplicated object id's
	errorLines *os.File //Error lines
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

//Process
//Creates output files and directories and calls processIndex
func (c *Configuration) Process() {
	var err error
	//zip source folder?
	//Zipping should be last.  defer is lifo
	if c.OutZipped {
		defer zip(c.OutDir, c.OutZippedDeleteSource)
	}

	//Create out dir if not exist, only one deep
	Mkdir(c.OutDir)

	//Duplicate lines file
	od := filepath.Join(c.OutDir, c.OutLinesDuplicateName)
	c.outDups, err = os.OpenFile(od, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer c.outDups.Close()

	//Error lines file
	oe := filepath.Join(c.OutDir, c.OutLinesErrorName)
	c.errorLines, err = os.OpenFile(oe, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer c.errorLines.Close()

	//Process missing or in or out
	if c.Missing {
		missing(c)
	} else {
		//Process file
		c.processIn()
	}
}

//processIndex processes flat file dump file line by line.
func (c *Configuration) processIn() {
	//New line object.
	l := &Line{
		Configuration: c,
	}

	//Logging
	InitLog(c)
	defer stopLog(l)

	//Open input file
	file, err := os.Open(c.InFlatFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	//Special case for RowOffset rows.
	for i := c.OutLinesRowOffset; i > 0; i-- {
		//Get the next line using the Scan() method
		scanner.Scan()

		//If you wanted to copy the offset lines somewhere
		//This would be the place to do it.
		//writeLine(scanner.Text(), outFlat)
		l.lineCount++
		l.skipped++
	}

	//For all rows we want to process
	for scanner.Scan() {
		//Get the line to be processed.
		l.Line = scanner.Text()
		//Increment the counter and process the line
		l.lineCount++
		fmt.Println("Line: ", l.lineCount)

		//Construct our line type
		//Get the columns in the line
		col := strings.Split(l.Line, c.Delimiter)
		l.Columns = col

		l.ProcessLine()
	}

	//Print to the log the last out ObjectID
	log.Println("Last ObjectID in:", l.ID)
	log.Println("Last ObjectID out:", l.successful+l.OutFileRenameIntOffset-1)

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
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
