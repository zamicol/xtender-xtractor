package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//Calculate files that appear in the SQL database but do not appear
//Format for missing sql dump:
//Column 0: Object ID
//Column 1: Path to root of application
//Column 2: Bucket (Probably going to be application name)
func missing(c *Configuration) error {

	//Open missing in flat file
	file, err := os.Open(c.MissingIn)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	//New line object.
	l := &Line{
		Configuration: c,
	}

	defer l.File.Close()
	//Keep track of previous bucket
	var bucket string

	//For all rows we want to process
	for scanner.Scan() {
		//Get the line to be processed.
		var lineText = scanner.Text()
		//Increment the counter and process the line
		lineCount++

		//Construct our line type
		//Get the columns in the line
		col := strings.Split(lineText, c.Delimiter)

		//Set the Line to line text
		l.Line = lineText
		l.Columns = col
		//We should always have at least columns
		if len(col) >= 3 {
			failed++
			log.Println("Line is not 3 columns")
		}
		//In directory should be column 2
		l.InDir = col[1]
		var err error
		l.ID, err = strconv.ParseInt(col[0], 10, 64)
		if err != nil {
			log.Println(err)
			failed++
		}

		//Open the text file for bucket
		//If new bucket, close old file, open new one.
		if bucket != col[2] {
			bucket = col[2]
			l.File.Close()

			od := filepath.Join(c.OutDir, bucket+"_missing.txt")

			l.File, err = os.OpenFile(od, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
			if err != nil {
				log.Println(err)
			}
		}

		l.GetInPath()

		fmt.Println(lineCount, l.Path)

		calcMissing(l)

	}

	fmt.Println("Total Missing:", l.MissingCount)
	return nil
}

func calcMissing(l *Line) {
	//Column 2 should be path.
	//Col 3 is "bucket", could be Application in AppXtender
	if _, err := os.Stat(l.Path); os.IsNotExist(err) {
		fmt.Println("File does not exist.")
		writeLine(l.Path, l.File)
		l.MissingCount++
	}

}
