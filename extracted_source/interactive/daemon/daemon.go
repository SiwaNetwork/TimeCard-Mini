package daemon

import (
	"sync"

	"github.com/shiwa/timecard-mini/extracted-source/config"
	"github.com/shiwa/timecard-mini/extracted-source/interactive/daemon/http_server"
	"github.com/shiwa/timecard-mini/extracted-source/interactive/daemon/ssh_server"
	"golang.org/x/crypto/ssh"
)

var (
	sshServerInstance   *ssh_server.SSHServer
	sshServerOnce       sync.Once
	httpServerInstance  *http_server.HttpServer
	httpServerOnce      sync.Once
)

// GetSSHServerInstance по дизассемблеру: sync.Once, создаёт SSHServer (NewSSHServer, ConfigureServerKeys), возвращает экземпляр.
func GetSSHServerInstance() *ssh_server.SSHServer {
	sshServerOnce.Do(func() {
		var cfg ssh.ServerConfig
		sshServerInstance = ssh_server.NewSSHServer(&cfg)
		if sshServerInstance != nil {
			sshServerInstance.ConfigureServerKeys()
		}
	})
	return sshServerInstance
}

// GetHTTPServerInstance по дизассемблеру: sync.Once, создаёт HttpServer, возвращает экземпляр.
func GetHTTPServerInstance() *http_server.HttpServer {
	httpServerOnce.Do(func() {
		httpServerInstance = &http_server.HttpServer{}
	})
	return httpServerInstance
}

// Автоматически извлечено из timebeat-2.2.20

func Extracted_Init_1() {
	_ = config.GetAppConfig
}

func CmdLineCompletionCallback() {
	// TODO: реконструировать
}

func CmdLineCompletionCallbackFm() {
	// TODO: реконструировать
}

// CreateCommands по дизассемблеру (0x4c209e0, size 0x3000): строит дерево команд — newobject(uKbinQcm); [2]string "show","version" → ShowVersion·f; [2]string "show","help" → ShowHelp·f; [3]string "show","phc","devices" → api.GetPHCDevices·f; [4]string "show",... и т.д. Регистрация через RegisterCommand. Заглушка до полной реконструкции.
func CreateCommands() {
	// TODO: реконструировать — полный список команд по дампу (ShowVersion, ShowHelp, show phc devices, show running-config, configure, exit, logout, …)
}

// DispatchCommand по дизассемблеру (0x4c29200 (*OneTerminal).DispatchCommand): receiver+0x10=state; если state==0: len==4 "exit" или len==6 "logout" → SetPrompt, state=0; len==9 "configure" → SetPrompt("timebeat(config)#"), state=1; len==19 "show running-config" → generateStartupConfig; иначе processCmd(cmd, receiver+0x18=commands). При state==1: "exit" → SetPrompt, state=0. Заглушка.
func DispatchCommand() {
	// TODO: реконструировать — тип OneTerminal (0x8=*Terminal, 0x10=state, 0x18=commands), processCmd 0x4c2b3e0
}


func NewSSHServer() {
	// TODO: реконструировать
}

func ProcessShowCommand() {
	// TODO: реконструировать
}

func RegisterCommand() {
	// TODO: реконструировать
}

func Run() {
	// TODO: реконструировать
}

func ShowHelp() {
	// TODO: реконструировать
}

func ShowVersion() {
	// TODO: реконструировать
}

func commands() {
	// TODO: реконструировать
}

// configureServerKeys по дизассемблеру (*SSHServer).configureServerKeys 0x4c27be0: вызов на экземпляре SSHServer. См. ssh_server.ConfigureServerKeys.
func configureServerKeys() {
	var s ssh_server.SSHServer
	s.ConfigureServerKeys()
}

func createSSHConfig() {
	// TODO: реконструировать
}

func encodePrivateKeyToPEM() {
	// TODO: реконструировать
}

func func1() {
	// TODO: реконструировать
}

func func2() {
	// TODO: реконструировать
}

func func3() {
	// TODO: реконструировать
}

func generateNewSSHKey() {
	// TODO: реконструировать
}

func generatePrivateSSHKey() {
	// TODO: реконструировать
}

func generateStartupConfig() {
	// TODO: реконструировать
}

func httpServer() {
	// TODO: реконструировать
}

func httpServerOnceStub() {
	// TODO: реконструировать (conflict with var httpServerOnce)
}

func inittask() {
	// TODO: реконструировать
}

func instance() {
	// TODO: реконструировать
}

func loadAuthorisedKeys() {
	// TODO: реконструировать
}

func loadConfig() {
	// TODO: реконструировать
}

func loadSSHKey() {
	// TODO: реконструировать
}

func once() {
	// TODO: реконструировать
}

func outputCLIShow() {
	// TODO: реконструировать
}

func outputCLIShowFm() {
	// TODO: реконструировать
}

func outputFormattedJSON() {
	// TODO: реконструировать
}

func outputGroupOffset() {
	// TODO: реконструировать
}

func outputGroupOffsetFm() {
	// TODO: реконструировать
}

func outputMasterTable() {
	// TODO: реконструировать
}

func outputMasterTableFm() {
	// TODO: реконструировать
}

// outputTimeSourcesStatus по дизассемблеру: вызов (*HttpServer).OutputTimeSourcesStatus(); используется как коллбэк. См. http_server.OutputTimeSourcesStatus.
func outputTimeSourcesStatus() {
	var h http_server.HttpServer
	h.OutputTimeSourcesStatus()
}

func outputTimeSourcesStatusFm() {
	// TODO: реконструировать (closure для HTTP handler)
}

func parseFlatStructure() {
	// TODO: реконструировать
}

func parseSlice() {
	// TODO: реконструировать
}

func processCmd() {
	// TODO: реконструировать
}

func processConnection() {
	// TODO: реконструировать
}

func processSSHTab() {
	// TODO: реконструировать
}

func runBroadcastReceiverLoop() {
	// TODO: реконструировать
}

func runShell() {
	// TODO: реконструировать
}

func writeKeyToFile() {
	// TODO: реконструировать
}

