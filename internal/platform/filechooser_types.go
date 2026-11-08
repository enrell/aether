package platform

// FileFilter is one entry in a file chooser's filter list.
type FileFilter struct {
	Name     string
	Patterns []string
}

// FileChooserOptions configures PortalOpenFile.
type FileChooserOptions struct {
	Title         string
	Directory     bool
	CurrentFolder string
	Filters       []FileFilter
}
