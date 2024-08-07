package main

import (
	"os"
)

func init() {
	_, err := os.Stat("KAP_logs")
	if os.IsNotExist(err) {
		if err := os.Mkdir("KAP_logs", 0777); err != nil {
			panic(err)
		}
	}

	_, err = os.Stat("kap_cfg.json")
	if os.IsNotExist(err) {
		fi, err := os.OpenFile("kap_cfg.json", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
		if err != nil {
			panic(err)
		}
		defer fi.Close()
		_, _ = fi.WriteString("{\n\"tick_time\": 60,\n\"coma_time\": 1800,\n\"log_time\": 2400,\n\"restart_time\": 3600,\n\"Directories\": [\n{\n\"cfg_name\": \"\",\n\"tick_time\": 60,\n\"coma_time\": 1800,\n\"log_type\": 0,\n\"log_time\": 2400,\n\"restart_time\": 3600,\n\"dir_name\": \"\",\n\"exe_name\": \"\",\n\"update_name\": \"\",\n\"update_fname\": \"\"\n}\n]\n}\n")
	}
}
