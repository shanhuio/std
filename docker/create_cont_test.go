package docker

import (
	"testing"
)

func TestNewCreateContainerRequestNilConfig(t *testing.T) {
	req := newCreateContainerRequest("nginx", nil)
	if req.Image != "nginx" {
		t.Errorf("Image: got %q, want %q", req.Image, "nginx")
	}
	if req.HostConfig != nil {
		t.Errorf("HostConfig: got %+v, want nil", req.HostConfig)
	}
}

func TestNewCreateContainerRequestEmptyConfig(t *testing.T) {
	req := newCreateContainerRequest("nginx", &ContConfig{})
	if req.HostConfig == nil {
		t.Fatal("HostConfig: got nil, want non-nil")
	}
	if req.HostConfig.Privileged {
		t.Errorf("HostConfig.Privileged: got true, want false")
	}
}

func TestNewCreateContainerRequestBasicFields(t *testing.T) {
	req := newCreateContainerRequest("nginx", &ContConfig{
		Hostname:   "web",
		WorkDir:    "/app",
		Privileged: true,
		Network:    "host",
	})
	if req.Hostname != "web" {
		t.Errorf("Hostname: got %q, want %q", req.Hostname, "web")
	}
	if req.WorkingDir != "/app" {
		t.Errorf("WorkingDir: got %q, want %q", req.WorkingDir, "/app")
	}
	if !req.HostConfig.Privileged {
		t.Errorf("Privileged: got false, want true")
	}
	if req.HostConfig.NetworkMode != "host" {
		t.Errorf("NetworkMode: got %q, want %q", req.HostConfig.NetworkMode, "host")
	}
}

func TestNewCreateContainerRequestEnv(t *testing.T) {
	req := newCreateContainerRequest("nginx", &ContConfig{
		Env: map[string]string{"FOO": "bar", "BAZ": "qux"},
	})
	// unmapEnv sorts keys alphabetically.
	want := []string{"BAZ=qux", "FOO=bar"}
	if len(req.Env) != len(want) {
		t.Fatalf("Env: got %v, want %v", req.Env, want)
	}
	for i, v := range want {
		if req.Env[i] != v {
			t.Errorf("Env[%d]: got %q, want %q", i, req.Env[i], v)
		}
	}
}

func TestNewCreateContainerRequestCmdAndLabels(t *testing.T) {
	req := newCreateContainerRequest("nginx", &ContConfig{
		Cmd:    []string{"sh", "-c", "echo hi"},
		Labels: map[string]string{"role": "frontend"},
	})
	if len(req.Cmd) != 3 || req.Cmd[0] != "sh" {
		t.Errorf("Cmd: got %v", req.Cmd)
	}
	if req.Labels["role"] != "frontend" {
		t.Errorf("Labels: got %+v", req.Labels)
	}
}

func TestNewCreateContainerRequestMounts(t *testing.T) {
	req := newCreateContainerRequest("nginx", &ContConfig{
		Mounts: []*ContMount{
			{Host: "/host/data", Cont: "/data", ReadOnly: true},
			{Host: "/host/cfg", Cont: "/cfg", Type: MountVolume},
		},
	})
	mounts := req.HostConfig.Mounts
	if len(mounts) != 2 {
		t.Fatalf("Mounts: got %d, want 2", len(mounts))
	}
	if mounts[0].Source != "/host/data" || mounts[0].Target != "/data" ||
		!mounts[0].ReadOnly || mounts[0].Type != MountBind {
		t.Errorf("Mounts[0]: got %+v", mounts[0])
	}
	if mounts[1].Type != MountVolume {
		t.Errorf("Mounts[1].Type: got %q, want %q", mounts[1].Type, MountVolume)
	}
}

func TestNewCreateContainerRequestDevices(t *testing.T) {
	req := newCreateContainerRequest("nginx", &ContConfig{
		Devices: []*ContDevice{
			{Host: "/dev/null", Cont: "/dev/nullx", CgroupPerms: "rw"},
			{Host: "/dev/zero"}, // Cont defaults to Host
		},
	})
	devs := req.HostConfig.Devices
	if len(devs) != 2 {
		t.Fatalf("Devices: got %d, want 2", len(devs))
	}
	if devs[0].PathOnHost != "/dev/null" || devs[0].PathInContainer != "/dev/nullx" ||
		devs[0].CgroupPermissions != "rw" {
		t.Errorf("Devices[0]: got %+v", devs[0])
	}
	if devs[1].PathInContainer != "/dev/zero" {
		t.Errorf("Devices[1].PathInContainer (default): got %q, want %q",
			devs[1].PathInContainer, "/dev/zero")
	}
}

