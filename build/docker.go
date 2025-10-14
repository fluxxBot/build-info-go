package build

import (
	"encoding/json"
	"errors"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/jfrog/gofrog/crypto"
	"os"
	"os/exec"
	"strings"
)

type DockerModule struct {
	containingBuild *Build
	imageName       string
	dockerArgs      []string
	username        string
	password        string
}

func newDockerModule(imagePath string, dockerArgs []string, containingBuild *Build) (*DockerModule, error) {
	return &DockerModule{
		imageName:       imagePath,
		containingBuild: containingBuild,
		dockerArgs:      dockerArgs,
	}, nil
}

func (dm *DockerModule) Build() error {
	if len(dm.dockerArgs) > 0 {

		command := exec.Command("docker", dm.dockerArgs...)
		output, err := command.CombinedOutput()
		if err != nil {
			return err
		}
		if len(output) > 0 {
			dm.containingBuild.logger.Output(strings.TrimSpace(string(output)))
		}
	}
	if !dm.containingBuild.buildNameAndNumberProvided() {
		return nil
	}
	return nil
}

func (dm *DockerModule) CalcDependencies() error {
	if dm.containingBuild.buildNameAndNumberProvided() {
		return errors.New("a build name must be provided in order to collect the image's dependencies")
	}
	imageManifest, manifestChecksum, err := getManifest(dm.imageName)
	if err != nil {
		return err
	}

	var dockerLayers []string
	for _, layer := range imageManifest.Layers {
		dockerLayers = append(dockerLayers, layer.Digest.String())
	}
	return nil
}

func getManifest(imageName string) (v1.Manifest, crypto.Checksum, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return v1.Manifest{}, crypto.Checksum{}, err
	}
	remoteImgDesc, err := remote.Get(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return v1.Manifest{}, crypto.Checksum{}, err
	}
	var imageManifest v1.Manifest
	err = json.Unmarshal(remoteImgDesc.Manifest, &imageManifest)
	if err != nil {
		return v1.Manifest{}, crypto.Checksum{}, errors.New("failed unmarshalling manifest.json content")
	}
	tempFileName := "manifest-" + imageName + ".json"
	tmpFile, err := os.CreateTemp("", tempFileName)
	if err != nil {
		return v1.Manifest{}, crypto.Checksum{}, errors.New("failed creating manifest.json file in local")
	}
	defer func(tmpFile *os.File) {
		err := tmpFile.Close()
		if err != nil {
			return
		}
	}(tmpFile)
	if _, err := tmpFile.Write(remoteImgDesc.Manifest); err != nil {
		return v1.Manifest{}, crypto.Checksum{}, errors.New("failed to write content in manifest.json file in local")
	}

	checksum, err := crypto.CalcChecksumDetails("./" + tempFileName)
	if err != nil {
		return v1.Manifest{}, crypto.Checksum{}, errors.New("failed calculating checksum for manifest.json file")
	}
	_ = os.Remove("./" + tempFileName)

	return imageManifest, checksum, nil
}

func searchForLayers() {

}
