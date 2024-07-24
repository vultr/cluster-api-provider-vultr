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

	s.scope.V(2).Info("fetching SSH key", "sshkey_id", sshkey)
	key, _, err := s.scope.SSHKeys.Get(s.ctx, sshkey)
	if err != nil {
		s.scope.V(2).Info("error fetching SSH key", "sshkey_id", sshkey, "error", err)
		return nil, err
	}

	s.scope.V(2).Info("successfully fetched SSH key", "sshkey_id", sshkey)
	return key, nil
}
