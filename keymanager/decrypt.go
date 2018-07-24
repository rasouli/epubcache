package keymanager

import (
	"os/exec"
)

func DecryptFile(in string, out string, key string) error {
	cmd := "openssl"

	cmdArgs := []string{"aes-128-cbc", "-d" , "-K", key, "-iv", key, "-in", in, "-out", out}
	err := exec.Command(cmd, cmdArgs...).Run()


	return err
}

func EncryptFile(in string, out string, key string) error {
	cmd := "openssl"

	cmdArgs := []string{"aes-128-cbc" , "-K", key, "-iv", key, "-in", in, "-out", out}
	err := exec.Command(cmd, cmdArgs...).Run()


	return err
}
