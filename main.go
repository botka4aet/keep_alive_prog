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
	Log_type     bool    `json:"log_type"`
	Restart_time float64 `json:"restart_time"`
	Exe_name     string  `json:"exe_name"`
	Dir_name     string  `json:"dir_name"`
	Update_fname string  `json:"update_fname"`
	Update_name  string  `json:"update_name"`
}

func main() {
	jsonFile, err := os.Open("kap_cfg.json")
	if err != nil {
		logerror(err,"[MAIN]")
	} else {
		fmt.Println(logtime("[MAIN]"), "Starting MAIN process...")
	}
	defer jsonFile.Close()
	byteValue, _ := io.ReadAll(jsonFile)
	var config kap_cfg
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

		fmt.Println(logtime(dir.Cfg_name), "Starting...")
		go kap_routine(dir)
	}
	for {
		time.Sleep(3600)
	}
}

func kap_routine(dir Directory) {
	var count uint8
	var pid int32
	var stimec bool
	ticktime := time.Second * time.Duration(dir.Tick_time)
	var check_type bool
	if dir.Update_fname != "" {
		check_type = true
	}

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

		var restart bool
		var restart_reason string
		var check_result string
		//Проверяем логи, если мы уже отслеживаем процесс
		if pid != 0 {
			//Проверяем определенный лог-файл
			if check_type {
				fileinfo, err := os.Stat(dir.Update_fname)
				if err != nil {
					logerror(err,dir.Cfg_name)
					continue
				}
				modtime := fileinfo.ModTime()
				tsince := time.Since(modtime).Seconds()
				if pid != 0 && tsince > dir.Restart_time && !restart {
					restart_reason = fmt.Sprint(logtime(dir.Cfg_name), " Restarting for old file", fileinfo.Name(), "\n")
					restart = true
				} else if tsince > dir.Log_time {
					check_result = fmt.Sprintf("%vOutdated file - %v - %.2f min\n", logtime(dir.Cfg_name),fileinfo.Name(), tsince/60)
				}
			//Проверяем папку лог-файлов
			} else {
				entries, err := os.ReadDir(dir.Update_name)
				if err != nil {
					logerror(err,dir.Cfg_name)
					continue
				}
				for _, e := range entries {
					if e.IsDir() {
						continue
					}
					fileinfo, _ := e.Info()
					modtime := fileinfo.ModTime()

					tsince := time.Since(modtime).Seconds()
					if pid != 0 && tsince > dir.Restart_time && !restart {
						restart_reason = fmt.Sprint(logtime(dir.Cfg_name), "Restarting for old file", fileinfo.Name(), "\n")
						restart = true
					} else if tsince > dir.Log_time && tsince < dir.Restart_time {
						if dir.Log_type {
							check_result = check_result + fmt.Sprintf("%vOutdated file - %v - %.2f min\n", logtime(dir.Cfg_name),fileinfo.Name(), tsince/60)
						} else if !modtime.After(dtime) {
							dtime = modtime
							check_result = fmt.Sprintf("%vOutdated file - %v - %.2f min\n", logtime(dir.Cfg_name),fileinfo.Name(), tsince/60)
						}
					}
				}
			}
		}
		if check_result != "" {
			fmt.Print(check_result)
		}
		if count > 5 {
			restart_reason = fmt.Sprint(logtime(dir.Cfg_name), " Restarting for errors\n")
			restart = true
		}

		processes, _ := process.Processes()
		for _, p := range processes {
			n, _ := p.Name()
			if n != dir.Exe_name {
				continue
			}
			l, err := p.Exe()
			if err != nil {
				logerror(err,dir.Cfg_name)
				continue
			} else if l != dir.Dir_name+n {
				continue
			}
			nid, err := p.Ppid()
			if err != nil {
				logerror(err,dir.Cfg_name)
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
				fmt.Print(restart_reason)
				fmt.Println(logtime(dir.Cfg_name), "Killing old process...")
				p.Kill()
				pid = 0
			}
			count = 0
		}
		if restart {
			time.Sleep(5 * time.Second)
			fmt.Println(logtime(dir.Cfg_name), "Starting new process...")
			cmd := exec.Command("cmd.exe", "/C", "start", dir.Dir_name+dir.Exe_name)
			cmd.Dir = dir.Dir_name
			err := cmd.Run()
			if err != nil {
				logerror(err,dir.Cfg_name)
			}
		}
	}
}

func logtime(cfg_name string) string {
	return fmt.Sprint("["+cfg_name+"]",time.Now().Format("[15:04:05]"))
}

func logerror(err error,cfg_name string) {
	fmt.Println(logtime(cfg_name), err)
}
