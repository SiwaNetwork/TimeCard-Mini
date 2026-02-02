package ssh_server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"

	"github.com/shiwa/timecard-mini/extracted-source/beater/logging"
	"github.com/shiwa/timecard-mini/extracted-source/config"
)

// SSHServer по дизассемблеру (configureServerKeys 0x4c27be0): appConfig+0x330/0x338 (host key path); loadSSHKey; если false → generateNewSSHKey(server); иначе загруженный ключ; AddHostKey(serverConfig+8, key).
type SSHServer struct {
	logger       *logging.Logger // +0 по дампу generateNewSSHKey: Logger.Info/Critical/Error
	ServerConfig interface{}     // +8 — *ssh.ServerConfig для AddHostKey
}

// NewSSHServer создаёт SSHServer с логгером (по дампу NewSSHServer — logger, serverConfig).
func NewSSHServer(serverConfig *ssh.ServerConfig) *SSHServer {
	return &SSHServer{
		logger:       logging.NewLogger("ssh-server"),
		ServerConfig: serverConfig,
	}
}

// ConfigureServerKeys по дизассемблеру (0x4c27be0): чтение appConfig.0x330/0x338; loadSSHKey(path); при неудаче — generateNewSSHKey(server, path); AddHostKey(serverConfig+8, key).
func (s *SSHServer) ConfigureServerKeys() {
	path := ""
	if cfg := config.GetAppConfig(); cfg != nil {
		path = cfg.SSHHostKey
	}
	signer, ok := s.loadSSHKey(path)
	if !ok {
		signer = s.generateNewSSHKey(path)
	}
	if signer == nil {
		return
	}
	if sc, ok := s.ServerConfig.(*ssh.ServerConfig); ok && sc != nil {
		sc.AddHostKey(signer)
	}
}

// loadSSHKey по дизассемблеру (0x4c27f80): os.ReadFile(path); при err — Logger.Error, return (nil, false); ssh.ParsePrivateKey(bytes); при err — Logger.Error, return (nil, false); return (signer, true).
func (s *SSHServer) loadSSHKey(path string) (ssh.Signer, bool) {
	if path == "" {
		return nil, false
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		if s.logger != nil {
			s.logger.Error(fmt.Sprintf("SSH key read failed: %v", err))
		}
		return nil, false
	}
	signer, err := ssh.ParsePrivateKey(bytes)
	if err != nil {
		if s.logger != nil {
			s.logger.Error(fmt.Sprintf("SSH key parse failed: %v", err))
		}
		return nil, false
	}
	return signer, true
}

// generateNewSSHKey по дизассемблеру (0x4c27ca0): Logger.Info; generatePrivateSSHKey(s); при nil — Logger.Critical; encodePrivateKeyToPEM(s, key); ssh.ParsePrivateKey(pem); при path!="" — os.WriteFile(path, pem, 0x180); return signer.
func (s *SSHServer) generateNewSSHKey(path string) ssh.Signer {
	if s.logger != nil {
		s.logger.Info("Generating new SSH host key", 0)
	}
	key := s.generatePrivateSSHKey()
	if key == nil {
		if s.logger != nil {
			s.logger.Critical("Failed to generate SSH private key", 0)
		}
		return nil
	}
	pemBytes := s.encodePrivateKeyToPEM(key)
	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		if s.logger != nil {
			s.logger.Error(fmt.Sprintf("SSH parse generated key failed: %v", err))
		}
		return nil
	}
	if path != "" {
		if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
			if s.logger != nil {
				s.logger.Error(fmt.Sprintf("SSH key write failed: %s: %v", path, err))
			}
		} else if s.logger != nil {
			s.logger.Info(fmt.Sprintf("SSH host key written to %s", path), 0)
		}
	}
	return signer
}

// generatePrivateSSHKey по дизассемблеру (0x4c281e0): rsa.GenerateKey(rand.Reader, 1024); Validate(); return key.
func (s *SSHServer) generatePrivateSSHKey() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil
	}
	if err := key.Validate(); err != nil {
		return nil
	}
	return key
}

// encodePrivateKeyToPEM по дизассемблеру (0x4c28120): x509.MarshalPKCS1PrivateKey(key); pem.EncodeToMemory("RSA PRIVATE KEY", bytes).
func (s *SSHServer) encodePrivateKeyToPEM(key *rsa.PrivateKey) []byte {
	der := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}
	return pem.EncodeToMemory(block)
}

// Run по дизассемблеру (0x4c28320): appConfig+0x320/0x328 host/port; Sprintf("%s:%d"); net.Listen("tcp"); Accept loop; newproc(func1) для каждой connection.
func (s *SSHServer) Run() {
	host := ""
	port := uint16(22)
	if cfg := config.GetAppConfig(); cfg != nil {
		if cfg.SSHListenAddr != "" {
			host = cfg.SSHListenAddr
		}
		if cfg.SSHPort != 0 {
			port = cfg.SSHPort
		}
	}
	if host == "" {
		host = "0.0.0.0"
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		if s.logger != nil {
			s.logger.Error(fmt.Sprintf("SSH listen failed %s: %v", addr, err))
		}
		return
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go s.processConnection(conn)
	}
}

// processConnection по дизассемблеру Run.func1: обработка SSH connection. Заглушка до ssh.NewServerConn, runShell.
func (s *SSHServer) processConnection(conn net.Conn) {
	defer conn.Close()
	_ = s
}

func Extracted_Go() {}