func TestNewCreateContainerRequestPortBindings(t *testing.T) {
	req := newCreateContainerRequest("nginx", &ContConfig{
		TCPBinds: []*PortBind{
			{ContPort: 80, HostIP: "0.0.0.0", HostPort: 8080},
		},
		UDPBinds: []*PortBind{
			{ContPort: 53, HostPort: 5353},
		},
	})

	pb := req.HostConfig.PortBindings
	if pb == nil {
		t.Fatal("PortBindings: got nil, want non-nil")
	}
	tcp, ok := pb["80/tcp"]
	if !ok || len(tcp) != 1 || tcp[0].HostIP != "0.0.0.0" || tcp[0].HostPort != "8080" {
		t.Errorf("PortBindings[80/tcp]: got %+v", tcp)
	}
	udp, ok := pb["53/udp"]
	if !ok || len(udp) != 1 || udp[0].HostPort != "5353" {
		t.Errorf("PortBindings[53/udp]: got %+v", udp)
	}

	if _, ok := req.ExposedPorts["80/tcp"]; !ok {
		t.Errorf("ExposedPorts missing 80/tcp: %+v", req.ExposedPorts)
	}
	if _, ok := req.ExposedPorts["53/udp"]; !ok {
		t.Errorf("ExposedPorts missing 53/udp: %+v", req.ExposedPorts)
	}
}

func TestNewCreateContainerRequestJSONLogConfig(t *testing.T) {
	t.Run("max size and file", func(t *testing.T) {
		req := newCreateContainerRequest("nginx", &ContConfig{
			JSONLogConfig: &JSONLogConfig{MaxSize: "10m", MaxFile: 3},
		})
		lc := req.HostConfig.LogConfig
		if lc == nil {
			t.Fatal("LogConfig: got nil")
		}
		if lc.Type != "json-file" {
			t.Errorf("Type: got %q, want %q", lc.Type, "json-file")
		}
		if lc.Config["max-size"] != "10m" || lc.Config["max-file"] != "3" {
			t.Errorf("Config: got %+v", lc.Config)
		}
	})

	t.Run("empty options", func(t *testing.T) {
		req := newCreateContainerRequest("nginx", &ContConfig{
			JSONLogConfig: &JSONLogConfig{},
		})
		lc := req.HostConfig.LogConfig
		if lc == nil {
			t.Fatal("LogConfig: got nil")
		}
		if len(lc.Config) != 0 {
			t.Errorf("Config: got %+v, want empty", lc.Config)
		}
	})
}

func TestNewCreateContainerRequestRestartPolicy(t *testing.T) {
	t.Run("always", func(t *testing.T) {
		req := newCreateContainerRequest("nginx", &ContConfig{AlwaysRestart: true})
		if req.HostConfig.RestartPolicy == nil ||
			req.HostConfig.RestartPolicy.Name != "always" {
			t.Errorf("RestartPolicy: got %+v, want always", req.HostConfig.RestartPolicy)
		}
	})

	t.Run("unless-stopped", func(t *testing.T) {
		req := newCreateContainerRequest("nginx", &ContConfig{AutoRestart: true})
		if req.HostConfig.RestartPolicy == nil ||
			req.HostConfig.RestartPolicy.Name != "unless-stopped" {
			t.Errorf("RestartPolicy: got %+v", req.HostConfig.RestartPolicy)
		}
	})

	t.Run("always wins over auto", func(t *testing.T) {
		req := newCreateContainerRequest("nginx", &ContConfig{
			AlwaysRestart: true,
			AutoRestart:   true,
		})
		if req.HostConfig.RestartPolicy.Name != "always" {
			t.Errorf("RestartPolicy.Name: got %q, want always",
				req.HostConfig.RestartPolicy.Name)
		}
	})

	t.Run("none", func(t *testing.T) {
		req := newCreateContainerRequest("nginx", &ContConfig{})
		if req.HostConfig.RestartPolicy != nil {
			t.Errorf("RestartPolicy: got %+v, want nil", req.HostConfig.RestartPolicy)
		}
	})
}
