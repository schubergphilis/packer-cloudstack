package cloudstack

import (
	"code.google.com/p/go.crypto/ssh"
	"fmt"
	"github.com/mitchellh/multistep"
	commonssh "github.com/mitchellh/packer/common/ssh"
	packerssh "github.com/mitchellh/packer/communicator/ssh"
)

func sshAddress(state multistep.StateBag) (string, error) {
	config := state.Get("config").(config)
	ipAddress := state.Get("virtual_machine_ip").(string)
	return fmt.Sprintf("%s:%d", ipAddress, config.SSHPort), nil
}

func sshConfig(state multistep.StateBag) (*ssh.ClientConfig, error) {
	config := state.Get("config").(config)
	privateKey := state.Get("ssh_private_key").(string)

	auth := []ssh.AuthMethod{
		ssh.Password(config.SSHPassword),
		ssh.KeyboardInteractive(
			packerssh.PasswordKeyboardInteractive(config.SSHPassword)),
	}

	if privateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return nil, fmt.Errorf("Error setting up SSH config: %s", err)
		}

		auth = append(auth, ssh.PublicKeys(signer))
	}

	return &ssh.ClientConfig{
		User: config.SSHUsername,
		Auth: auth,
	}, nil
}
