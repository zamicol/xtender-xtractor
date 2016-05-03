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
	flatOut    *os.File                 //"index" file out.  Append existing rows plus new file path.
	errorOut   *os.File                 //Error lines
	configFile          = "config.json" //Configuration file.

	last     string = "" //Remeber last object ID processed.  Prevents duplicates.
	lastPath        = "" //Path of the last object.
)

type Configuration struct {
	OutDir             string
	InDir              string
	FlatFileIn         string
	FlatFileOut        string
	FlatFileErrorLines string
	Log                string
	InFileExt          string
	OutFileExt         string
	OutFileNameInt     bool
	DirDepth           int
	FolderSize         int
	Delimiter          string
	CountOffset        int //Offset used to start incrementing at another number
	RowOffset          int //Rows that should be ignored before processing index rows.  Usefull for headers.  Will be copied to output.
	ColObjectID        int
	ColFileName        int
	ColFileExt         int
	ColFileExtOut      int
}

//main opens and parses the config, Starts logging, and then call setup
func main() {
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

	//logging
	initLog(c)
	defer stopLog()

	setup(c)

}

//Setup
//Creates output files and directories and calls processIndex
func setup(c *Configuration) {
	//Create out dir if not exist, only one deep
	_, e := os.Stat(c.OutDir)
	if os.IsNotExist(e) {
		fmt.Println("Out directory does not exist.", c.OutDir)
		e := os.Mkdir(c.OutDir, 0777)
		if e == nil {
			fmt.Println("Created output directory: ", c.OutDir)
		} else {
			fmt.Println("Unable to create output directory: ", c.OutDir)
		}
	} else {
		fmt.Println("Out directory exists.", c.OutDir)
	}

	var err error

	flatOut, err = os.OpenFile(c.FlatFileOut, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}

	errorOut, err = os.Create(c.FlatFileErrorLines)
	if err != nil {
		log.Fatal(err)
	}
	defer flatOut.Close()

	//Process file
	processIndex(c.FlatFileIn, c)
}

//processIndex processes flat file dump file line by line.
func processIndex(flat string, c *Configuration) {
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
		writeLine(scanner.Text(), flatOut)
		lineCount++
		skipped++
	}

	//For all rows we want to process
	for scanner.Scan() {
		//Get the line to be processed.
		var l = scanner.Text()
		//Increment the counter and process the line
		lineCount++
		processLine(&l, c)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

//processLine copies file to output.
func processLine(line *string, c *Configuration) {

	columns := strings.Split(*line, c.Delimiter)

	//Check to see if object was already processed.
	if last == columns[c.ColObjectID] {
		log.Println("Skipping duplicate.", columns[c.ColObjectID])
		writeLine(*line+c.Delimiter+lastPath, flatOut)
		duplicates++
		return
	} else {
		last = columns[c.ColObjectID]
	}

	//Get parent path for in file.
	var inExtension string
	if c.InFileExt == "" {
		inExtension = columns[c.ColFileExt]
	} else {
		inExtension = c.InFileExt
	}

	//Full file path in
	var subpath string
	subpath, err := getPathFromId(columns[c.ColObjectID], c)

	//Use a static path OR path from flat file dump row
	var inPath string
	if c.InDir != "" {
		inPath = c.InDir
	} else {
		inPath = columns[c.ColFileName]
	}

	fullpath := filepath.Join(inPath, subpath, columns[c.ColObjectID]) + inExtension
	if err != nil {
		log.Println("Unable to process line: ", lineCount, err.Error())
	}

	//full file path out.
	var filename string
	if c.OutFileNameInt == true {
		//Add the offset to the number of successful.
		fileIntName := c.CountOffset + successful
		filename = strconv.Itoa(fileIntName)
	} else {
		filename = columns[c.ColFileName]
	}

	//Extension for out file.
	//Use static extension.  If blank, assume this value is provided in row's column
	var outExtension string
	if c.OutFileExt == "" {
		outExtension = columns[c.ColFileExtOut]
	} else {
		outExtension = c.OutFileExt
	}

	//Create fulle path for out file
	out := filepath.Join(c.OutDir, filename) + outExtension
	lastPath = out

	fmt.Println(fullpath, out, subpath, columns)

	//Copy file
	err = copy(fullpath, out)
	if err == nil {
		writeLine(*line+c.Delimiter+out, flatOut)
	} else {
		writeLine(*line, errorOut)
	}
}

//copy copies file from in to out.
func copy(in, out string) (e error) {
	i, err := os.Open(in)
	if err != nil {
		message := fmt.Sprint("File does not exist.", in)
		log.Println(message)
		fmt.Println(message)
		failed++
		return errors.New("File not found")
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
	exitMessage := fmt.Sprint("Process stopped. \nLines processed: ", lineCount,
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
