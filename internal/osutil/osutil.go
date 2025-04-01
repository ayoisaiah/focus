package osutil

const (
	Windows = "windows"
	Darwin  = "darwin"
)

type exitCode int

const (
	ExitOK    exitCode = 0
	ExitError exitCode = 1
)

const DirPermission = 0o755
