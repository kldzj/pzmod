package version

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/creativeprojects/go-selfupdate"
)

const repoSlug = "kldzj/pzmod"

func NewUpdater() (*selfupdate.Updater, error) {
	return selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{
			UniqueFilename: "checksums.txt",
		},
	})
}

func GetLatestRelease(updater *selfupdate.Updater) (*selfupdate.Release, error) {
	latest, found, err := updater.DetectLatest(context.Background(), selfupdate.ParseSlug(repoSlug))
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

func Update(current string, latest *selfupdate.Release, updater *selfupdate.Updater) error {
	if IsLatest(current, latest) {
		fmt.Printf("Current version %s is the latest", current)
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return errors.New("could not locate executable path")
	}

	if err := updater.UpdateTo(context.Background(), latest, exe); err != nil {
		return fmt.Errorf("error occurred while updating binary: %w", err)
	}

	fmt.Printf("Successfully updated to version %s", latest.Version())
	return nil
}
