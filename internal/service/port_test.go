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

func TestPortService_RangeMode_InitUsesWholePortRange(t *testing.T) {
	runner := &fakeIPTablesRunner{}
	svc := NewPortService(config.DynamicPortServiceConfig{
		Name:       "range-service",
		Enable:     true,
		Mode:       config.DynamicPortModeRange,
		Protocol:   "tcp",
		Min:        30000,
		Max:        30010,
		TargetPort: 443,
		Cycle:      "@every 1m",
	}, nil)
	svc.ipt = runner

	if err := svc.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if len(runner.commands) < 5 {
		t.Fatalf("expected at least 5 iptables commands, got %d", len(runner.commands))
	}
	last := runner.commands[len(runner.commands)-1]
	if got := last[7]; got != "30000:30010" {
		t.Fatalf("expected range redirect dport 30000:30010, got %q", got)
	}
	if got := last[11]; got != "443" {
		t.Fatalf("expected range redirect target port 443, got %q", got)
	}
}

func TestPortService_RangeMode_RunResyncsRedirectWithoutExtraAdds(t *testing.T) {
	runner := &fakeIPTablesRunner{}
	svc := NewPortService(config.DynamicPortServiceConfig{
		Name:       "range-service",
		Enable:     true,
		Mode:       config.DynamicPortModeRange,
		Protocol:   "udp",
		Min:        40000,
		Max:        40010,
		TargetPort: 8443,
		Cycle:      "@every 1m",
	}, nil)
	svc.ipt = runner

	if err := svc.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	svc.Run()

	redirectActions := make([]string, 0, 4)
	for _, cmd := range runner.commands {
		if len(cmd) < 12 || cmd[0] != "-t" || cmd[1] != natTable || cmd[3] != svc.chainName {
			continue
		}
		redirectActions = append(redirectActions, cmd[2])
		if got := cmd[7]; got != "40000:40010" {
			t.Fatalf("expected range redirect dport 40000:40010, got %q", got)
		}
	}

	want := []string{"-D", "-A", "-D", "-A"}
	if len(redirectActions) != len(want) {
		t.Fatalf("expected %d redirect actions, got %d: %#v", len(want), len(redirectActions), redirectActions)
	}
	for i, action := range want {
		if redirectActions[i] != action {
			t.Fatalf("expected redirect action %d to be %q, got %q", i, action, redirectActions[i])
		}
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
