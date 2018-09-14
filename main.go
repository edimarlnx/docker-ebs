package main

import (
	"fmt"
	"log"

	// With fix: No such volume error
	"github.com/edimarlnx/go-plugins-helpers/volume"
)

func main() {
	dockerEbsVolume, err := newDockerEbs("/mnt")
	if err != nil {
		fmt.Println("Erro ao criar o volume", err)
	}
	vol := volume.NewHandler(dockerEbsVolume)
	log.Println("listening on", socketAddress)
	log.Println(vol.ServeUnix(socketAddress, 0))
}
