package services

import (
	"errors"

	"github.com/vultr/govultr/v3"
)

// GetSSHKey returns the SSH key from Vultr
func (s *Service) GetSSHKey(sshkey string) (*govultr.SSHKey, error) {
	if sshkey == "" {
		return nil, errors.New("missing ssh key")
	}
	key, _, err := s.scope.SSHKEYS.Get(s.ctx, sshkey)
	if err != nil {
		return nil, err
	}

	return key, nil
}
