ApplicationXtender File Extraction Tool
=================
xtender-xtractor is used to copy files out manually from ApplicationXtender.  

Depending on setup, indexes/Object ID's and thier "docid"'s can be pulled from the 
dt and dl tables in ApplicationXtender


# Dump File #
Flat File should be ordered by object ID *in ascending order*.  If list is not sorted 
and not in ascending order, you will get duplicates if duplicates exit.  If there are 
duplicate object ID's in the sorted list, those duplicates will be skipped.  


# File Path Calculation #
Path Calculation Equation: (int)(ObjectId / FolderSize^(DirDepth)) % FolderSize)

See getPathFromId() for the exact logic.  

## Flat file ##
Flat file MUST HAVE object ID colum.  This column is used to determine file path.  

The Object ID column MUST BE sorted in asscending order.  


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














