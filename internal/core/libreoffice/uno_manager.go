package libreoffice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// UnoManagerConfig UNO 实例管理配置。
type UnoManagerConfig struct {
	Host                string
	BasePort            int
	PoolSize            int
	LibreOfficePath     string
	UserProfileBaseDir  string
	StartupTimeout      time.Duration
	HealthcheckInterval time.Duration
	RestartAfterJobs    int
}

// UnoManager 管理多个常驻 soffice 实例。
type UnoManager struct {
	cfg       UnoManagerConfig
	logger    zerolog.Logger
	available chan *unoInstance
	instances []*unoInstance
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

type unoInstance struct {
	index        int
	host         string
	port         int
	profileDir   string
	cmd          *exec.Cmd
	exitCh       chan struct{}
	jobCount     int
	needsRestart bool
	inUse        bool
	restarting   bool
	running      bool
	mu           sync.Mutex
}

// NewUnoManager 创建 UNO 管理器。
func NewUnoManager(cfg UnoManagerConfig, logger zerolog.Logger) (*UnoManager, error) {
	if cfg.PoolSize <= 0 {
		return nil, errors.New("UNO pool_size 必须大于 0")
	}
	if cfg.BasePort < 1 || cfg.BasePort > 65535 {
		return nil, errors.New("UNO base_port 范围必须在 1-65535")
	}
	if cfg.StartupTimeout <= 0 {
		cfg.StartupTimeout = 15 * time.Second
	}
	if cfg.HealthcheckInterval <= 0 {
		cfg.HealthcheckInterval = 10 * time.Second
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}

	manager := &UnoManager{
		cfg:       cfg,
		logger:    logger,
		available: make(chan *unoInstance, cfg.PoolSize),
	}
	manager.ctx, manager.cancel = context.WithCancel(context.Background())
	return manager, nil
}

// Start 启动所有 UNO 实例并开始健康检查。
func (m *UnoManager) Start() error {
	m.instances = make([]*unoInstance, 0, m.cfg.PoolSize)
	for i := 0; i < m.cfg.PoolSize; i++ {
		inst := &unoInstance{
			index:      i,
			host:       m.cfg.Host,
			port:       m.cfg.BasePort + i,
			profileDir: filepath.Join(m.cfg.UserProfileBaseDir, fmt.Sprintf("uno-%d", i)),
		}
		if err := m.startInstance(inst); err != nil {
			m.stopAll()
			return err
		}
		m.instances = append(m.instances, inst)
		m.available <- inst
	}

	m.wg.Add(1)
	go m.healthcheckLoop()

	return nil
}

// Shutdown 停止所有实例并等待退出。
func (m *UnoManager) Shutdown(ctx context.Context) error {
	m.cancel()
	stopped := make(chan struct{})
	go func() {
		m.stopAll()
		m.wg.Wait()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-stopped:
		return nil
	}
}

// Acquire 获取一个可用实例。
func (m *UnoManager) Acquire(ctx context.Context) (*unoInstance, error) {
	select {
	case <-m.ctx.Done():
		return nil, errors.New("UNO 管理器已停止")
	case <-ctx.Done():
		return nil, ctx.Err()
	case inst := <-m.available:
		inst.mu.Lock()
		inst.inUse = true
		inst.mu.Unlock()
		if err := m.prepareInstance(ctx, inst); err != nil {
			inst.mu.Lock()
			inst.inUse = false
			inst.mu.Unlock()
			m.safeRelease(inst)
			return nil, err
		}
		return inst, nil
	}
}

// Release 释放实例并归还到池中。
func (m *UnoManager) Release(inst *unoInstance, convertErr error) {
	if inst == nil {
		return
	}
	inst.mu.Lock()
	inst.jobCount++
	inst.inUse = false
	if convertErr != nil {
		inst.needsRestart = true
	}
	if m.cfg.RestartAfterJobs > 0 && inst.jobCount >= m.cfg.RestartAfterJobs {
		inst.needsRestart = true
	}
	inst.mu.Unlock()

	m.safeRelease(inst)
}

// CheckAvailable 检查是否有可用实例。
func (m *UnoManager) CheckAvailable(ctx context.Context) error {
	inst, err := m.Acquire(ctx)
	if err != nil {
		return err
	}
	m.Release(inst, nil)
	return nil
}

func (m *UnoManager) prepareInstance(ctx context.Context, inst *unoInstance) error {
	inst.mu.Lock()
	needsRestart := inst.needsRestart
	running := inst.running
	restarting := inst.restarting
	inst.mu.Unlock()

	if restarting {
		return nil
	}
	if needsRestart || !running || !m.isPortOpen(inst.host, inst.port) {
		if err := m.restartInstance(inst); err != nil {
			return err
		}
	}
	return nil
}

func (m *UnoManager) restartInstance(inst *unoInstance) error {
	inst.mu.Lock()
	if inst.restarting {
		inst.mu.Unlock()
		return nil
	}
	inst.restarting = true
	inst.mu.Unlock()
	defer func() {
		inst.mu.Lock()
		inst.restarting = false
		inst.mu.Unlock()
	}()

	m.logger.Warn().
		Int("port", inst.port).
		Msg("UNO 实例准备重启")
	m.stopInstance(inst)
	if err := m.startInstance(inst); err != nil {
		return err
	}
	inst.mu.Lock()
	inst.jobCount = 0
	inst.needsRestart = false
	inst.mu.Unlock()
	return nil
}

func (m *UnoManager) startInstance(inst *unoInstance) error {
	if err := os.RemoveAll(inst.profileDir); err != nil {
		return fmt.Errorf("清理 UNO 用户目录失败: %w", err)
	}
	if err := os.MkdirAll(inst.profileDir, 0o755); err != nil {
		return fmt.Errorf("创建 UNO 用户目录失败: %w", err)
	}
	userProfileURL, err := fileURLFromPath(inst.profileDir)
	if err != nil {
		return fmt.Errorf("构建 UNO 用户目录 URL 失败: %w", err)
	}

	args := []string{
		"--headless",
		"--invisible",
		"--nologo",
		"--nolockcheck",
		"--nodefault",
		"--nofirststartwizard",
		"--nocrashreport",
		"--norestore",
		fmt.Sprintf("-env:UserInstallation=%s", userProfileURL),
		fmt.Sprintf("--accept=socket,host=%s,port=%d;urp;", inst.host, inst.port),
	}

	cmd := exec.Command(m.cfg.LibreOfficePath, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 UNO 实例失败: %w", err)
	}

	inst.mu.Lock()
	inst.cmd = cmd
	exitCh := make(chan struct{})
	inst.exitCh = exitCh
	inst.running = true
	inst.mu.Unlock()

	m.logger.Info().
		Int("index", inst.index).
		Int("port", inst.port).
		Str("profile_dir", inst.profileDir).
		Msg("UNO 实例已启动")

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		err := cmd.Wait()
		inst.mu.Lock()
		inst.running = false
		close(exitCh)
		inst.mu.Unlock()
		if err != nil && m.ctx.Err() == nil {
			m.logger.Warn().
				Int("port", inst.port).
				Err(err).
				Msg("UNO 实例异常退出")
		}
	}()

	if err := m.waitForPort(inst.host, inst.port, m.cfg.StartupTimeout); err != nil {
		m.stopInstance(inst)
		return err
	}
	return nil
}

