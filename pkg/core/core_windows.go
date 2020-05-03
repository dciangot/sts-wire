package core

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
)

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func DownloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func DownloadRClone() error {

	_, err := os.Stat("./rclone.exe")
	if os.IsNotExist(err) {

		fileUrl := "https://golangcode.com/images/avatar.jpg"

		if err := DownloadFile("rclone.exe", fileUrl); err != nil {
			return err
		}
	}

	return nil
}

func MountVolume(instance string, remotePath string, localPath string, configPath string) error {

	err := DownloadRClone()
	if err != nil {
		return err
	}

	conf := fmt.Sprintf("%s:%s", instance, remotePath)

	grepCmd := exec.Command(
		".\\rclone.exe",
		"--config",
		configPath+"/rclone.conf",
		"--no-check-certificate",
		"mount",
		//"--daemon",
		"--log-file",
		configPath + "/rclone.log",
		"--log-level=DEBUG",
		"--vfs-cache-mode",
		"full",
		"--no-modtime",
		conf,
		localPath,
	)

	grepCmd.Start()

	return nil
}
