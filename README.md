xtender-xtractor - ApplicationXtender File Extraction Tool
=================
# Introduction #
xtender-xtractor is used to copy files out manually from an ApplicationXtender environment.  
Depending on setup, Object ID's and their "docid"'s can be pulled from the 
dt and dl tables in ApplicationXtender.  This application is designed to be used in conjunction 
with a sql dump from ApplicationXtender tables.


# Config #
Config is stored in config.json.  Must be valid json.

## Config Values ##
* `InFlatFile` - *string* - In flat file to be processed.  Should be a delimited file.  Header rows can be offset with RowOffset.
* `OutFlatFile` - *string* - Output of running the process.  Basically appends copied file information to the end of each row processed from FlatFileIn.
* `OutErrorLines` - *string* - Lines that errored out will be placed here.  These lines are not processed.
* `OutDuplicateLines` - *string* - Lines that are duplicates are placed here.  These lines are not processed.
* `Log` - *string* - name of the log file.  The log give start and stop times, lines processed, and other summary details.
* `ComputeChecksum` - *boolean* - Compute checksums of input/output files.  Insures that modification hasn't happened and someone is trying to do something naughty.  
* `Delimiter` - *string* - delimiter for input and output flat file.  
* `RowOffset` - *int* - Used to skip header rows.  RowOffset rows will be output to FlatFileOut
* `OutDir` - *string* -  Path for output files.
* `OutDirXtenderStructure` - *boolean* - Put output files in a directory structure that mimics ApplicationXtender
* `AutoBatch` - *boolean* - Autobatch?  Will output files into batch folders first.  Might make migration more manageable.  The program will also look to see if the out directory already contains batches.  If it does, it will pick put where it left off.
* `AutoBatchCount` - *int* - How many files per batch?
* `AutoBatchName` - *string* -  Name of the batch directory.  Incrementer will be appended.  
* `InDir` - *string* - Starting path for input files.
* `InFileExt` - *string* - File extension for input files.  If empty string, ColFileExt will be used instead.  
* `OutFileExt` - *string* - Extension for output files.  If blank, ColFileExtOut will be used.
* `OutFileNameInt` - *boolean* - File will be named an incrementing integer.  If set to false, files will be named to ColFileName's value
* `CountOffset` - *int* - If file naming sequentially using OutFileNameInt, this is the offset
* `DirDepth` - *int* - How many directories deep?  (Usually 2 or 3)
* `FolderSize` - *int* - how many files per folder and folder per folder?  Typically, this value should be 1024
* `ColObjectID` - *int* - **Important**.  Object ID column number.  Used to calculate path.  
* `ColFileName` - *int* - Column name for file name.  Only used if OutFileNameInt is set to "false"
* `ColFileExtIn` - *int* - Specify in file extension column.  Only used if InFileExt is set empty ("").
* `ColFileExtOut` - *int* - Column name for file extension out.  Only used if OutFileExt is empty ("")


# Dump File #
Flat File should be ordered by object ID **in ascending order**.  If list is not sorted 
and not in ascending order, you will get duplicates if duplicates exist.  When sorted,  
duplicate object ID's are skipped.


# File Path Calculation #
Path Calculation Equation: (int)(ObjectId / FolderSize^(DirDepth)) % FolderSize)

See getPathFromId() for the exact logic.  


## Flat file ##
Flat file **must have** object ID colum (ColObjectID).  This column is used to determine file path.  

To avoid duplicates, the ColObjectID column MUST BE sorted in ascending order.  


# Examples #

#### Example One ####
```
Document ID: 2927782 
Parent Directory: X:\whatever\
Folder depth: 2
```
#### Calculation ####
Given these parameters:

DirDepth = 2
FolderSize = 1024 (default)
ObjectId = 2927782

Direct Folder = round down((ObjectId / FolderSize^(DirDepth (2 in this case))) % FolderSize) = 2
Parent Folder = round down((ObjectId / FolderSize^(DirDepth (1 in this case))) % FolderSize) = 811

Then we concat the components to give the full path:`X:\whatever\2\811\2927782`

#### Example Two ####
DirDepth = 2
FolderSize = 1024 (default)
ObjectId = 7957574

(7957574÷1024^2) % 1024 = 7
(7957574÷1024^1) % 1024 = 603

Add it all together:
Path = \7\603\7957574.bin













