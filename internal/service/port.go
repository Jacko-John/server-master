package service

import (
	"fmt"
	"hash/crc32"
	"log/slog"
	"math/rand"
	"os/exec"
	"server-master/internal/config"
	"server-master/pkg/utils"
)

// iptablesRunner specializes in executing iptables-related commands.
type iptablesRunner interface {
	Run(args ...string) error
}

type execIptablesRunner struct{}

func (r *execIptablesRunner) Run(args ...string) error {
	return exec.Command("iptables", args...).Run()
}

type PortService struct {
	dpCfg     config.DynamicPortServiceConfig
	queue     *utils.Queue[string]
	ipt       iptablesRunner
	chainName string
}

func NewPortService(dpCfg config.DynamicPortServiceConfig, queue *utils.Queue[string]) *PortService {
	return &PortService{
		dpCfg:     dpCfg,
		queue:     queue,
		ipt:       &execIptablesRunner{},
		chainName: buildChainName(dpCfg.Name),
	}
}

const natTable = "nat"

func buildChainName(name string) string {
	return fmt.Sprintf("smdp-%08x", crc32.ChecksumIEEE([]byte(name)))
}

func (s *PortService) isRangeMode() bool {
	return s.dpCfg.Mode == config.DynamicPortModeRange
}

func (s *PortService) portRange() string {
	return fmt.Sprintf("%d:%d", s.dpCfg.Min, s.dpCfg.Max)
}

func (s *PortService) targetPort() string {
	return fmt.Sprintf("%d", s.dpCfg.TargetPort)
}

func (s *PortService) syncRangeRedirect() error {
	portRange := s.portRange()
	targetPort := s.targetPort()
	_ = s.modifyRedirect("-D", portRange, targetPort)
	if err := s.modifyRedirect("-A", portRange, targetPort); err != nil {
		return fmt.Errorf("failed to add range redirect for %s: %w", portRange, err)
	}
	return nil
}

// InitIptables prepares the iptables rules for dynamic port forwarding.
func (s *PortService) InitIptables() error {
	c := s.dpCfg
	portRange := s.portRange()

	slog.Info("Initializing iptables for dynamic ports", "service", c.Name, "protocol", c.Protocol, "range", portRange)

	_ = s.ipt.Run("-D", "INPUT", "-p", c.Protocol, "--dport", portRange, "-j", "DROP")
	if err := s.ipt.Run("-A", "INPUT", "-p", c.Protocol, "--dport", portRange, "-j", "DROP"); err != nil {
		return fmt.Errorf("failed to add drop rule for range %s: %w", portRange, err)
	}

	return s.ensureCustomChain()
}

func (s *PortService) ensureCustomChain() error {
	if err := s.ipt.Run("-t", natTable, "-F", s.chainName); err != nil {
		slog.Debug("Creating new iptables chain", "service", s.dpCfg.Name, "chain", s.chainName)
		if err := s.ipt.Run("-t", natTable, "-N", s.chainName); err != nil {
			return fmt.Errorf("failed to create chain %s: %w", s.chainName, err)
		}
		if err := s.ipt.Run("-t", natTable, "-A", "PREROUTING", "-j", s.chainName); err != nil {
			return fmt.Errorf("failed to link %s chain to PREROUTING: %w", s.chainName, err)
		}
	}
	return nil
}

// InitialSetup fills the queue with initial random ports and sets up iptables rules.
func (s *PortService) InitialSetup() {
	if s.isRangeMode() {
		if err := s.syncRangeRedirect(); err != nil {
			slog.Error("Failed to sync initial range redirect", "service", s.dpCfg.Name, "error", err)
			return
		}
		slog.Info("Dynamic port range setup complete", "service", s.dpCfg.Name, "range", s.portRange())
		return
	}
	if s.queue == nil {
		return
	}
	targetPort := s.targetPort()

	s.queue.Clear()
	for !s.queue.IsFull() {
		port := s.generateUniquePort()
		if port == "" {
			slog.Error("Failed to generate initial dynamic port", "service", s.dpCfg.Name)
			break
		}
		if err := s.modifyRedirect("-A", port, targetPort); err != nil {
			slog.Error("Failed to add initial redirect", "service", s.dpCfg.Name, "port", port, "error", err)
			continue
		}
		s.queue.Enqueue(port)
	}
	slog.Info("Dynamic port initial setup complete", "service", s.dpCfg.Name, "active_ports", s.queue.Size())
}

