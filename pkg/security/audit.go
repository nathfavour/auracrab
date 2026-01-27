package security

import (
"fmt"
"os"
"os/user"
"path/filepath"
"runtime"
)

type Severity string

const (
SeverityInfo     Severity = "info"
SeverityWarn     Severity = "warn"
SeverityCritical Severity = "critical"
)

type Finding struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Severity    Severity `json:"severity"`
	Remediation string   `json:"remediation,omitempty"`
}

type AuditReport struct {
	Findings []Finding `json:"findings"`
}

func RunAudit() (*AuditReport, error) {
	report := &AuditReport{
		Findings: make([]Finding, 0),
	}

	if err := auditFilesystem(report); err != nil {
		return nil, err
	}

	auditEnvironment(report)
	auditEcosystem(report)

	return report, nil
}

func auditFilesystem(report *AuditReport) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	targets := []string{
		filepath.Join(home, ".local", "share", "auracrab"),
		filepath.Join(home, ".vibeauracle"),
	}

	for _, target := range targets {
		info, err := os.Stat(target)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}

		mode := info.Mode()
		if mode&0002 != 0 {
			report.Findings = append(report.Findings, Finding{
ID:          "fs.world_writable",
Title:       "Sensitive Directory is World-Writable",
Description: fmt.Sprintf("%s has permissions %o", target, mode.Perm()),
				Severity:    SeverityCritical,
				Remediation: fmt.Sprintf("chmod 700 %s", target),
			})
		}
		if mode&0020 != 0 {
			report.Findings = append(report.Findings, Finding{
ID:          "fs.group_writable",
Title:       "Sensitive Directory is Group-Writable",
Description: fmt.Sprintf("%s has permissions %o", target, mode.Perm()),
				Severity:    SeverityWarn,
				Remediation: fmt.Sprintf("chmod 700 %s", target),
			})
		}
	}
	return nil
}

func auditEnvironment(report *AuditReport) {
	if runtime.GOOS == "linux" {
		currUser, err := user.Current()
		if err == nil && currUser.Uid == "0" {
			report.Findings = append(report.Findings, Finding{
ID:          "env.root",
Title:       "Running as Root",
Description: "Auracrab is running as root. This increases the blast radius.",
Severity:    SeverityWarn,
Remediation: "Run as a non-privileged user.",
})
		}
	}

	sensitiveKeys := []string{"GITHUB_TOKEN", "OPENAI_API_KEY", "ANTHROPIC_API_KEY"}
	for _, key := range sensitiveKeys {
		if val := os.Getenv(key); val != "" {
			report.Findings = append(report.Findings, Finding{
ID:          "env.sensitive_var",
Title:       fmt.Sprintf("Sensitive Env Var: %s", key),
Description: fmt.Sprintf("%s is set in the environment.", key),
Severity:    SeverityInfo,
Remediation: "Consider using a secure vault.",
})
		}
	}
}

func auditEcosystem(report *AuditReport) {
	// check if vibeauracle is misconfigured
}
