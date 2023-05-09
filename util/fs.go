package util

import "os"

func FileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func StoreCredentials(apiKey string) error {
	path, err := GetCredentialsPath()
	if err != nil {
		return err
	}

	err = os.WriteFile(path, []byte(apiKey), 0644)
	if err != nil {
		return err
	}

	return nil
}

func LoadCredentials() (string, error) {
	path, err := GetCredentialsPath()
	if err != nil {
		return "", err
	}

	if !FileExists(path) {
		return "", ErrNoCredFile
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func DeleteCredentials() error {
	path, err := GetCredentialsPath()
	if err != nil {
		return err
	}

	if !FileExists(path) {
		return nil
	}

	err = os.Remove(path)
	if err != nil {
		return err
	}

	return nil
}

func GetCredentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return home + "/.pzmod", nil
}
