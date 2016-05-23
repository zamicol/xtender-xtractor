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
	"runtime"
	"strconv"
	"strings"
)

var (
	lineCount  int //Increment for each line
	successful int //Sucessfully copied files
	failed     int //Failed to copy count.
	duplicates int //duplicates found in the sort file

	skipped    int             //offset rows (future other skips)
	logFile    *os.File        //Log file
	outFlat    *os.File        //"index" file out.  Append existing rows plus new file path.
	outDups    *os.File        //Copy over any lines that are skipped due to being duplicated object id's
	errorLines *os.File        //Error lines
	configFile = "config.json" //Configuration file.

	last int //Remeber last object ID processed.  Prevents duplicates.
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
	//Lines
	OutLinesName          string
	OutLinesErrorName     string
	OutLog                string
	OutLinesDuplicateName string
	OutLinesColomns       string
	OutLinesRowOffset     int
	//Files
	OutFileExt             string
	OutFileRenameInt       bool
	OutFileRenameIntOffset int
	OutXtenderStructure    bool
	//AutoBatch
	OutAutoBatch        bool
	OutAutoBatchCount   int
	OutAutoBatchName    string
	OutAutoBatchZeroPad int

	//Global

	DirDepth        int
	FolderSize      int
	Delimiter       string //Applies to both in and out.
	ComputeChecksum bool

	//Columns
	ColObjectID   int
	ColFileName   int
	ColFileExtIn  int
	ColFileExtOut int
}

//Line represents line contents and settings.
type Line struct {
	*Configuration
	Line    string   //String of the line
	Columns []string //Parsed columns
	ID      int      //uniqueobject ID.  Used for path calculation.
	Dir     string   //Directory of file
	Path    string   //Full path including file
}

//main opens and parses the config, Starts logging, and then call setup
func main() {
	//Load Config
	c := Config()

	//Process file
	setup(c)
}

