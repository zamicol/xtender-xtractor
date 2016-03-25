ApplicationXtender File Extraction Tool
=================
# Dump File #
Flat File should be ordered by object ID *in ascending order*.  If list is not sorted and not in ascending order, you will get duplicates if duplicates exit.  If there are duplicate object ID's in the sorted list, it will skip the duplicates.  


# File Path Calculation #

Path Calculation Equation: (ObjectId / FolderSize^(DirDepth)) % FolderSize

## Flat file ##
Flat file MUST HAVE object ID colum.  This column is used to determine file path.  

The Object ID column MUST BE sorted in asscending order.  


The way this is calculated:

```
Parent folder = round down((File ID % 1024^(folder depth)) / 1024)
Folder = round down((File ID / 1024^(this folder depth)) )
```

### Example ###
```
Document ID: 2927782 
Parent Directory: X:\whatever\
Folder depth: 2
```
### Calculation ###
Given these parameters:

DirDepth = 2
FolderSize = 1024
ObjectId = 2927782


Parent Folder = round down((ObjectId / FolderSize^(DirDepth)) % FolderSize) = 811
1048576
Direct Folder = round down((ObjectId / FolderSize^(DirDepth)) % FolderSize) = 2

Then we concat the components to give the full path:`X:\whatever\2\811\2927782`


Another Example:
DirDepth = 2
FolderSize = 1024
ObjectId = 7957574

(7957574÷1024^2) % 1024 = 7
Broken down: 
(7957574÷1024^1) % 1024 = 603

Path = \7\603\7957574.bin














