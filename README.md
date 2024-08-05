# keep_alive_prog

## Сокращения
ОП - отслеживаемое приложение
 
## Настройка
Значения в секундах, которые будут применены, если не указаны для конкретной программы
```
{
"tick_time": 60, - Время, через которое будет происходить обновление статуса ОП
"coma_time": 1800, - Время, в течении которого обновление статуса происходить не будет(после запуска KAP или перезапуска ОП)
"log_time": 2400, - Если файлы логов ОП старше этого времени, то вывести в консоль
"restart_time": 3600, - Если файлы логов ОП старше этого времени, то перезапустить ОП
"Directories": []
}
```
Настройка для каждого ОП отдельно
```
[
"tick_time": 60, - см.выше
"coma_time": 1800, - см.выше
"log_time": 2400, - см.выше
"restart_time": 3600, - см.выше
"cfg_name": "", - Название ОП для отображения в логах
"log_type": false, - Если у ОП несколько устаревших логов, то сообщать о все(true) или только об одном(false)
"dir_name": "", - Путь до .exe файла ОП, двойной обратный слеш в пути(\\) и заканчивается ими же 
"exe_name": "", - Название .exe файла ОП,
"update_fname": "", - Путь до лог-файла ОП, двойной обратный слеш в пути(\\) 
"update_name": "" - Путь до папки с лог-файлами ОП, двойной обратный слеш в пути(\\). Если есть update_fname, то будет ИГНОРИРОВАТЬСЯ  
]
```
