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
#### In ####
* `InFlatFile` - *string* - In flat file to be processed.  Should be a delimited file.  Header rows can be offset with RowOffset.
* `InDir` - *string* - Starting path for input files.  The ApplicationXtender folder structure will be calculated after this point.
* `InFileExt` - *string* - File extension for input files.  ApplicationXtender should use ".bin" by default.  If empty string, ColFileExt will be used instead.  

#### Out ####
* `OutDir` - *string* -  Path for output file.

* **OutLines** If using batching, for `OutLinesName`, `OutLinesErrorName`, and `OutLinesDuplicateName` batch name will be prepended plus an underscore.  One of each file per batch.   
  * `OutLinesName` - *string* - Output of running the process.  Basically appends copied file information to the end of each row processed from InFlatFile.  This is the "index" file for copied files.
  * `OutLinesErrorName` - *string* - Lines that errored out will be placed here.  These lines are not processed.
  * `OutLinesDuplicateName` - *string* - Lines that are duplicates are placed here.  These lines are not processed.
* `OutLog` - *string* - name of the log file.  The log give start and stop times, lines processed, and other summary details.
* `OutLinesColomns` - *String* - Comma seperated columns to be copied to OutLinesName.  If blank, everything is copied. Does not apply to error or duplicate files.
* `	OutLinesRowOffset      int` - *int* - Used to skip header rows. Rows are discarded.  

* `OutFileExt` - *string* - Extension for output files.  If blank, ColFileExtOut will be used.
* `OutFileRenameInt` - *boolean* - File will be named an incrementing integer.  If set to false, files will be named to ColFileName's value.
* `OutFileRenameIntOffset` - *int* - If file naming sequentially using OutFileRenameInt, this is the offset. Typically should be 0.

* `OutDirXtenderStructure` - *boolean* - Put output files in a directory structure that mimics ApplicationXtender.
* `OutAutoBatch` - *boolean* - Autobatch?  Will output files into batch folders first.  Might make migration more manageable.  The program will also look to see if the out directory already contains batches.  If it does, it will pick put where it left off.
* `OutAutoBatchCount` - *int* - How many files per batch?
* `OutAutoBatchName` - *string* -  Name of the batch directory.  Incrementer will be appended.  

#### Global ####
* `DirDepth` - *int* - How many directories deep?  (Usually 2 or 3).
* `FolderSize` - *int* - how many files per folder and folder per folder?  Typically, this value should be 1024.
* `Delimiter` - *string* - delimiter for input and output flat file.  
* `ComputeChecksum` - *boolean* - Compute checksums of input/output files.  Insures that modification hasn't happened and someone is trying to do something naughty.  

#### Columns ####
* `ColObjectID` - *int* - **Important**.  Object ID column number.  Used to calculate path.  
* `ColFileName` - *int* - Column name for file in name.  Only used if OutFileNameInt is set to "false".
* `ColFileExtIn` - *int* - Specify in file extension column.  Only used if InFileExt is set empty ("").
* `ColFileExtOut` - *int* - Column name for file extension out.  Only used if OutFileExt is empty ("").

# Dump File #
Flat File should be ordered by object ID.  If list is not sorted, duplicates will be copied and overwrite existing object id's.  When sorted,  duplicate object ID's are skipped and their lines written to the duplicate file.

```
SELECT TOP 100000 
dl.[objectid]
,paths.path
,dt.[docid]
,dt.[field1]
,dt.[field2]
,dt.[field3]
,dt.[field4]
,dt.[field5]
,dt.[numobjects]
FROM [ae_dtX] dt
INNER JOIN [ae_dlX] dl
ON dt.docid = dl.docid
INNER JOIN [ae_paths] paths
```

Replace ae_dtX and ae_dlX with your appid (ie. ae_dt1 and ae_dl1 for an application with an appid of 1). Query the ae_apps table to determine the corresponding appid for your application.Also note that the amount of fields may change depending on the number of indexes for each application. Ex. if you have 3 index fields then you will only have field1, field2, and field3 (not field4 or field5).

## Flat file ##
Flat file **must have** object ID colum (ColObjectID).  This column is used to determine file path.  

To avoid duplicates, the ColObjectID column MUST BE sorted in ascending order.  

# File Path Calculation #
Path Calculation Equation: (int)(ObjectId / FolderSize^(DirDepth)) % FolderSize)

See getPathFromId() for the exact logic.  

## Examples ##

#### Example One ####
```
Document ID: 2927782 
Parent Directory: X:\whatever\
```
**Calculation** Given these parameters:
```
Folder depth: 2
FolderSize = 1024 (default)
ObjectId = 2927782
```

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













