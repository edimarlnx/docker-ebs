package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	// With fix: No such volume error
	"github.com/edimarlnx/go-plugins-helpers/volume"
)

const (
	_ErrorVolumeNotFound       = "Volume not found %s"
	_ErrorMountpointNotAllowed = "Mountpoint not allowed %s"
	_pluginName                = "docker-ebs"
)

type dockerEbsVolume struct {
	VolumeID   string
	IntanceID  string
	Mountpoint string
	Device     string
	Containers []string
}

type dockerEbs struct {
	m               sync.RWMutex
	rootMount       string
	Options         []string
	statePath       string
	ec2serverConfig *EC2ServerConfig
	volumes         map[string]*dockerEbsVolume
}

const socketAddress = "/run/docker/plugins/" + _pluginName + "-volume.sock"

func newDockerEbs(root string) (*dockerEbs, error) {
	d := &dockerEbs{
		rootMount:       filepath.Join(root, _pluginName, "volumes"),
		volumes:         map[string]*dockerEbsVolume{},
		statePath:       filepath.Join(root, _pluginName, "state"),
		ec2serverConfig: EC2ServerConfigNew(),
	}
	d.saveState()
	return d, nil
}

func (dv *dockerEbsVolume) addContainer(ID string) {
	var exists = false
	for _, val := range dv.Containers {
		if ID == val {
			exists = true
		}
	}
	if !exists {
		dv.Containers = append(dv.Containers, ID)
	}
}

func (dv *dockerEbsVolume) removeContainer(ID string) {
	var containers []string
	var exists = false
	for _, val := range dv.Containers {
		if ID != val {
			containers = append(containers, val)
		} else {
			exists = true
		}
	}
	if exists {
		dv.Containers = containers
	}
}

func (d *dockerEbs) Create(r *volume.CreateRequest) error {
	d.m.Lock()
	defer d.m.Unlock()

	vol := &dockerEbsVolume{}

	for key, val := range r.Options {
		switch key {
		case "volume-id":
			vol.VolumeID = val
		}
	}

	if vol.VolumeID == "" {
		return logError("Option volume-id is required.")
	}

	_, volumeMountDesc, err := mountVolume(vol.VolumeID, d.ec2serverConfig)
	fmt.Println(volumeMountDesc)
	if err != nil {
		if volumeMountDesc == nil {
			return logError("Not attach volume to instance.", err.Error())
		}
	}

	vol.Device = volumeMountDesc.virtualDevice
	vol.Mountpoint = filepath.Join(d.rootMount, vol.VolumeID)
	vol.Containers = []string{}

	d.volumes[r.Name] = vol
	d.saveState()
	return nil
}

func (d *dockerEbs) List() (*volume.ListResponse, error) {
	d.m.RLock()
	defer d.m.RUnlock()
	var vols []*volume.Volume
	for name, v := range d.volumes {
		vols = append(vols, &volume.Volume{Name: name, Mountpoint: v.Mountpoint})
	}
	return &volume.ListResponse{Volumes: vols}, nil
}

func (d *dockerEbs) Get(r *volume.GetRequest) (*volume.GetResponse, error) {
	d.m.RLock()
	defer d.m.RUnlock()

	vol, ok := d.volumes[r.Name]
	if !ok {
		return &volume.GetResponse{}, logError(_ErrorVolumeNotFound, r.Name)
	}

	volRes := &volume.Volume{Name: r.Name}
	volRes.Mountpoint = vol.Mountpoint

	return &volume.GetResponse{Volume: volRes}, nil
}

func (d *dockerEbs) Remove(r *volume.RemoveRequest) error {
	log.Println("Remove", r)
	//TODO Implementar
	d.saveState()
	return nil
}

func (d *dockerEbs) Path(r *volume.PathRequest) (*volume.PathResponse, error) {
	d.m.RLock()
	defer d.m.RUnlock()

	vol, ok := d.volumes[r.Name]
	if !ok {
		return &volume.PathResponse{}, logError(_ErrorVolumeNotFound, r.Name)
	}

	return &volume.PathResponse{Mountpoint: vol.Mountpoint}, nil
}

func (d *dockerEbs) Mount(r *volume.MountRequest) (*volume.MountResponse, error) {
	d.m.Lock()
	defer d.m.Unlock()

	vol, ok := d.volumes[r.Name]
	if !ok {
		return &volume.MountResponse{}, logError(_ErrorVolumeNotFound, r.Name)
	}
	_, err := os.Lstat(vol.Mountpoint)
	if os.IsNotExist(err) {
		if err := createPathIfNotExist(vol.Mountpoint, 0755); err != nil {
			return &volume.MountResponse{}, logError(err.Error())
		}
	}

	if !strings.HasPrefix(vol.Mountpoint, d.rootMount) {
		return &volume.MountResponse{}, logError(_ErrorMountpointNotAllowed, vol.Mountpoint)
	}

	if err := syscall.Mount(vol.Device, vol.Mountpoint, "ext4", 0, ""); err != nil {
		return &volume.MountResponse{}, logError("No mount device: %s", err.Error())
	}

	vol.addContainer(r.ID)
	d.saveState()
	return &volume.MountResponse{Mountpoint: vol.Mountpoint}, nil
}

func (d *dockerEbs) Unmount(r *volume.UnmountRequest) error {
	log.Println("Unmount", r)
	d.saveState()
	return nil
}

func (d *dockerEbs) Capabilities() *volume.CapabilitiesResponse {
	return &volume.CapabilitiesResponse{Capabilities: volume.Capability{Scope: "local"}}
}

func (d *dockerEbs) saveState() {
	data, err := json.Marshal(d.volumes)
	if err != nil {
		logError(err.Error())
		return
	}
	if err := createPathIfNotExist(filepath.Base(d.statePath), 0644); err != nil {
		logError(err.Error())
	}
	if err := ioutil.WriteFile(d.statePath, data, 0644); err != nil {
		logError(err.Error())
	}
}
