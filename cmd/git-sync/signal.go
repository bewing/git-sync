package main

import (
	"syscall"

	"k8s.io/git-sync/pkg/gopsutil"
)

// SignalProcs will send the integer signal to processes with the listed name
func SignalProcs(flProcName string, flProcSignal int) error {
	procs, err := gopsutil.Processes()
	if err != nil {
		return err
	}
	for idx := range procs {
		name, _ := procs[idx].Name()
		if name == flProcName {
			err := procs[idx].SendSignal(syscall.Signal(flProcSignal))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
