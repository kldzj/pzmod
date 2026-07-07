package cli

import (
	"fmt"

	"github.com/kldzj/pzmod/pkg/build"
	"github.com/kldzj/pzmod/pkg/domain"
	"github.com/kldzj/pzmod/pkg/store"
	"github.com/spf13/cobra"
)

func newDoctorCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "doctor",
		Short:         "Run health checks on the target config (exits non-zero on errors)",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var checks []doctorCheckJSON

			t, err := resolveTarget(cmd, st)
			if err != nil {
				checks = append(checks, doctorCheckJSON{Name: "target", Status: "error", Detail: err.Error()})
				return finishDoctor(cmd, checks)
			}

			if st.HasAPIKey(t.profileID()) {
				checks = append(checks, doctorCheckJSON{Name: "api-key", Status: "ok", Detail: "Steam API key configured"})
			} else {
				checks = append(checks, doctorCheckJSON{Name: "api-key", Status: "warn", Detail: "no Steam API key; run `pzmod api-key <key>` (validation skipped)"})
			}

			cfg, cerr := t.config()
			if cerr != nil {
				checks = append(checks, doctorCheckJSON{Name: "config", Status: "error", Detail: cerr.Error()})
				return finishDoctor(cmd, checks)
			}
			sm := cfg.ServerMods()
			checks = append(checks, doctorCheckJSON{
				Name:   "config",
				Status: "ok",
				Detail: fmt.Sprintf("%s: %d mod(s), %d item(s), %d map(s)", t.iniPath(), len(sm.Mods), len(sm.WorkshopItems), len(sm.Maps)),
			})

			b := t.build()
			if b == build.Unknown {
				checks = append(checks, doctorCheckJSON{Name: "build", Status: "warn", Detail: "no build declared (b41/b42); compatibility checks limited"})
			} else {
				checks = append(checks, doctorCheckJSON{Name: "build", Status: "ok", Detail: b.Label()})
			}

			offline, _ := cmd.Flags().GetBool("offline")
			switch {
			case offline:
				checks = append(checks, doctorCheckJSON{Name: "validation", Status: "skip", Detail: "--offline"})
			case !st.HasAPIKey(t.profileID()):
				checks = append(checks, doctorCheckJSON{Name: "validation", Status: "skip", Detail: "no API key"})
			default:
				report, verr := t.services(st).Validate(cmd.Context(), sm, b)
				switch {
				case verr != nil:
					checks = append(checks, doctorCheckJSON{Name: "validation", Status: "error", Detail: verr.Error()})
				case report.HasErrors():
					checks = append(checks, doctorCheckJSON{
						Name:   "validation",
						Status: "error",
						Detail: fmt.Sprintf("%d error(s), %d warning(s); run `pzmod validate` for detail", report.Count(domain.SeverityError), report.Count(domain.SeverityWarning)),
					})
				default:
					checks = append(checks, doctorCheckJSON{
						Name:   "validation",
						Status: "ok",
						Detail: fmt.Sprintf("%d warning(s), %d info", report.Count(domain.SeverityWarning), report.Count(domain.SeverityInfo)),
					})
				}
			}

			return finishDoctor(cmd, checks)
		},
	}
	cmd.Flags().Bool("offline", false, "skip checks that need the Steam API")
	addTargetFlags(cmd)
	return cmd
}

// finishDoctor renders the checks (JSON or text) and returns a non-nil error
// when any check failed, so the process exits non-zero.
func finishDoctor(cmd *cobra.Command, checks []doctorCheckJSON) error {
	hasError := false
	for _, c := range checks {
		if c.Status == "error" {
			hasError = true
		}
	}
	if jsonEnabled(cmd) {
		if err := emitJSON(cmd, doctorJSON{Checks: checks, OK: !hasError}); err != nil {
			return err
		}
	} else {
		for _, c := range checks {
			cmd.Printf("%s %s  %s\n", doctorTag(c.Status), c.Name, styleMuted.Render(c.Detail))
		}
		if hasError {
			cmd.Println(styleError.Render("PROBLEMS") + " one or more checks failed")
		} else {
			cmd.Println(styleOK.Render("OK") + " all checks passed")
		}
	}
	if hasError {
		return fmt.Errorf("doctor found problems")
	}
	return nil
}

func doctorTag(status string) string {
	switch status {
	case "ok":
		return styleOK.Render("OK  ")
	case "warn":
		return styleWarn.Render("WARN")
	case "error":
		return styleError.Render("ERR ")
	default:
		return styleMuted.Render("SKIP")
	}
}
