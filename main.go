package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	lineCount  int = 0 //Increment for each line
	successful int = 0 //Sucessfully copied files
	failed     int = 0 //Failed to copy count.
	duplicates int = 0 //duplicates found in the sort file

	skipped    int      = 0             //offset rows (future other skips)
	logFile    *os.File                 //Log file
	outFlat    *os.File                 //"index" file out.  Append existing rows plus new file path.
	outDups    *os.File                 //Copy over any lines that are skipped due to being duplicated object id's
	errorLines *os.File                 //Error lines
	configFile          = "config.json" //Configuration file.

	last     string = "" //Remeber last object ID processed.  Prevents duplicates.
	lastPath        = "" //Path of the last object.
)

type Configuration struct {
	//See README for description of each variable.
	InFlatFile string
	InDir      string
	InFileExt  string

	OutFlatFile         string
	OutDir              string
	OutErrorLines       string
	OutDuplicateLines   string
	OutFileExt          string
	OutFileRenameInt    bool
	OutColomns          []interface{}
	OutCountOffset      int
	OutXtenderStructure bool
	OutAutoBatch        bool
	OutAutoBatchCount   int
	OutAutoBatchName    string

	Log             string
	RowOffset       int //Process ignorded rows.  Usefull for headers.  Will be copied to output.
	DirDepth        int
	FolderSize      int
	Delimiter       string //Applies to both in and out.
	ComputeChecksum bool

	ColObjectID   int
	ColFileName   int
	ColFileExtIn  int
	ColFileExtOut int
}

type Line struct {
	*Configuration
	Line    string
	Columns []string
}

//main opens and parses the config, Starts logging, and then call setup
func main() {
	//Load Config
	c := Config()

	//Logging
	initLog(c)
	defer stopLog()

	//Process file
	setup(c)
}

func Config() *Configuration {
	//Open config
	file, _ := os.Open(configFile)
	defer file.Close()
	//Config is json.
	decoder := json.NewDecoder(file)
	c := new(Configuration)
	err := decoder.Decode(c)
	if err != nil {
		panic("Unable to process config file. " + configFile +
			". This probably means that the file isn't valid json." + err.Error())
	}
	return c
}

