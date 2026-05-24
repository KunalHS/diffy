package main

import "fmt"

type CommandKind string

const (
	KindLocal   CommandKind = "local"
	KindCompare CommandKind = "compare"
	KindAhead   CommandKind = "ahead"
	KindBehind  CommandKind = "behind"
	KindRecent  CommandKind = "recent"
	KindFile    CommandKind = "file"
	KindStash   CommandKind = "stash"
	KindHistory CommandKind = "history"
	KindCommit  CommandKind = "commit"
)

type Layout string

const (
	LayoutSplit   Layout = "split"
	LayoutUnified Layout = "unified"
)

type Options struct {
	Help        bool
	Summary     bool
	Raw         bool
	Layout      Layout
	FileFlag    bool
	FilePath    string
	LocalMode   string
	UsedFile    bool
	UsedPath    bool
	UsedRaw     bool
	UsedNoTUI   bool
	UsedUnified bool
	UsedSplit   bool
}

type Command struct {
	Kind        CommandKind
	Options     Options
	Source      string
	Target      string
	Count       int
	Path        string
	Ref         string
	FromRef     string
	ToRef       string
	StashRef    string
	Commit      string
	Interactive bool
	Alias       string
}

type Config struct {
	Layout Layout
	Path   string
}

type FileStat struct {
	Path     string
	Add      int
	Del      int
	Status   string
	Binary   bool
	Selected bool
}

func (f FileStat) String() string {
	if f.Status != "" {
		return fmt.Sprintf("%s  %s", f.Status, f.Path)
	}
	return f.Path
}

type CommitItem struct {
	Hash    string
	Subject string
	Raw     string
}

type StashItem struct {
	Ref     string
	Subject string
	Raw     string
}

type ViewData struct {
	Title           string
	Subtitle        string
	GitCommand      string
	Files           []FileStat
	Commits         []CommitItem
	Stashes         []StashItem
	Diff            string
	Mode            CommandKind
	RestrictedPath  string
	SourceIsCurrent bool
	Message         string
}

func knownCommand(s string) bool {
	switch s {
	case "local", "compare", "review", "ahead", "behind", "recent", "file", "stash", "history", "commit":
		return true
	default:
		return false
	}
}
