package cli

import (
	"errors"
	"path/filepath"

	"github.com/kldzj/pzmod/internal/build"
	"github.com/kldzj/pzmod/internal/pathutil"
	"github.com/kldzj/pzmod/internal/serverconfig"
	"github.com/kldzj/pzmod/internal/service"
	"github.com/kldzj/pzmod/internal/steam"
	"github.com/kldzj/pzmod/internal/store"
	"github.com/spf13/cobra"
)

// errNoKey is returned by commands that need the Steam API but have no key.
var errNoKey = errors.New("no Steam API key set - run `pzmod api-key <key>`")

// steamFactory builds the Steam API client from a key. It is a package var so
// tests can substitute a fake.
var steamFactory = func(key string) steam.API { return steam.New(key) }

// target identifies which server config a command operates on, resolved from
// --file, --profile, or the default profile.
type target struct {
	profile store.Profile // for ad-hoc --file this is a synthetic profile
	adHoc   bool          // true when resolved from --file rather than a stored profile
}

func (t target) iniPath() string    { return t.profile.IniPath }
func (t target) profileID() string  { return t.profile.ID }
func (t target) build() build.Build { return build.Parse(t.profile.Build) }

// addTargetFlags adds the --file/--profile selector flags to a command.
func addTargetFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("file", "f", "", "path to a servertest.ini (ad-hoc, no profile)")
	cmd.Flags().StringP("profile", "p", "", "named profile to operate on")
	cmd.MarkFlagFilename("file", "ini")
}

// resolveTarget picks the target from flags, falling back to the default profile.
func resolveTarget(cmd *cobra.Command, st *store.Store) (target, error) {
	file, _ := cmd.Flags().GetString("file")
	profileName, _ := cmd.Flags().GetString("profile")

	switch {
	case file != "":
		abs := pathutil.Expand(file)
		return target{
			adHoc:   true,
			profile: store.Profile{ID: store.EphemeralProfileID(abs), Name: filepath.Base(abs), IniPath: abs},
		}, nil
	case profileName != "":
		p, err := st.Profile(profileName)
		if err != nil {
			return target{}, err
		}
		return target{profile: p}, nil
	default:
		p, err := st.DefaultProfile()
		if err != nil {
			return target{}, errors.New("no --file or --profile given and no default profile is set")
		}
		return target{profile: p}, nil
	}
}

// config loads the target's servertest.ini.
func (t target) config() (*serverconfig.Config, error) {
	return serverconfig.Load(t.iniPath())
}

// services builds a Services aggregate for the target. The steam client is
// constructed with the resolved key (which may be empty for offline commands).
func (t target) services(st *store.Store) *service.Services {
	key, _ := st.APIKey(t.profileID())
	return service.New(steamFactory(key), st)
}

// servicesWithSteam is like services but errors when no API key is configured,
// for commands that must reach the Steam API.
func (t target) servicesWithSteam(st *store.Store) (*service.Services, error) {
	if !st.HasAPIKey(t.profileID()) {
		return nil, errNoKey
	}
	return t.services(st), nil
}
