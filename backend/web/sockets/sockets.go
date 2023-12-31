package sockets

type Message string
type ErrorType string

const (
	// Messages sent by the client
	Preliminary Message = "preliminary"
	Process     Message = "process"
	ToggleLink  Message = "toggle_link"
	Package     Message = "package"
	GetDownload Message = "get_download"
	Delete      Message = "delete"

	// Messages sent to the client
	Connected        Message = "connected"
	PreliminaryStart Message = "preliminary_start"
	PreliminaryStep  Message = "preliminary_step"
	PreliminaryDone  Message = "preliminary_done"
	ProcessStart     Message = "process_start"
	ProcessStep      Message = "process_step"
	ProcessDone      Message = "process_done"
	PackageStart     Message = "package_start"
	PackageDone      Message = "package_done"
	GetDownloadStart Message = "get_download_start"
	GetDownloadDone  Message = "get_download_done"
	GetDownloadError Message = "get_download_error"
	Deleted          Message = "deleted"

	// Error types
	NoSuitableVersion ErrorType = "no_suitable_version"
)
