package service

import (
	"server-master/internal/config"
	"server-master/pkg/utils"
	"testing"
)

type fakeIPTablesRunner struct {
	commands [][]string
	err      error
}

func (r *fakeIPTablesRunner) Run(args ...string) error {
	copied := append([]string(nil), args...)
	r.commands = append(r.commands, copied)
	return r.err
}

func TestPortService_InitIptables_UsesConfiguredProtocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
	}{
		{name: "tcp service", protocol: "tcp"},
		{name: "udp service", protocol: "udp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeIPTablesRunner{}
			svc := NewPortService(config.DynamicPortServiceConfig{
				Name:       tt.name,
				Enable:     true,
				Protocol:   tt.protocol,
				Min:        10000,
				Max:        10010,
				ActiveNum:  1,
				TargetPort: 443,
				Cycle:      "@every 1m",
			}, utils.NewQueue[string](1))
			svc.ipt = runner

			if err := svc.InitIptables(); err != nil {
				t.Fatalf("InitIptables failed: %v", err)
			}
			if len(runner.commands) < 2 {
				t.Fatalf("expected at least 2 iptables commands, got %d", len(runner.commands))
			}
			if got := runner.commands[0][3]; got != tt.protocol {
				t.Fatalf("expected delete rule protocol %q, got %q", tt.protocol, got)
			}
			if got := runner.commands[1][3]; got != tt.protocol {
				t.Fatalf("expected add rule protocol %q, got %q", tt.protocol, got)
			}
		})
	}
}

func TestPortService_CleanupIptables_UsesOwnChain(t *testing.T) {
	runner := &fakeIPTablesRunner{}
	svc := NewPortService(config.DynamicPortServiceConfig{
		Name:       "udp-service",
		Enable:     true,
		Protocol:   "udp",
		Min:        20000,
		Max:        20010,
		ActiveNum:  1,
		TargetPort: 8443,
		Cycle:      "@every 1m",
	}, utils.NewQueue[string](1))
	svc.ipt = runner

	if err := svc.CleanupIptables(); err != nil {
		t.Fatalf("CleanupIptables failed: %v", err)
	}
	if len(runner.commands) != 4 {
		t.Fatalf("expected 4 cleanup commands, got %d", len(runner.commands))
	}
	if got := runner.commands[0][3]; got != "udp" {
		t.Fatalf("expected cleanup protocol udp, got %q", got)
	}
	if got := runner.commands[1][5]; got != svc.chainName {
		t.Fatalf("expected PREROUTING cleanup chain %q, got %q", svc.chainName, got)
	}
	if got := runner.commands[2][3]; got != svc.chainName {
		t.Fatalf("expected flush chain %q, got %q", svc.chainName, got)
	}
	if got := runner.commands[3][3]; got != svc.chainName {
		t.Fatalf("expected delete chain %q, got %q", svc.chainName, got)
	}
}

func TestPortService_NameAndChainNameAreUnique(t *testing.T) {
	svcA := NewPortService(config.DynamicPortServiceConfig{Name: "alpha", Enable: true, Protocol: "tcp", Min: 10000, Max: 10010, ActiveNum: 1, TargetPort: 443, Cycle: "@every 1m"}, utils.NewQueue[string](1))
	svcB := NewPortService(config.DynamicPortServiceConfig{Name: "beta", Enable: true, Protocol: "tcp", Min: 10020, Max: 10030, ActiveNum: 1, TargetPort: 443, Cycle: "@every 1m"}, utils.NewQueue[string](1))

	if svcA.Name() == svcB.Name() {
		t.Fatalf("expected unique task names, got %q", svcA.Name())
	}
	if svcA.chainName == svcB.chainName {
		t.Fatalf("expected unique chain names, got %q", svcA.chainName)
	}
}

func TestCronService_AddTaskRejectsDuplicateNames(t *testing.T) {
	cronService := NewCronService()
	task := &stubTask{name: "dup", spec: "@every 1m"}
	if err := cronService.AddTask(task); err != nil {
		t.Fatalf("first AddTask failed: %v", err)
	}
	if err := cronService.AddTask(task); err == nil {
		t.Fatal("expected duplicate task registration to fail")
	}
}

type stubTask struct {
	name string
	spec string
}

func (t *stubTask) Name() string { return t.name }
func (t *stubTask) Spec() string { return t.spec }
func (t *stubTask) Run()         {}
