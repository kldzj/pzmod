package version

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/creativeprojects/go-selfupdate"
)

const repoSlug = "kldzj/pzmod"

func GetLatestRelease() (*selfupdate.Release, error) {
	latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.ParseSlug(repoSlug))
	if err != nil {
		return nil, fmt.Errorf("error occurred while fetching release: %w", err)
	}

	if !found {
		return nil, errors.New("latest version could not be found from github repository")
	}

	return latest, nil
}

func IsLatest(current string, latest *selfupdate.Release) bool {
	return latest != nil && latest.LessOrEqual(current)
}

func Update(current string, latest *selfupdate.Release) error {
	if IsLatest(current, latest) {
		fmt.Printf("Current version %s is the latest", current)
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return errors.New("could not locate executable path")
	}

	if err := selfupdate.UpdateTo(context.Background(), latest.AssetURL, latest.AssetName, exe); err != nil {
		return fmt.Errorf("error occurred while updating binary: %w", err)
	}

	fmt.Printf("Successfully updated to version %s", latest.Version())
	return nil
}
