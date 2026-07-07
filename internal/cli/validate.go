package cli

import (
	"fmt"

	"github.com/kldzj/pzmod/pkg/domain"
	"github.com/kldzj/pzmod/pkg/store"
	"github.com/spf13/cobra"
)

func newValidateCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "validate",
		Short:         "Validate mods and dependencies (exits non-zero on errors)",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := resolveTarget(cmd, st)
			if err != nil {
				return err
			}
			svc, err := t.servicesWithSteam(st)
			if err != nil {
				return err
			}
			cfg, err := t.config()
			if err != nil {
				return err
			}

			report, err := svc.Validate(cmd.Context(), cfg.ServerMods(), t.build())
			if err != nil {
				return err
			}

			findings := report.Sorted()

			if jsonEnabled(cmd) {
				out := validateJSON{Findings: make([]findingJSON, 0, len(findings)), OK: !report.HasErrors()}
				for _, f := range findings {
					out.Findings = append(out.Findings, findingJSON{
						Severity: f.Severity.String(),
						Code:     f.Code,
						Subject:  f.Subject,
						Message:  f.Message,
					})
				}
				out.Summary.Errors = report.Count(domain.SeverityError)
				out.Summary.Warnings = report.Count(domain.SeverityWarning)
				out.Summary.Info = report.Count(domain.SeverityInfo)
				if err := emitJSON(cmd, out); err != nil {
					return err
				}
				if report.HasErrors() {
					return fmt.Errorf("validation failed with %d error(s)", report.Count(domain.SeverityError))
				}
				return nil
			}

			if len(findings) == 0 {
				cmd.Println(styleOK.Render("OK") + " no problems found")
				return nil
			}
			for _, f := range findings {
				cmd.Printf("%s %s\n", severityTag(f.Severity), f.Message)
			}
			cmd.Printf("\n%d error(s), %d warning(s), %d info\n",
				report.Count(domain.SeverityError),
				report.Count(domain.SeverityWarning),
				report.Count(domain.SeverityInfo))

			if report.HasErrors() {
				return fmt.Errorf("validation failed with %d error(s)", report.Count(domain.SeverityError))
			}
			return nil
		},
	}
	addTargetFlags(cmd)
	return cmd
}
