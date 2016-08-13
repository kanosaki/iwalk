package main

import (
	"container/list"
	"fmt"
	"io"
	"os"
	"github.com/Sirupsen/logrus"
)

type IOEngine struct {
	actions *list.List
	trashActions *list.List
}

func NewIOEngine() *IOEngine {
	return &IOEngine{
		actions: list.New(),
	}
}

func (e *IOEngine) Commit() error {
	dryRun := *argDryRun
	for action := e.actions.Front(); action != nil; action = action.Next() {
		if ioAction, ok := action.Value.(IOAction); ok {
			if dryRun {
				logrus.Infof("DRYRUN: IO: %s", ioAction)
			} else {
				logrus.Infof("%s", ioAction)
				err := ioAction.Perform()
				if err != nil {
					logrus.Errorf("Error: %s", err)
					return err
				}
			}
		} else {
			logrus.Fatalf("Invalid IOAction: %s is not IOAction", ioAction)
		}
	}
	return nil
}

func (e *IOEngine) Push(action IOAction) {
	e.actions.PushBack(action)
}

// ----------------------------------

type IOAction interface {
	Perform() error
	String() string
}

// Move
type Rename struct {
	from string
	to   string
}

func (r *Rename) Perform() error {
	return os.Rename(r.from, r.to)
}

func (r *Rename) String() string {
	return fmt.Sprintf("RENAME %s --> %s", r.from, r.to)
}

type Copy struct {
	from string
	to   string
}

func (c *Copy) Perform() error {
	return CopyFile(c.from, c.to)
}

func (c *Copy) String() string {
	return fmt.Sprintf("COPY   %s --> %s", c.from, c.to)
}

type Delete struct {
	target string
}

func (d *Delete) Perform() error {
	return os.Remove(d.target)
}

func (d *Delete) String() string {
	return fmt.Sprintf("DELETE %s", d.target)
}

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	//if err = os.Link(src, dst); err == nil {
	//	return
	//}
	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
