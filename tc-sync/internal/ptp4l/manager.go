// Package ptp4l запускает ptp4l (linuxptp) как дочерние процессы и останавливает их при выходе.
package ptp4l

import (
	"log"
	"os/exec"
	"strconv"
	"sync"

	"github.com/shiwa/timecard-mini/tc-sync/internal/logger"
)

// Job — один запуск ptp4l (один интерфейс).
type Job struct {
	Interface string   // -i eth0
	Domain    int      // опционально -d N
	Path      string   // путь к ptp4l (по умолчанию "ptp4l")
	Args      []string // доп. аргументы, например ["-m", "-s"]
}

// Start запускает ptp4l для каждого job. Один интерфейс — один процесс (дубликаты по interface отбрасываются).
// Возвращает функцию stop(), которую нужно вызвать при выходе (останавливает все процессы).
func Start(jobs []Job, quiet bool) (stop func()) {
	if len(jobs) == 0 {
		return func() {}
	}
	byIface := make(map[string]Job)
	for _, j := range jobs {
		if j.Interface == "" {
			continue
		}
		if _, ok := byIface[j.Interface]; !ok {
			byIface[j.Interface] = j
		}
	}
	var cmds []*exec.Cmd
	for _, j := range byIface {
		path := j.Path
		if path == "" {
			path = "ptp4l"
		}
		args := make([]string, 0, 6+len(j.Args))
		args = append(args, "-i", j.Interface)
		args = append(args, "-d", strconv.Itoa(j.Domain))
		// Стандартные аргументы, если не заданы: slave + вывод в stdout
		if len(j.Args) == 0 {
			args = append(args, "-m", "-s")
		} else {
			args = append(args, j.Args...)
		}
		cmd := exec.Command(path, args...)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if !quiet {
			cmd.Stdout = log.Writer()
			cmd.Stderr = log.Writer()
		}
		if err := cmd.Start(); err != nil {
			logger.Info("ptp4l start %s: %v", j.Interface, err)
			continue
		}
		cmds = append(cmds, cmd)
		logger.Info("ptp4l started: %s -i %s", path, j.Interface)
	}
	var once sync.Once
	stop = func() {
		once.Do(func() {
			for _, cmd := range cmds {
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
			}
		})
	}
	return stop
}
