package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

func main() {
	dir_name := ""
	exe_name := ""
	update_name := ""

	var count uint8
	var pid int32
	var stimec bool
	var restart bool
	dtime := time.Now()
	fmt.Println(logtime(), "Starting...")

	for {
		time.Sleep(60 * time.Second)
		if !stimec {
			if time.Since(dtime).Minutes() < 45 {
				continue
			}
			stimec = true
		}
		count++
		entries, err := os.ReadDir(update_name)
		if err != nil {
			logerror(err)
			continue
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			fileinfo, _ := e.Info()
			modtime := fileinfo.ModTime()
			if modtime.Before(dtime) {
				dtime = modtime
			}
			tsince := time.Since(modtime).Minutes()
			if pid != 0 && tsince > 60 && !restart {
				fmt.Println(logtime(), "Restarting for old file -", e.Name())
				restart = true
				break
			} else if tsince > 45 {
				fmt.Println(logtime(), "Outdated file -", e.Name(), "-", tsince, "min")
			}
		}

		processes, _ := process.Processes()
		for _, p := range processes {
			n, _ := p.Name()
			if n != exe_name {
				continue
			}
			l, err := p.Exe()
			if err != nil {
				logerror(err)
				continue
			} else if l != dir_name+exe_name {
				continue
			}
			nid, err := p.Ppid()
			if err != nil {
				logerror(err)
				continue
			}
			if nid != pid {
				if pid != 0 {
					stimec = false
				}
				pid = nid
				dtime = time.Now()
				restart = false
			}
			if restart {
				fmt.Println(logtime(), "Killing old process...")
				p.Kill()
			}
			count = 0
		}
		if restart {
			time.Sleep(5 * time.Second)
			fmt.Println(logtime(), "Starting new process...")
			cmd := exec.Command("cmd.exe", "/C", "start", dir_name+exe_name)
			cmd.Dir = dir_name
			err := cmd.Run()
			if err != nil {
				logerror(err)
			}
		}
		if count > 2 {
			restart = true
		}
	}
}

func logtime() string {
	return fmt.Sprint(time.Now().Format("[15:04:05]"))
}

func logerror(err error) {
	fmt.Println(logtime(), err)
}
