package ec2

import (
	"github.com/aquasecurity/defsec/pkg/providers/aws/ec2"
	defsecTypes "github.com/aquasecurity/defsec/pkg/types"
	parser2 "github.com/aquasecurity/trivy/pkg/iac/scanners/cloudformation/parser"
)

func getInstances(ctx parser2.FileContext) (instances []ec2.Instance) {
	instanceResources := ctx.GetResourcesByType("AWS::EC2::Instance")

	for _, r := range instanceResources {
		instance := ec2.Instance{
			Metadata: r.Metadata(),
			// metadata not supported by CloudFormation at the moment -
			// https://github.com/aws-cloudformation/cloudformation-coverage-roadmap/issues/655
			MetadataOptions: ec2.MetadataOptions{
				Metadata:     r.Metadata(),
				HttpTokens:   defsecTypes.StringDefault("optional", r.Metadata()),
				HttpEndpoint: defsecTypes.StringDefault("enabled", r.Metadata()),
			},
			UserData: r.GetStringProperty("UserData"),
		}

		if launchTemplate, ok := findRelatedLaunchTemplate(ctx, r); ok {
			instance = launchTemplate.Instance
		}

		if instance.RootBlockDevice == nil {
			instance.RootBlockDevice = &ec2.BlockDevice{
				Metadata:  r.Metadata(),
				Encrypted: defsecTypes.BoolDefault(false, r.Metadata()),
			}
		}

		blockDevices := getBlockDevices(r)
		for i, device := range blockDevices {
			copyDevice := device
			if i == 0 {
				instance.RootBlockDevice = copyDevice
				continue
			}
			instance.EBSBlockDevices = append(instance.EBSBlockDevices, device)
		}
		instances = append(instances, instance)
	}

	return instances
}

func findRelatedLaunchTemplate(fctx parser2.FileContext, r *parser2.Resource) (ec2.LaunchTemplate, bool) {
	launchTemplateRef := r.GetProperty("LaunchTemplate.LaunchTemplateName")
	if launchTemplateRef.IsString() {
		res := findLaunchTemplateByName(fctx, launchTemplateRef)
		if res != nil {
			return adaptLaunchTemplate(res), true
		}
	}

	launchTemplateRef = r.GetProperty("LaunchTemplate.LaunchTemplateId")
	if !launchTemplateRef.IsString() {
		return ec2.LaunchTemplate{}, false
	}

	resource := fctx.GetResourceByLogicalID(launchTemplateRef.AsString())
	if resource == nil {
		return ec2.LaunchTemplate{}, false
	}
	return adaptLaunchTemplate(resource), true
}

func findLaunchTemplateByName(fctx parser2.FileContext, prop *parser2.Property) *parser2.Resource {
	for _, res := range fctx.GetResourcesByType("AWS::EC2::LaunchTemplate") {
		templateName := res.GetProperty("LaunchTemplateName")
		if templateName.IsNotString() {
			continue
		}

		if prop.EqualTo(templateName.AsString()) {
			return res
		}
	}

	return nil
}

func getBlockDevices(r *parser2.Resource) []*ec2.BlockDevice {
	var blockDevices []*ec2.BlockDevice

	devicesProp := r.GetProperty("BlockDeviceMappings")

	if devicesProp.IsNil() {
		return blockDevices
	}

	for _, d := range devicesProp.AsList() {
		device := &ec2.BlockDevice{
			Metadata:  d.Metadata(),
			Encrypted: d.GetBoolProperty("Ebs.Encrypted"),
		}

		blockDevices = append(blockDevices, device)
	}

	return blockDevices
}
