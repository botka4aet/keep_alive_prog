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

	Log_type uint8 `json:"log_type"`
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
	log_file := "KAP_logs\\" + time.Now().Format("2006-01-02-15-04-05") + ".log"
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
		if dir.Update_fname != "" {
			dir.Update_name = ""
		} else if dir.Log_type&4 != 0 || (dir.Update_name == "" && dir.Update_fname == "") {
			dir.Update_fname = log_file
		}

		fmt.Println(logtime(dir.Cfg_name), "Starting...")
		go kap_routine(dir, ch)
	}
	if i > 0 {
		//Откроем файл для записи логов
		fi, err := os.OpenFile(log_file, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
		if err != nil {
			panic(err)
		}
		defer fi.Close()

		for {
			mes := <-ch
			if mes != "" {
				fmt.Print(mes)
				_, _ = fi.WriteString(mes)
			} else {
				err := os.Chtimes(log_file, time.Now(), time.Now())
				if err != nil {
					logerror(err,"MAIN")
				}
			}
		}
	}
}

func kap_routine(dir Directory, ch chan<- string) {
	var err error
	var count uint8
	var pid int32
	ticktime := time.Second * time.Duration(dir.Tick_time)

	var restart bool
	var coma_status bool
	var cp *process.Process
	dtime := time.Now()
	for {
		time.Sleep(ticktime)
		if dir.Log_type&4 != 0 {
			ch <- ""
		}
		if time.Since(dtime).Seconds() < dir.Coma_time && pid != 0 {
			if dir.Log_type&1 != 0 && !coma_status {
				ch <- fmt.Sprintln(logtime(dir.Cfg_name), "Skiping time~")
				coma_status = true
			}
			continue
		} else if dir.Log_type&1 != 0 && coma_status && pid != 0 {
			ch <- fmt.Sprintln(logtime(dir.Cfg_name), "On watch.")
			coma_status = false
		}
		count++

		var restart_reason string
		var running bool
		if pid > 0 {
			running, err = cp.IsRunning()
			if err != nil {
				ch <- logerror(err, dir.Cfg_name)
				continue
			}
		}
		if !running {
			processes, _ := process.Processes()
			for _, p := range processes {
				n, _ := p.Name()
				if n != dir.Exe_name {
					continue
				}
				l, err := p.Exe()
				if err != nil {
					ch <- logerror(err, dir.Cfg_name)
					break
				} else if l != dir.Dir_name+n {
					continue
				}
				pid = p.Pid
				cp = p
				dtime = time.Now()
				if dir.Log_type&1 != 0 {
					ch <- fmt.Sprintln(logtime(dir.Cfg_name), "ID changed.")
				}
				restart = false
				count = 0
				break
			}
			//Если не удалось получить ID процесса несколько раз, то запускаем новый
			if count > 5 {
				restart_reason = fmt.Sprint(logtime(dir.Cfg_name), " Restarting for errors\n")
				restart = true
				pid = 0
			} else {
				continue
			}
		}

		var check_result string
		//Проверяем определенный лог-файл
		if dir.Update_fname != "" && !restart {
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
		} else if !restart {
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
					if dir.Log_type&2 != 0 {
						check_result = check_result + fmt.Sprintf("%v Outdated file - %v - %.2f min\n", logtime(dir.Cfg_name), fileinfo.Name(), tsince/60)
					} else if !modtime.After(dtime) || check_result == "" {
						dtime = modtime
						check_result = fmt.Sprintf("%v Outdated file - %v - %.2f min\n", logtime(dir.Cfg_name), fileinfo.Name(), tsince/60)
					}
				}
			}
		}
		if check_result != "" {
			ch <- check_result
		}

		if restart {
			ch <- fmt.Sprint(restart_reason)
			if pid != 0 {
				ch <- fmt.Sprintln(logtime(dir.Cfg_name), "Killing old process...")
				err := cp.Kill()
				if err != nil {
					ch <- logerror(err, dir.Cfg_name)
					break
				}
				pid = 0

			}
			time.Sleep(5 * time.Second)
			ch <- fmt.Sprintln(logtime(dir.Cfg_name), "Starting new process...")
			cmd := exec.Command("cmd.exe", "/C", "start", dir.Dir_name+dir.Exe_name)
			cmd.Dir = dir.Dir_name
			err := cmd.Run()
			if err != nil {
				ch <- logerror(err, dir.Cfg_name)
			} else {
				dtime = time.Now()
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
