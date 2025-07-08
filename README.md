# go-auto-file-backup

This tool detects file creation and file modification events in specific multiple directories and creates a directory for each date and time in another directory for backup.

`.DS_Store`, `Thumbs.db`, etc., can be specified from an external `config.yaml`, including filename patterns for files you wish to exclude from specific backup targets.
