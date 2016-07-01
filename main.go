package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	lineCount  int //Increment for each line
	successful int //Sucessfully copied files
	failed     int //Failed to copy count.
	duplicates int //duplicates found in the sort file

	skipped    int             //offset rows (future other skips)
	logFile    *os.File        //Log file
	outDups    *os.File        //Copy over any lines that are skipped due to being duplicated object id's
	errorLines *os.File        //Error lines
	configFile = "config.json" //Configuration file.
)

//main opens and parses the config, Starts logging, and then call setup
func main() {
	//Load Config
	c := new(Configuration)
	c.Parse(configFile)
	//Process file
	setup(c)
}

//Setup
//Creates output files and directories and calls processIndex
func setup(c *Configuration) {
	var err error
	//zip source folder?
	//Zipping should be last.  defer is lifo
	if c.OutZipped {
		defer zip(c.OutDir, c.OutZippedDeleteSource)
	}

	//Create out dir if not exist, only one deep
	Mkdir(c.OutDir)

	//Logging
	InitLog(c)
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

		//If you wanted to copy the offset lines somewhere
		//This would be the place to do it.
		//writeLine(scanner.Text(), outFlat)
		lineCount++
		skipped++
	}

	//New line object.
	line := &Line{
		Configuration: c,
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
		line.Columns = col
		line.Line = l

		line.ProcessLine()
	}

	//Print to the log the last out ObjectID
	log.Println("Last ObjectID in:", line.ID)
	log.Println("Last ObjectID out:", successful+line.OutFileRenameIntOffset-1)

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