// RotatePort replaces one old port with a new random port.
func (s *PortService) RotatePort() {
	if s.isRangeMode() {
		if err := s.syncRangeRedirect(); err != nil {
			slog.Error("Failed to sync range redirect", "service", s.dpCfg.Name, "error", err)
		}
		return
	}
	if s.queue == nil {
		return
	}
	targetPort := s.targetPort()

	if oldPort := s.queue.Dequeue(); oldPort != "" {
		if err := s.modifyRedirect("-D", oldPort, targetPort); err != nil {
			slog.Error("Failed to delete old redirect", "service", s.dpCfg.Name, "port", oldPort, "error", err)
		}
	}

	newPort := s.generateUniquePort()
	if newPort == "" {
		slog.Error("Failed to generate new dynamic port", "service", s.dpCfg.Name)
		return
	}
	if err := s.modifyRedirect("-A", newPort, targetPort); err != nil {
		slog.Error("Failed to add new redirect", "service", s.dpCfg.Name, "port", newPort, "error", err)
		return
	}
	s.queue.Enqueue(newPort)

	slog.Info("Dynamic port rotated", "service", s.dpCfg.Name, "new_port", newPort)
}

func (s *PortService) generateUniquePort() string {
	c := s.dpCfg
	for range 100 {
		p := rand.Intn(c.Max-c.Min+1) + c.Min
		port := fmt.Sprintf("%d", p)
		if s.queue == nil || !s.queue.Has(port) {
			return port
		}
	}
	return ""
}

func (s *PortService) modifyRedirect(action, srcPort, dstPort string) error {
	return s.ipt.Run("-t", natTable, action, s.chainName, "-p", s.dpCfg.Protocol, "--dport", srcPort, "-j", "REDIRECT", "--to-port", dstPort)
}

// Task interface implementation

func (s *PortService) Name() string {
	return fmt.Sprintf("DynamicPortRotation[%s]", s.dpCfg.Name)
}

func (s *PortService) Spec() string {
	return s.dpCfg.Cycle
}

func (s *PortService) Run() {
	s.RotatePort()
}

func (s *PortService) Init() error {
	if err := s.InitIptables(); err != nil {
		return err
	}
	s.InitialSetup()
	return nil
}

// Cleanup removes all iptables rules created by this service.
func (s *PortService) Cleanup() {
	if err := s.CleanupIptables(); err != nil {
		slog.Error("Failed to cleanup iptables", "service", s.dpCfg.Name, "error", err)
	} else {
		slog.Info("Iptables cleanup complete", "service", s.dpCfg.Name)
	}
}

func (s *PortService) CleanupIptables() error {
	c := s.dpCfg
	portRange := s.portRange()

	slog.Info("Cleaning up iptables for dynamic ports", "service", c.Name, "protocol", c.Protocol, "range", portRange)

	_ = s.ipt.Run("-D", "INPUT", "-p", c.Protocol, "--dport", portRange, "-j", "DROP")
	_ = s.ipt.Run("-t", natTable, "-D", "PREROUTING", "-j", s.chainName)
	_ = s.ipt.Run("-t", natTable, "-F", s.chainName)
	if err := s.ipt.Run("-t", natTable, "-X", s.chainName); err != nil {
		return fmt.Errorf("failed to delete chain %s: %w", s.chainName, err)
	}

	return nil
}
