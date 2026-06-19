package ecosystem

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	anyislandInstallURL = "https://raw.githubusercontent.com/nathfavour/anyisland/master/install.sh"
	polygeistEnvMarker  = "# polygeist agentic stack"
)

// EnsureAnyisland installs anyisland when it is not already available.
func EnsureAnyisland() error {
	if _, err := exec.LookPath("anyisland"); err == nil {
		return nil
	}
	fmt.Println("Installing anyisland (package manager)...")
	cmd := exec.Command("bash", "-c", fmt.Sprintf("curl -fsSL %s | bash", anyislandInstallURL))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Install installs a library by anyisland package name.
func Install(name string) error {
	lib, ok := Lookup(name)
	if !ok {
		return fmt.Errorf("unknown library %q (run auracrab stack list)", name)
	}
	return installPackage(lib)
}

// InstallPolygeistStack installs polygeist and its runtime configuration.
func InstallPolygeistStack() error {
	if err := EnsureAnyisland(); err != nil {
		return fmt.Errorf("anyisland bootstrap: %w", err)
	}
	lib, _ := Lookup("polygeist")
	if err := installPackage(lib); err != nil {
		return err
	}
	return ConfigurePolygeistRuntime()
}

func installPackage(lib Library) error {
	if err := EnsureAnyisland(); err != nil {
		return fmt.Errorf("anyisland bootstrap: %w", err)
	}

	if IsInstalled(lib) {
		fmt.Printf("%s is already installed (%s)\n", lib.Name, lib.Binary)
		return nil
	}

	fmt.Printf("Installing %s via anyisland...\n", lib.Name)
	cmd := exec.Command("anyisland", "install", lib.Package)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("anyisland install %s: %w", lib.Package, err)
	}
	return nil
}

// ConfigurePolygeistRuntime writes UDS env under ~/.config/polygeist/env.
func ConfigurePolygeistRuntime() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	installDir := filepath.Join(home, ".local", "bin")
	runDir := filepath.Join(home, ".polygeist", "run")
	configDir := filepath.Join(home, ".config", "polygeist")
	envFile := filepath.Join(configDir, "env")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return err
	}

	content := fmt.Sprintf(`# Source this file: . "%s"
export AGENTIC_RUN_DIR="%s"
export ANYISLAND_SOCKET="%s/anyisland.sock"
export VIBEAURA_SOCKET="%s/vibeaura.sock"
export POLYGEIST_SOCKET="%s/polygeist.sock"
export ANYISLAND_BIN_DIR="%s"
export PATH="%s:${PATH}"
`, envFile, runDir, runDir, runDir, runDir, installDir, installDir)

	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		return err
	}

	profile := filepath.Join(home, ".profile")
	if _, err := os.Stat(filepath.Join(home, ".zshrc")); err == nil {
		profile = filepath.Join(home, ".zshrc")
	}

	data, _ := os.ReadFile(profile)
	if !strings.Contains(string(data), polygeistEnvMarker) {
		f, err := os.OpenFile(profile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, _ = fmt.Fprintf(f, "\n%s\n. \"%s\"\n", polygeistEnvMarker, envFile)
	}

	fmt.Printf("\nPolygeist runtime configured.\n  env: %s\n  UDS: %s\n", envFile, runDir)
	fmt.Println("\nStart daemons:")
	fmt.Println("  anyisland daemon start")
	fmt.Println("  vibeaura daemon start")
	fmt.Println("  polygeist --once \"hello\" --workdir .")
	fmt.Printf("\nRestart your shell or run: . \"%s\"\n", envFile)
	return nil
}
