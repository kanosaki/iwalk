package main

import (
	"container/list"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"gopkg.in/cheggaaa/pb.v1"
	"io/ioutil"
	"os"
	"path"
	"time"
)

type IOEngine struct {
	actions      *list.List
	trashActions *list.List
}

type IOAction interface {
	Perform() error
	Finish() error
	String() string
	SizeDelta() int64
	ProcessCost() int64
}

func NewIOEngine() *IOEngine {
	return &IOEngine{
		actions: list.New(),
	}
}

func (e *IOEngine) Check(targetPath string) (bool, error) {
	if e.actions.Len() == 0 {
		logrus.Infof("No actions: nothing todo")
		fmt.Println("Everything up-to-date.")
		return false, nil
	}
	var willConsume int64 = 0
	for action := e.actions.Front(); action != nil; action = action.Next() {
		if ioAction, ok := action.Value.(IOAction); ok {
			willConsume += ioAction.SizeDelta()
		} else {
			logrus.Fatalf("Invalid IOAction: %s is not IOAction", ioAction)
		}
	}
	stat, err := DiskUsage(targetPath)
	if err != nil {
		return false, err
	}
	fmt.Printf("Disk %s: Free %dMB(%d%%), will consume %dMB(%d%%)\n", targetPath, stat.Free/MiB, (stat.Free*100)/stat.All, willConsume/MiB, uint64(willConsume*100)/stat.All)
	if int64(stat.All) < int64(stat.Free)+willConsume {
		return false, errors.New("Capacity over! ")
	}
	return true, nil
}

func (e *IOEngine) Run() error {
	var wholeCost int64 = 0
	for action := e.actions.Front(); action != nil; action = action.Next() {
		if ioAction, ok := action.Value.(IOAction); ok {
			wholeCost += ioAction.ProcessCost()
		} else {
			logrus.Fatalf("Invalid IOAction: %s is not IOAction", ioAction)
		}
	}
	bar := pb.New64(int64(wholeCost)).SetUnits(pb.U_BYTES)
	bar.SetRefreshRate(500 * time.Millisecond)
	bar.ShowTimeLeft = true
	bar.ShowSpeed = true
	bar.Start()

	dryRun := *argDryRun
	// Perform
	for action := e.actions.Front(); action != nil; action = action.Next() {
		if ioAction, ok := action.Value.(IOAction); ok {
			if dryRun {
				logrus.Infof("DRYRUN: IO: %s", ioAction)
			} else {
				logrus.Debugf("%s", ioAction)
				err := ioAction.Perform()
				if err != nil {
					logrus.Errorf("Error: %s", err)
					return err
				}
				bar.Add64(ioAction.ProcessCost())
			}
		} else {
			logrus.Fatalf("Invalid IOAction: %s is not IOAction", ioAction)
		}
	}
	bar.Finish()
	logrus.Infof("Finishing sync...")
	// Finish
	for action := e.actions.Front(); action != nil; action = action.Next() {
		if ioAction, ok := action.Value.(IOAction); ok {
			if dryRun {
				logrus.Infof("DRYRUN: Finish: %s", ioAction)
			} else {
				logrus.Debugf("%s", ioAction)
				err := ioAction.Finish()
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
	if action == nil {
		logrus.Warnf("IOAction is nil!")
	} else {
		e.actions.PushBack(action)
	}
}

// ----------------------------------

// Move
type Rename struct {
	from string
	to   string
}

func NewRename(from, to string) *Rename {
	return &Rename{
		from: from,
		to:   to,
	}
}

func (r *Rename) Perform() error {
	return nil
}

func (r *Rename) Finish() error {
	return os.Rename(r.from, r.to)
}

func (r *Rename) String() string {
	return fmt.Sprintf("RENAME %s --> %s", r.from, r.to)
}

func (r *Rename) SizeDelta() int64 {
	return 0
}

func (r *Rename) ProcessCost() int64 {
	return 0
}

type Copy struct {
	from     string
	to       string
	size     int64
	track    *Track
	tempFile string
}

func NewCopy(from, to string, copyingTrack *Track) *Copy {
	st, err := os.Stat(from)
	if err != nil {
		logrus.Errorf("Cannot access: %s", from)
		return nil
	}
	tempFile := path.Join(path.Dir(to), fmt.Sprintf("%s.tmp", copyingTrack.PersistentId))
	return &Copy{
		from:     from,
		to:       to,
		tempFile: tempFile,
		track:    copyingTrack,
		size:     st.Size(),
	}
}

func (c *Copy) Perform() error {
	stat, err := os.Stat(c.tempFile)
	if err != nil {
		if os.IsNotExist(err) {
			// normal copy
			return CopyFile(c.from, c.tempFile)
		} else {
			return err
		}
	} else {
		// resume
		if stat.Size() == c.size {
			// ok
			logrus.Debugf("Skipping: %s (temp: %s, size match)", c.to, c.tempFile)
			return nil
		} else {
			// overwrite copy
			logrus.Debugf("Overwrite: broken file %s (temp: %s)", c.to, c.tempFile)
			return CopyFile(c.from, c.tempFile)
		}
	}
}

func (c *Copy) Finish() error {
	return os.Rename(c.tempFile, c.to)
}

func (c *Copy) String() string {
	return fmt.Sprintf("COPY   %s --> %s", c.from, c.to)
}

func (c *Copy) SizeDelta() int64 {
	return c.size
}

func (c *Copy) ProcessCost() int64 {
	return c.size
}

type Delete struct {
	target string
	size   int64
}

func NewDelete(target string) *Delete {
	st, err := os.Stat(target)
	if err != nil {
		logrus.Errorf("Cannot access: %s", target)
		return nil
	}
	return &Delete{
		target: target,
		size:   st.Size(),
	}
}

func (d *Delete) Perform() error {
	return nil
}

func (d *Delete) Finish() error {
	return os.Remove(d.target)
}

func (d *Delete) String() string {
	return fmt.Sprintf("DELETE %s", d.target)
}

func (d *Delete) SizeDelta() int64 {
	return -d.size
}

func (d *Delete) ProcessCost() int64 {
	return 0
}

type WriteFileAction struct {
	data       []byte
	targetPath string
	tempPath   string
}

func NewWriteFileAction(targetPath, tmpPath string, data []byte) *WriteFileAction {
	return &WriteFileAction{
		targetPath: targetPath,
		tempPath:   tmpPath,
		data:       data,
	}
}

func (uma *WriteFileAction) Perform() error {
	return ioutil.WriteFile(uma.tempPath, uma.data, 0644)
}

func (uma *WriteFileAction) Finish() error {
	return os.Rename(uma.tempPath, uma.targetPath)
}

func (uma *WriteFileAction) String() string {
	return fmt.Sprintf("WRITE  %s(%d KiB)", uma.targetPath, len(uma.data)/KiB)
}
func (uma *WriteFileAction) SizeDelta() int64 {
	return int64(len(uma.data))
}
func (uma *WriteFileAction) ProcessCost() int64 {
	return int64(len(uma.data))
}
