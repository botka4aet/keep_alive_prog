package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
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
	Cfg_name     string `json:"cfg_name"`
	Exe_name     string `json:"exe_name"`
	Dir_name     string `json:"dir_name"`
	Update_fname string `json:"update_fname"`
	Update_name  string `json:"update_name"`

	Tick_time int     `json:"tick_time"`
	Coma_time float64 `json:"coma_time"`

	Log_time     float64 `json:"log_time"`
	Restart_time float64 `json:"restart_time"`

	Log_type bool `json:"log_type"`
}

func init() {
	err := os.Remove("KAP_logs.log")
	if err != nil && !os.IsNotExist(err) {
		fmt.Println(logtime("[MAIN]"), err)
	}
}

func main() {
	jsonFile, err := os.Open("kap_cfg.json")
	if err != nil {
		fmt.Println(logtime("MAIN"), err)
	} else {
		fmt.Println(logtime("MAIN"), "Starting MAIN process...")
	}
	defer jsonFile.Close()
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		fmt.Println(logtime("MAIN"), err)
	}
	var config kap_cfg
	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		fmt.Println(logtime("MAIN"), err)
	}
	ch := make(chan string)
	var i int
	for _, dir := range config.Directories {
		if dir.Dir_name == "" || dir.Exe_name == "" {
			continue
		}
		i++
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
		if dir.Cfg_name == "" {
			dir.Cfg_name = strconv.Itoa(i)
		}

		fmt.Println(logtime(dir.Cfg_name), "Starting...")
		go kap_routine(dir, ch)
	}
	for i > 0 {
		mes := <-ch
		fmt.Print(mes)
	}
}

func kap_routine(dir Directory, ch chan<- string) {
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
					ch <- logerror(err, dir.Cfg_name)
					continue
				}
				modtime := fileinfo.ModTime()
				tsince := time.Since(modtime).Seconds()
				if pid != 0 && tsince > dir.Restart_time && !restart {
					restart_reason = fmt.Sprintf("%v Restarting for old file - %v\n", logtime(dir.Cfg_name), fileinfo.Name())
					restart = true
				} else if tsince > dir.Log_time {
					check_result = fmt.Sprintf("%v Outdated file - %v - %.2f min\n", logtime(dir.Cfg_name), fileinfo.Name(), tsince/60)
				}
				//Проверяем папку лог-файлов
			} else {
				entries, err := os.ReadDir(dir.Update_name)
				if err != nil {
					ch <- logerror(err, dir.Cfg_name)
					continue
				}
				for _, e := range entries {
					if e.IsDir() {
						continue
					}
					fileinfo, _ := e.Info()
					modtime := fileinfo.ModTime()

					tsince := time.Since(modtime).Seconds()
					if tsince > dir.Restart_time && !restart {
						restart_reason = fmt.Sprintf("%v Restarting for old file - %v\n", logtime(dir.Cfg_name), fileinfo.Name())
						restart = true
					} else if tsince > dir.Log_time && tsince < dir.Restart_time {
						if dir.Log_type {
							check_result = check_result + fmt.Sprintf("%v Outdated file - %v - %.2f min\n", logtime(dir.Cfg_name), fileinfo.Name(), tsince/60)
						} else if !modtime.After(dtime) || check_result == "" {
							dtime = modtime
							check_result = fmt.Sprintf("%v Outdated file - %v - %.2f min\n", logtime(dir.Cfg_name), fileinfo.Name(), tsince/60)
						}
					}
				}
			}
		}
		if check_result != "" {
			ch <- check_result
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
				ch <- logerror(err, dir.Cfg_name)
				continue
			} else if l != dir.Dir_name+n {
				continue
			}
			nid, err := p.Ppid()
			if err != nil {
				ch <- logerror(err, dir.Cfg_name)
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
				ch <- fmt.Sprint(restart_reason)
				ch <- fmt.Sprintln(logtime(dir.Cfg_name), "Killing old process...")
				err := p.Kill()
				if err != nil {
					ch <- logerror(err, dir.Cfg_name)
					continue
				}
				pid = 0
			}
			count = 0
		}
		if restart {
			time.Sleep(5 * time.Second)
			ch <- fmt.Sprintln(logtime(dir.Cfg_name), "Starting new process...")
			cmd := exec.Command("cmd.exe", "/C", "start", dir.Dir_name+dir.Exe_name)
			cmd.Dir = dir.Dir_name
			err := cmd.Run()
			if err != nil {
				ch <- logerror(err, dir.Cfg_name)
			}
		}
	}
}

func logtime(cfg_name string) string {
	return fmt.Sprint("["+cfg_name+"]", time.Now().Format("[15:04:05]"))
}

func logerror(err error, cfg_name string) string {
	return fmt.Sprintln(logtime(cfg_name), err)
}
