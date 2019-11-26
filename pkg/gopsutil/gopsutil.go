package gopsutil

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Process holds information about a running process
type Process struct {
	Pid  int32 `json:"pid"`
	name string
}

// Processes returns a slice of pointers to Process structs
func Processes() ([]*Process, error) {
	return ProcessesWithContext(context.Background())
}

// SendSignal sends a unix.Signal to the process.
// Currently, SIGSTOP, SIGCONT, SIGTERM and SIGKILL are supported.
func (p *Process) SendSignal(sig syscall.Signal) error {
	return p.SendSignalWithContext(context.Background(), sig)
}

// SendSignal sends a unix.Signal to the process.
// Currently, SIGSTOP, SIGCONT, SIGTERM and SIGKILL are supported.
func (p *Process) SendSignalWithContext(ctx context.Context, sig syscall.Signal) error {
	process, err := os.FindProcess(int(p.Pid))
	if err != nil {
		return err
	}

	err = process.Signal(sig)
	if err != nil {
		return err
	}

	return nil
}

// ProcessesWithContext returns a slice of pointers to Process structs
func ProcessesWithContext(ctx context.Context) ([]*Process, error) {
	out := []*Process{}

	pids, err := pidsWithContext(ctx)
	if err != nil {
		return out, err
	}

	for _, pid := range pids {
		p := &Process{Pid: pid}
		out = append(out, p)
	}
	return out, nil
}

func pidsWithContext(ctx context.Context) ([]int32, error) {
	return readPidsFromDir(hostProc())
}

// Name returns name of the process.
func (p *Process) Name() (string, error) {
	return p.NameWithContext(context.Background())
}

// NameWithContext returns name of process from HOST_PROC/(pid)/status
func (p *Process) NameWithContext(ctx context.Context) (string, error) {
	if p.name == "" {
		pid := p.Pid
		statPath := hostProc(strconv.Itoa(int(pid)), "status")
		contents, err := ioutil.ReadFile(statPath)
		if err != nil {
			return "", err
		}
		lines := strings.Split(string(contents), "\n")
		for _, line := range lines {
			tabParts := strings.SplitN(line, "\t", 2)
			if len(tabParts) < 2 {
				continue
			}
			value := tabParts[1]
			switch strings.TrimRight(tabParts[0], ":") {
			case "Name":
				p.name = strings.Trim(value, " \t")
				if len(p.name) >= 15 {
					cmdlineSlice, err := p.CmdlineSlice()
					if err != nil {
						return "", err
					}
					if len(cmdlineSlice) > 0 {
						extendedName := filepath.Base(cmdlineSlice[0])
						if strings.HasPrefix(extendedName, p.name) {
							p.name = extendedName
						} else {
							p.name = cmdlineSlice[0]
						}
					}
				}
			}
		}
	}
	return p.name, nil
}

// CmdlineSlice returns the command line arguments of the process as a slice with each
// element being an argument.
func (p *Process) CmdlineSlice() ([]string, error) {
	return p.CmdlineSliceWithContext(context.Background())
}

// CmdlineSliceWithContext returns the command line arguments of the process as a slice with each
// element being an argument.
func (p *Process) CmdlineSliceWithContext(ctx context.Context) ([]string, error) {
	return p.fillSliceFromCmdlineWithContext(ctx)
}

func (p *Process) fillSliceFromCmdlineWithContext(ctx context.Context) ([]string, error) {
	pid := p.Pid
	cmdPath := hostProc(strconv.Itoa(int(pid)), "cmdline")
	cmdline, err := ioutil.ReadFile(cmdPath)
	if err != nil {
		return nil, err
	}
	if len(cmdline) == 0 {
		return nil, nil
	}
	if cmdline[len(cmdline)-1] == 0 {
		cmdline = cmdline[:len(cmdline)-1]
	}
	parts := bytes.Split(cmdline, []byte{0})
	var strParts []string
	for _, p := range parts {
		strParts = append(strParts, string(p))
	}

	return strParts, nil
}

func getEnv(key string, dfault string, combineWith ...string) string {
	value := os.Getenv(key)
	if value == "" {
		value = dfault
	}

	switch len(combineWith) {
	case 0:
		return value
	case 1:
		return filepath.Join(value, combineWith[0])
	default:
		all := make([]string, len(combineWith)+1)
		all[0] = value
		copy(all[1:], combineWith)
		return filepath.Join(all...)
	}
	panic("invalid switch case")
}

func hostProc(combineWith ...string) string {
	return getEnv("HOST_PROC", "/proc", combineWith...)
}

func readPidsFromDir(path string) ([]int32, error) {
	var ret []int32

	d, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer d.Close()

	fnames, err := d.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	for _, fname := range fnames {
		pid, err := strconv.ParseInt(fname, 10, 32)
		if err != nil {
			// if not numeric name, just skip
			continue
		}
		ret = append(ret, int32(pid))
	}

	return ret, nil
}
