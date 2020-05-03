package core

import (
	"fmt"

	execute "github.com/alexellis/go-execute/pkg/v1"
)

func DownloadRClone() error {

	return nil
}

func MountVolume(instance string, remotePath string, localPath string, configPath string) error {

	conf := fmt.Sprintf("%s:%s", instance, remotePath)

	args := []string{
		"--config",
		configPath,
		"--no-check-certificate",
		"mount",
		"--log-file",
		"rclone.log",
		"--log-level=DEBUG",
		"--vfs-cache-mode",
		"full",
		"--no-modtime",
		conf,
		localPath,
	}

	cmd := execute.ExecTask{
		Command:     "rclone",
		Args:        args,
		StreamStdio: true,
	}

	res, err := cmd.Execute()
	if err != nil {
		panic(err)
	}

	if res.ExitCode != 0 {
		panic("Non-zero exit code: " + res.Stderr)
	}

	return nil
}