//Setup
//Creates output files and directories and calls processIndex
func setup(c *Configuration) {
	var err error

	//Create out dir if not exist, only one deep
	_, e := os.Stat(c.OutDir)
	if os.IsNotExist(e) {
		fmt.Println("Out directory does not exist.", c.OutDir)
		e := os.Mkdir(c.OutDir, 0777)
		if e == nil {
			fmt.Println("Created output directory: ", c.OutDir)
		} else {
			fmt.Println("Unable to create output directory: ", c.OutDir, err)
			panic(err)
		}
	} else {
		fmt.Println("Out directory exists.", c.OutDir)
	}

	outFlat, err = os.OpenFile(c.OutFlatFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer outFlat.Close()

	outDups, err = os.OpenFile(c.OutDuplicateLines, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer outDups.Close()

	errorLines, err = os.OpenFile(c.OutErrorLines, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer errorLines.Close()

	//Process file
	processIn(c.InFlatFile, c)
}

//processIndex processes flat file dump file line by line.
func processIn(flat string, c *Configuration) {
	file, err := os.Open(flat)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	//Special case for RowOffset rows.
	for i := c.RowOffset; i > 0; i-- {
		//Get the nex line using the Scan() method
		scanner.Scan()

		//We will copy the RowOffset lines to the out file.
		writeLine(scanner.Text(), outFlat)
		lineCount++
		skipped++
	}

	//For all rows we want to process
	for scanner.Scan() {
		//Get the line to be processed.
		var l = scanner.Text()
		//Increment the counter and process the line
		lineCount++

		//Get the columns in the line
		col := strings.Split(l, c.Delimiter)

		//New line object.
		line := &Line{
			Configuration: c,
			Line:          l,
			Columns:       col,
		}

		line.ProcessLine()

	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

//processLine copies file to output.
//Returns false in the event of error
func (l *Line) ProcessLine() (b bool) {
	var err error

	//Check to see if object was already processed.
	if last == l.Columns[l.ColObjectID] {
		log.Println("Skipping duplicate.", l.Columns[l.ColObjectID])
		//Write this line to duplicates
		writeLine(l.Line, outDups)
		duplicates++
		return false
	} else {
		//Remember this line to compare with the next line to check for duplicates
		last = l.Columns[l.ColObjectID]
	}

	//Get extension for in file.
	var inExtension string
	if l.InFileExt == "" {
		inExtension = l.Columns[l.ColFileExtIn]
	} else {
		inExtension = l.InFileExt
	}

	//Full file path for in file
	var subpath string
	subpath = l.GetOutPath()
	//	if err != nil {
	//		return errorLine(line, err)
	//	}

	//Use a static path OR path from flat file dump row
	var inPath string
	if l.InDir != "" {
		inPath = l.InDir
	} else {
		inPath = l.Columns[l.ColFileName]
	}

	//Full path for file in.
	inFullPath := filepath.Join(inPath, subpath, l.Columns[l.ColObjectID]) + inExtension

	//Get full path for file out.
	outFullPath := l.GetOutPath()

	fmt.Println(inFullPath, outFullPath, subpath, l.Columns)

	//Copy file
	err = copy(inFullPath, outFullPath)
	//Write line to out file if successful
	if err == nil {
		writeLine(l.Line+l.Delimiter+outFullPath, outFlat)
	} else {
		return errorLine(&l.Line, err)
	}

	return true
}

func (l *Line) GetOutPath() string {
	//full file path out.
	var filename string
	if l.OutFileRenameInt == true {
		//Add the offset to the number of successful.  This gives use the current file name
		current := l.OutCountOffset + successful
		filename = strconv.Itoa(current)
	} else {
		filename = l.Columns[l.ColFileName]
	}

	//Extension for out file.
	//Use static extension.  If blank, assume this value is provided in row's column
	var outExtension string
	if l.OutFileExt == "" {
		outExtension = l.Columns[l.ColFileExtOut]
	} else {
		outExtension = l.OutFileExt
	}

	//Create fulle path for out file
	out := filepath.Join(l.OutDir, filename) + outExtension

	//TODO document me.
	lastPath = out

	return out
}

//copy copies file from in to out.
func copy(in, out string) (e error) {
	i, err := os.Open(in)
	if err != nil {
		message := fmt.Sprint("File does not exist. ", in)
		fmt.Println(message)
		failed++
		return errors.New(message)
	}

	o, err := os.Create(out)
	if err != nil {
		log.Println("Cannot create file out.  Stopping execution", out)
		panic(err)
	}
	defer i.Close()
	defer o.Close()

	w, err := io.Copy(o, i)
	if err != nil {
		log.Println("Copying failed.  Stopping execution", out)
		panic(err)
	} else {
		successful++
	}

	fmt.Println(w)
	return nil
}

//Calculate the ApplicationXtender from a given object id, s string
//
func getPathFromId(s string, c *Configuration) (p string, e error) {
	id, e := strconv.Atoi(s)

	//For each step of the path, we will calculate that portion of the path
	for i := c.DirDepth; i > 0; i-- {
		//Get maximum of how many objects there could be at this level.
		powered := math.Pow(float64(c.FolderSize), float64(i))
		//divide ID by max objects, and then Mod the id by how many objects per folder.
		subpath := int(math.Mod((float64(id) / powered), float64(c.FolderSize)))
		//Add this portion of the path to the directory string.  Iterate next level until done.
		p = filepath.Join(p, strconv.Itoa(subpath))
		//fmt.Println("Path:", p, "Id value:", id, "Subpath:", subpath, "I:", i, "Powered: ", powered)
	}
	return p, e
}

//initLog Opens or creates log file, set log output.
func initLog(c *Configuration) {
	f, err := os.OpenFile(c.Log, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		panic("error opening file log file.")
	} else {
		fmt.Println("Created and opened log:", c.Log)
	}

	log.SetOutput(f)
	logFile = f
	log.Println("Started process.")
	m := fmt.Sprintf("Configuration: %+v\n", *c)
	fmt.Println(m)
	log.Println(m)
}

//stopLog closses the log file and prints the final exit message.
func stopLog() {
	exitMessage := fmt.Sprint(
		"Process stopped. \nLines processed: ", lineCount,
		"\nSkipped rows:", skipped,
		"\nSucessfully copied: ", successful,
		"\nDuplicates skipped: ", duplicates,
		"\nFailed:", failed)
	log.Println(exitMessage)
	fmt.Println(exitMessage)
	defer logFile.Close()
}

//writeLine writes given string to given file with a newline appended at the end.
func writeLine(s string, f *os.File) {
	f.WriteString(s + "\n")
}

//errorLine
//helper function for when encountering error lines.  Logs the line number with error and and writes the line to the error line file.
func errorLine(line *string, err error) (b bool) {
	writeLine(*line, errorLines)
	log.Println("Line:", lineCount, err)
	return false
}
