package service

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os/exec"
	"server-master/internal/config"
	"server-master/pkg/utils"
)

// iptablesRunner specializes in executing iptables-related commands.
type iptablesRunner struct{}

func (r *iptablesRunner) Run(args ...string) error {
	return exec.Command("iptables", args...).Run()
}

type PortService struct {
	cfg   *config.Config
	queue *utils.Queue[string]
	ipt   *iptablesRunner
}

func NewPortService(cfg *config.Config, queue *utils.Queue[string]) *PortService {
	return &PortService{
		cfg:   cfg,
		queue: queue,
		ipt:   &iptablesRunner{},
	}
}

const (
	chainName = "trojan-port-redir"
	natTable  = "nat"
)

// InitIptables prepares the iptables rules for dynamic port forwarding.
func (s *PortService) InitIptables() error {
	c := s.cfg.Cron.DynamicPort
	portRange := fmt.Sprintf("%d:%d", c.Min, c.Max)

	slog.Info("Initializing iptables for dynamic ports", "range", portRange)

	// Clean up existing drop rule to avoid duplicates and re-add it.
	_ = s.ipt.Run("-D", "INPUT", "-p", "tcp", "--dport", portRange, "-j", "DROP")
	if err := s.ipt.Run("-A", "INPUT", "-p", "tcp", "--dport", portRange, "-j", "DROP"); err != nil {
		return fmt.Errorf("failed to add drop rule for range %s: %w", portRange, err)
	}

	return s.ensureCustomChain()
}

func (s *PortService) ensureCustomChain() error {
	// If the chain exists, flush it; otherwise, create it and link to PREROUTING.
	if err := s.ipt.Run("-t", natTable, "-F", chainName); err != nil {
		slog.Debug("Creating new iptables chain", "chain", chainName)
		if err := s.ipt.Run("-t", natTable, "-N", chainName); err != nil {
			return fmt.Errorf("failed to create chain %s: %w", chainName, err)
		}
		if err := s.ipt.Run("-t", natTable, "-A", "PREROUTING", "-j", chainName); err != nil {
			return fmt.Errorf("failed to link %s chain to PREROUTING: %w", chainName, err)
		}
	}
	return nil
}

// InitialSetup fills the queue with initial random ports and sets up iptables rules.
func (s *PortService) InitialSetup() {
	targetPort := fmt.Sprintf("%d", s.cfg.Cron.DynamicPort.TrojanPort)

	s.queue.Clear()
	for !s.queue.IsFull() {
		port := s.generateUniquePort()
		if err := s.modifyRedirect("-A", port, targetPort); err != nil {
			slog.Error("Failed to add initial redirect", "port", port, "error", err)
			continue
		}
		s.queue.Enqueue(port)
	}
	slog.Info("Dynamic port initial setup complete", "active_ports", s.queue.Size())
}

// RotatePort replaces one old port with a new random port.
func (s *PortService) RotatePort() {
	targetPort := fmt.Sprintf("%d", s.cfg.Cron.DynamicPort.TrojanPort)

	// Remove the oldest port from both the queue and iptables.
	if oldPort := s.queue.Dequeue(); oldPort != "" {
		if err := s.modifyRedirect("-D", oldPort, targetPort); err != nil {
			slog.Error("Failed to delete old redirect", "port", oldPort, "error", err)
		}
	}

	// Generate a new unique port and add its redirect rule.
	newPort := s.generateUniquePort()
	if err := s.modifyRedirect("-A", newPort, targetPort); err != nil {
		slog.Error("Failed to add new redirect", "port", newPort, "error", err)
		return
	}
	s.queue.Enqueue(newPort)

	slog.Info("Dynamic port rotated", "new_port", newPort)
}

func (s *PortService) generateUniquePort() string {
	c := s.cfg.Cron.DynamicPort
	for range 100 { // Limit attempts to prevent hanging if range is too small.
		p := rand.Intn(c.Max-c.Min+1) + c.Min
		port := fmt.Sprintf("%d", p)
		if !s.queue.Has(port) {
			return port
		}
	}
	return ""
}

func (s *PortService) modifyRedirect(action, srcPort, dstPort string) error {
	return s.ipt.Run("-t", natTable, action, chainName, "-p", "tcp", "--dport", srcPort, "-j", "REDIRECT", "--to-port", dstPort)
}

// Task interface implementation

func (s *PortService) Name() string {
	return "DynamicPortRotation"
}

func (s *PortService) Spec() string {
	return s.cfg.Cron.DynamicPort.Cycle
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
		slog.Error("Failed to cleanup iptables", "error", err)
	} else {
		slog.Info("Iptables cleanup complete")
	}
}

func (s *PortService) CleanupIptables() error {
	c := s.cfg.Cron.DynamicPort
	portRange := fmt.Sprintf("%d:%d", c.Min, c.Max)

	slog.Info("Cleaning up iptables for dynamic ports", "range", portRange)

	// 1. Remove the drop rule from INPUT chain
	_ = s.ipt.Run("-D", "INPUT", "-p", "tcp", "--dport", portRange, "-j", "DROP")

	// 2. Remove the jump from PREROUTING to our custom chain
	_ = s.ipt.Run("-t", natTable, "-D", "PREROUTING", "-j", chainName)

	// 3. Flush the custom chain
	_ = s.ipt.Run("-t", natTable, "-F", chainName)

	// 4. Delete the custom chain
	if err := s.ipt.Run("-t", natTable, "-X", chainName); err != nil {
		return fmt.Errorf("failed to delete chain %s: %w", chainName, err)
	}

	return nil
}
