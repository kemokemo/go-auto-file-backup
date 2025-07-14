$CurrentPath = $PWD.Path
New-Service -Name "GoAutoBackup" -BinaryPathName $CurrentPath\go-auto-file-backup.exe