# go-auto-file-backup

This tool detects file creation and file modification events in specific multiple directories and creates a directory for each date and time in another directory for backup.

## Setup

Please edit the `config.yaml` to backup your files.

|Parameter name|Descriptions|
|:--|:--|
|`backup_base`| The directory to which the files are to be copied. The tool creates a date/time directory under this directory and copies files that have been changed.|
|`watch_dirs`|Specifies the directory to be monitored for file changes. Multiple directories can be monitored.|
|`ignore_patterns`|Specify a filename pattern for files to ignore, such as `Thumbs.db` on Windows or `.DS_Store` file on macOS.|

Once the above settings are completed, simply start the tool to begin monitoring the target directories (`watch_dirs`) and the backup process.

## For Windows users

You can install this tool as a Windows service. Please use the PowerShell script.  
The default configuration is to register the content as a service named `GoAutoBackup`, so rewrite and use it as needed.

```sh
.\install-as-windows-service.ps1
```

When no longer needed, the service registration can be deleted by a script as well.

```sh
.\uninstall-as-windows-service.ps1
```
