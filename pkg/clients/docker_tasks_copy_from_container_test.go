package clients

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	clients "github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var fileContent = `kubeconfig.yam@@@@i@@@@@@@@@@@@@@@@@@@@@@2@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@lapiVersion: v1`

func TestCopyFromContainerCopiesFile(t *testing.T) {
	id := "abc"
	src := "/output/file.hcl"

	md := &mocks.MockDocker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)

	mic := &clients.ImageLog{}
	md.On("CopyFromContainer", mock.Anything, id, src).Return(
		ioutil.NopCloser(bytes.NewBufferString(fileContent)),
		types.ContainerPathStat{},
		nil,
	)
	dt := NewDockerTasks(md, mic, &TarGz{}, clients.NewTestLogger(t))

	tmpDir, _ := ioutil.TempDir("", "")
	defer os.RemoveAll(tmpDir)

	err := dt.CopyFromContainer(id, src, tmpDir+"/new.hcl")
	assert.NoError(t, err)

	// check the file was written correctly
	d, err := ioutil.ReadFile(tmpDir + "/new.hcl")
	assert.NoError(t, err)
	assert.Equal(t, ": v1", string(d))
}

func TestCopyFromContainerReturnsErrorOnDockerError(t *testing.T) {
	id := "abc"
	src := "/output/file.hcl"

	md := &mocks.MockDocker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)

	mic := &clients.ImageLog{}
	md.On("CopyFromContainer", mock.Anything, id, src).Return(
		nil,
		types.ContainerPathStat{},
		fmt.Errorf("boom"),
	)
	dt := NewDockerTasks(md, mic, &TarGz{}, clients.NewTestLogger(t))

	err := dt.CopyFromContainer(id, src, "/new.hcl")
	assert.Error(t, err)
}
