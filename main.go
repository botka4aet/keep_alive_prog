package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

type kap_cfg struct {
	Tick_time    int         `json:"tick_time"`
	Coma_time    float64     `json:"coma_time"`
	Log_time     float64     `json:"log_time"`
	Restart_time float64     `json:"restart_time"`
	Directories  []Directory `json:"Directories"`
}

type Directory struct {
	Cfg_name     string  `json:"cfg_name"`
	Tick_time    int     `json:"tick_time"`
	Coma_time    float64 `json:"coma_time"`
	Log_time     float64 `json:"log_time"`
	Restart_time float64 `json:"restart_time"`
	Exe_name     string  `json:"exe_name"`
	Dir_name     string  `json:"dir_name"`
	Update_fname string  `json:"update_fname"`
	Update_name  string  `json:"update_name"`
}

func main() {
	jsonFile, err := os.Open("kap_cfg.json")
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()
	byteValue, _ := io.ReadAll(jsonFile)
	var config kap_cfg

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(byteValue, &config)

	for _, dir := range config.Directories {
		if dir.Dir_name == "" || dir.Exe_name == "" {
			continue
		}
		if dir.Tick_time == 0 {
			dir.Tick_time = max(config.Tick_time, 1)
		}
		if dir.Coma_time == 0 {
			dir.Coma_time = config.Coma_time
		}
		if dir.Log_time == 0 {
			dir.Log_time = config.Log_time
		}
		if dir.Restart_time == 0 {
			dir.Restart_time = config.Restart_time
		}

		fmt.Println(logtime(), "Starting...", dir.Cfg_name)
		go kap_routine(dir)
	}
}

func kap_routine(dir Directory) {
	var count uint8
	var pid int32
	var stimec bool
	var restart bool
	ticktime := time.Second * time.Duration(dir.Tick_time)
	dtime := time.Now()
	for {
		time.Sleep(ticktime)
		if !stimec {
			if time.Since(dtime).Seconds() < dir.Coma_time {
				continue
			}
			stimec = true
		}
		count++
		entries, err := os.ReadDir(dir.Update_name)
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
			tsince := time.Since(modtime).Seconds()
			if pid != 0 && tsince > dir.Restart_time && !restart {
				fmt.Println(logtime(), "Restarting for old file -", e.Name())
				restart = true
				break
			} else if tsince > dir.Log_time {
				fmt.Println(logtime(), "Outdated file -", e.Name(), "-", tsince/60, "min")
			}
		}

		processes, _ := process.Processes()
		for _, p := range processes {
			n, _ := p.Name()
			if n != dir.Exe_name {
				continue
			}
			l, err := p.Exe()
			if err != nil {
				logerror(err)
				continue
			} else if l != dir.Dir_name+n {
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
			cmd := exec.Command("cmd.exe", "/C", "start", dir.Dir_name+dir.Exe_name)
			cmd.Dir = dir.Dir_name
			err := cmd.Run()
			if err != nil {
				logerror(err)
			}
		}
		if count > 5 {
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
