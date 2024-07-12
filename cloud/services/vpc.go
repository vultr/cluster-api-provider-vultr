package services

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/vultr/govultr/v3"
)

// GetVPC retrieves a VPC by VPC ID.
func (s *Service) GetVPC(vpcid string) (*govultr.VPC, error) {
	if vpcid == "" {
		return nil, nil
	}

	s.scope.Logger.V(2).Info("Looking for VPC ID", "vpc-id", vpcid)

	vpc, res, err := s.scope.VPCs.Get(s.ctx, vpcid)
	if err != nil {
		if res != nil && res.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get instance with ID %q", vpcid)
	}

	return vpc, nil
}

// ListVPCs lists all VPCs.
func (s *Service) ListVPCs() ([]govultr.VPC, error) {
	vpcs, _, _, err := s.scope.VPCs.List(s.ctx, nil)
	if err != nil {
		return nil, err
	}
	return vpcs, nil
}

// GetDefaultVPC retrieves the VPC with the description "Default private network".
func (s *Service) GetDefaultVPC() (*govultr.VPC, error) {
	s.scope.Logger.V(2).Info("Looking for the default VPC with description 'Default private network'")

	// Create a context and list options
	ctx := context.Background()
	options := &govultr.ListOptions{}

	// Call the List function with the context and options
	vpcs, _, _, err := s.scope.VPCs.List(ctx, options)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list VPCs")
	}

	// Iterate through the VPCs to find the one with the desired description
	for _, vpc := range vpcs {
		if vpc.Description == "Default private network" {
			s.scope.Logger.V(2).Info("Found default VPC", "vpc-id", vpc.ID)
			return &vpc, nil
		}
	}

	return nil, errors.New("no default VPC found with description 'Default private network'")
}

// // Get the default VPC
// defaultVPC, err := s.GetDefaultVPC()
// if err != nil {
// 	log.Error(err, "Failed to get default VPC")
// 	return nil, errors.Wrap(err, "failed to get default VPC")
// }

// // Attach the instance to the default VPC
// s.scope.V(2).Info("Attaching instance to default VPC", "instance-id", instance.ID, "vpc-id", defaultVPC.ID)
// err = s.scope.Instances.AttachVPC(s.ctx, instance.ID, defaultVPC.ID)
// if err != nil {
// 	log.Error(err, "Failed to attach default VPC to instance", "instance-id", instance.ID, "vpc-id", defaultVPC.ID)
// 	return nil, errors.Wrap(err, "failed to attach default VPC to instance")
// }

// return instance, nil