//Config parses config file into memory
func Config() *Configuration {
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
	c := new(Configuration)
	err = decoder.Decode(c)
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
	Mkdir(c.OutDir)

	//Logging
	initLog(c)
	defer stopLog()

	//Duplicate lines file
	od := filepath.Join(c.OutDir, c.OutLinesDuplicateName)
	outDups, err = os.OpenFile(od, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer outDups.Close()

	//Error lines file
	oe := filepath.Join(c.OutDir, c.OutLinesErrorName)
	errorLines, err = os.OpenFile(oe, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer errorLines.Close()

	//Process file
	processIn(c.InFlatFile, c)
}

//processIndex processes flat file dump file line by line.
func processIn(flat string, c *Configuration) {
	//Open input file
	file, err := os.Open(flat)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	//Special case for RowOffset rows.
	for i := c.OutLinesRowOffset; i > 0; i-- {
		//Get the next line using the Scan() method
		scanner.Scan()

		//We will copy the RowOffset lines to the out file.
		//TODO remove me
		//Maybe put lines in Log?
		//No we wont!
		//writeLine(scanner.Text(), outFlat)
		lineCount++
		skipped++
	}

	//For all rows we want to process
	for scanner.Scan() {
		//Get the line to be processed.
		var l = scanner.Text()
		//Increment the counter and process the line
		lineCount++
		fmt.Println("Line: ", lineCount)

		//Construct our line type
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

//ProcessLine copies file to output.
//Returns false in the event of error
func (l *Line) ProcessLine() (b bool) {
	var err error

	//TODO create out lines
	l.OutLineFile()

	//Get object ID from column
	l.ID, err = strconv.Atoi(l.Columns[l.ColObjectID])
	if err != nil {
		return errorLine(&l.Line, err)
	}

	//Check to see if object was already processed.
	//We do this by comparing objectID, which represents a unique object
	if last == l.ID {
		duplicates++
		log.Println("Skipping duplicate.", l.ID)
		writeLine(l.Line, outDups)
		return false
	}

	//Remember this line to compare with the next line to check for duplicates
	last = l.ID

	//Get full path for file in
	l.Path, err = l.GetInPath()
	if err != nil {
		return errorLine(&l.Line, err)
	}

	//Create new Line for line out.  Copy values from line in.
	lo := *l
	//What object out are we on?  Should be successful plus offset
	current := successful + l.OutFileRenameIntOffset
	lo.ID = current

	//Get full path for file out.
	lo.Path, err = lo.GetOutPath()
	if err != nil {
		//Write line in to error line.
		return errorLine(&l.Line, err)
	}

	//fmt.Println("inFullPath:", l.Path, "outFullPath:", lo.Path)
	//Create path for out.
	MkdirAll(lo.Dir)

	//Copy file
	err = copy(l.Path, lo.Path)
	//Write line to out file if successful
	if err != nil {
		return errorLine(&l.Line, err)
	}

	//Columns
	err = lo.GenLineFromColumns()
	if err != nil {
		return errorLine(&l.Line, err)
	}
	writeLine(lo.Line+lo.Delimiter+lo.Path, outFlat)

	return true
}

//GetInPath Get full path for file in
func (l *Line) GetInPath() (fullPath string, err error) {
	//Get extension for in file.
	var inExtension string
	if l.InFileExt == "" {
		inExtension = l.Columns[l.ColFileExtIn]
	} else {
		inExtension = l.InFileExt
	}

	//Use a static path OR path from flat file dump row
	var parentPath string
	if l.InDir != "" {
		parentPath = l.InDir
	} else {
		parentPath = l.Columns[l.ColFileName]
	}

	var subpath string
	subpath, err = l.GetPathFromID()
	if err != nil {
		return "", err
	}

	var fileName = strconv.Itoa(l.ID)

	//Full path for file in.
	fullPath = filepath.Join(parentPath, subpath, fileName) + inExtension
	return fullPath, nil
}

//GetOutPath Get full path for out file
func (l *Line) GetOutPath() (out string, err error) {
	//full file path out.
	var filename string

	if l.OutFileRenameInt == true {
		//Use ID as file name
		filename = strconv.Itoa(l.ID)
	} else {
		//Get the file name from the in file
		filename = l.Columns[l.ColFileName]
	}

	//Extension for out file.
	//Use static extension.  If blank, assume this value is provided in row's column
	outExtension := l.OutFileExt
	if outExtension == "" {
		outExtension = l.Columns[l.ColFileExtOut]
	}

	//Are we batching?  If so, add batch name to path.
	if l.OutAutoBatch == true {
		l.Dir = l.GetBatch()
	}

	//Create full path for out file
	//Create parent path first.
	if l.OutXtenderStructure == true {
		var subpath string
		subpath, err = l.GetPathFromID()
		if err != nil {
			return "", err
		}

		//Add the subpath to the current dir.
		l.Dir = filepath.Join(l.Dir, subpath)
	}

	//Add the parent Out directory.  This give the final directory.
	l.Dir = filepath.Join(l.OutDir, l.Dir)

	//Full path including file name.
	l.Path = filepath.Join(l.Dir, filename) + outExtension

	return l.Path, nil
}

//OutLineFile Create the "OutLines" file.
//Should call defer os.Close
func (l *Line) OutLineFile() {
	outFile := l.GetBatch() + "_" + l.OutLinesName
	outPath := filepath.Join(l.OutDir, outFile)
	var err error
	outFlat, err = os.OpenFile(outPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}
	//return outFlat
}

//GetBatch Return the batch name, including incrementer
func (l *Line) GetBatch() string {
	//If batch is false, there is no batch.  Return blank string
	if l.OutAutoBatch == false {
		return ""
	}

	var batchCount int
	batchCount = l.ID / l.OutAutoBatchCount
	//bcn := strconv.Itoa(batchCount)

	//Zero pad our batch
	pad := strconv.Itoa(l.OutAutoBatchZeroPad)
	bcn := fmt.Sprintf("%0"+pad+"d", batchCount)

	return l.OutAutoBatchName + bcn
}

//GenLineFromColumns Instead of copying the line from flat in, grab only some columns and write those to out.
func (l *Line) GenLineFromColumns() (err error) {

	var line string
	cols := strings.Split(l.OutLinesColomns, ",")
	for _, v := range cols {
		i, err := strconv.Atoi(v)
		if err != nil {
			return err
		}

		//Fencepost
		if line == "" {
			line += l.Columns[i]
		} else {
			line += l.Delimiter + l.Columns[i]
		}
	}

	l.Line = line

	return nil
}

//GetPathFromID Calculate the ApplicationXtender from a given object id, s
func (l *Line) GetPathFromID() (p string, e error) {
	//For each step of the path, we will calculate that portion of the path
	for i := l.DirDepth; i > 0; i-- {
		//Get maximum of how many objects there could be at this level.
		powered := math.Pow(float64(l.FolderSize), float64(i))
		//divide ID by max objects, and then Mod the id by how many objects per folder.
		subpath := int(math.Mod((float64(l.ID) / powered), float64(l.FolderSize)))
		//Add this portion of the path to the directory string.  Iterate next level until done.
		p = filepath.Join(p, strconv.Itoa(subpath))
		//fmt.Println("Path:", p, "Id value:", id, "Subpath:", subpath, "I:", i, "Powered: ", powered)
	}
	return p, e
}

//copy copies file from in to out.
func copy(in, out string) (err error) {
	i, err := os.Open(in)
	if err != nil {
		failed++
		message := fmt.Sprint("File does not exist. ", in)
		fmt.Println(message)
		return errors.New(message)
	}
	defer i.Close()

	o, err := os.Create(out)
	if err != nil {
		log.Println("Cannot create file out.  Stopping execution", out)
		panic(err)
	}
	defer o.Close()

	_, err = io.Copy(o, i)
	if err != nil {
		log.Println("Copying failed.  Stopping execution", out)
		panic(err)
	} else {
		successful++
	}

	return nil
}

//Mkdir Create out dir if not exist, only one deep
func Mkdir(dir string) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		fmt.Println("Out directory does not exist.", dir)
		err := os.Mkdir(dir, 0777)
		if err == nil {
			fmt.Println("Created output directory: ", dir)
		} else {
			message := fmt.Sprint("Unable to create output directory.  Stopping execution ", dir, err)
			panic(message)
		}
	} else {
		fmt.Println("Out directory exists.", dir)
	}
}

//MkdirAll Create out dir if not exist recursivly
func MkdirAll(dir string) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			message := fmt.Sprint("Unable to create output directory.  Stopping execution ", dir, err)
			panic(message)
		}
	}
}

//initLog Opens or creates log file, set log output.
func initLog(c *Configuration) {
	fmt.Println(runtime.GOOS)

	ol := filepath.Join(c.OutDir, c.OutLog)
	f, err := os.OpenFile(ol, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		panic("error opening file log file.")
	} else {
		fmt.Println("Created and opened log:", c.OutLog)
	}

	log.SetOutput(f)
	logFile = f
	log.Println("Started process.")
	m := fmt.Sprintf("Configuration: %+v\n", *c)
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
