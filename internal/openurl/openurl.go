// Package openurl opens a URL in the user's default browser, best-effort.
package openurl

import "os/exec"

// Command returns the executable + args to open url on the given GOOS.
func Command(goos, url string) (string, []string) {
	switch goos {
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		return "open", []string{url}
	default:
		return "xdg-open", []string{url}
	}
}

// Open launches the default browser for url. It does not wait for the browser.
func Open(url string) error {
	name, args := Command(runtimeGOOS(), url)
	return exec.Command(name, args...).Start()
}
