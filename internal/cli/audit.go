package cli

import (
"encoding/json"
"fmt"
"os"

"github.com/nathfavour/auracrab/pkg/security"
"github.com/spf13/cobra"
)

var (
auditFormat string
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Run security audit on the system and ecosystem",
	Run: func(cmd *cobra.Command, args []string) {
		report, err := security.RunAudit()
		if err != nil {
			fmt.Printf("Audit failed: %v\n", err)
			os.Exit(1)
		}

		if auditFormat == "json" {
			data, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(data))
			return
		}

		fmt.Println("=== Auracrab Security Audit Report ===")
		if len(report.Findings) == 0 {
			fmt.Println("✅ No findings. Your system looks system-intimate and secure.")
			return
		}

		for _, f := range report.Findings {
			severityChar := "ℹ️"
			if f.Severity == security.SeverityWarn {
				severityChar = "⚠️"
			} else if f.Severity == security.SeverityCritical {
				severityChar = "🚨"
			}
			fmt.Printf("%s [%s] %s\n", severityChar, f.Severity, f.Title)
			fmt.Printf("   Desc: %s\n", f.Description)
			if f.Remediation != "" {
				fmt.Printf("   Fix:  %s\n", f.Remediation)
			}
			fmt.Println()
		}
	},
}

func init() {
	auditCmd.Flags().StringVarP(&auditFormat, "format", "f", "text", "Output format (text, json)")
	rootCmd.AddCommand(auditCmd)
}