func (m *UnoManager) stopInstance(inst *unoInstance) {
	inst.mu.Lock()
	cmd := inst.cmd
	exitCh := inst.exitCh
	inst.mu.Unlock()

	if cmd == nil || cmd.Process == nil || exitCh == nil {
		return
	}
	_ = cmd.Process.Signal(os.Interrupt)

	select {
	case <-exitCh:
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		select {
		case <-exitCh:
		case <-time.After(2 * time.Second):
		}
	}
}

func (m *UnoManager) stopAll() {
	for _, inst := range m.instances {
		m.stopInstance(inst)
	}
}

func (m *UnoManager) waitForPort(host string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if m.isPortOpen(host, port) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("UNO 端口未就绪: %s", net.JoinHostPort(host, strconv.Itoa(port)))
}

func (m *UnoManager) isPortOpen(host string, port int) bool {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (m *UnoManager) healthcheckLoop() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.cfg.HealthcheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			for _, inst := range m.instances {
				inst.mu.Lock()
				inUse := inst.inUse
				inst.mu.Unlock()
				if inUse {
					continue
				}
				if !m.isPortOpen(inst.host, inst.port) {
					inst.mu.Lock()
					inst.needsRestart = true
					inst.mu.Unlock()
					m.logger.Warn().Int("port", inst.port).Msg("UNO 实例端口不可达，标记重启")
				}
			}
		}
	}
}

func (m *UnoManager) safeRelease(inst *unoInstance) {
	select {
	case m.available <- inst:
	default:
		m.logger.Warn().Int("port", inst.port).Msg("UNO 实例池已满，忽略释放")
	}
}
