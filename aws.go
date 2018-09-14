package main

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2ServerConfig struct {
	region     string
	instanceID string
}

type VolumeMountDesc struct {
	device        string
	virtualDevice string
}

func EC2ServerConfigNew() *EC2ServerConfig {
	ec2meta := ec2metadata.New(session.New())
	ec2serverConfig := &EC2ServerConfig{}
	ec2serverConfig.region, _ = ec2meta.Region()
	ec2serverConfig.instanceID = getFromEC2Metadata(ec2meta, "instance-id")
	return ec2serverConfig
}

func getFromEC2Metadata(ec2metadata *ec2metadata.EC2Metadata, value string) string {
	val, err := ec2metadata.GetMetadata(value)
	if err != nil {
		return err.Error()
	}
	return val
}

func getInstanceAttributes(ec2Srv *ec2.EC2, instanceID string, attribute string) (*ec2.DescribeInstanceAttributeOutput, error) {
	iAttr, err := ec2Srv.DescribeInstanceAttribute(&ec2.DescribeInstanceAttributeInput{
		InstanceId: &instanceID,
		Attribute:  &attribute,
	})
	if err != nil {
		return nil, err
	}
	return iAttr, nil
}

func getNextDevice(ec2Srv *ec2.EC2, instanceID string, volumeID string) (*VolumeMountDesc, error) {
	iAttr, err := getInstanceAttributes(ec2Srv, instanceID, "blockDeviceMapping")
	if err != nil {
		return nil, fmt.Errorf("Intance attributes not load. %s", err.Error())
	}
	letter := "f"
	for _, blk := range iAttr.BlockDeviceMappings {
		deviceName := *blk.DeviceName
		if strings.Compare(*blk.Ebs.VolumeId, volumeID) == 0 {
			return &VolumeMountDesc{
				device:        "/dev/sd" + letter,
				virtualDevice: "/dev/xvd" + letter,
			}, fmt.Errorf("Volume id has been monted: %s", volumeID)
		}
		deviceNameEnd := string(deviceName[len(deviceName)-1])
		if deviceNameEnd[0] >= letter[0] {
			letter = string(deviceNameEnd[0] + 1)
		}

	}
	return &VolumeMountDesc{
		device:        "/dev/sd" + letter,
		virtualDevice: "/dev/xvd" + letter,
	}, nil
}

func mountVolume(volumeID string, ec2ServerConfig *EC2ServerConfig) (*ec2.VolumeAttachment, *VolumeMountDesc, error) {
	ec2Srv := ec2.New(session.New(), &aws.Config{
		Region: &ec2ServerConfig.region,
	})

	device, err := getNextDevice(ec2Srv, ec2ServerConfig.instanceID, volumeID)
	if err != nil {
		return nil, device, err
	}

	avi := &ec2.AttachVolumeInput{
		VolumeId:   &volumeID,
		InstanceId: &ec2ServerConfig.instanceID,
		Device:     &device.device,
	}

	attachedVolumeOut, err := ec2Srv.AttachVolume(avi)
	if err != nil {
		return nil, nil, err
	}

	// err = ec2Srv.WaitUntilVolumeInUse(&ec2.DescribeVolumesInput{
	// 	VolumeIds: []*string{&volumeID},
	// })
	if err != nil {
		return nil, nil, err
	}

	return attachedVolumeOut, device, nil
}
