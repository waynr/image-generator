package image

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	// letter* variables are used in RandomImageFactory.randBytes to efficiently
	// produce a slice of random bytes for use in generating a random pool of
	// files to be added to randomly-generated docker images.
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits

	fileSizeKB = 50
)

var (
	imageDir = "./generated-files"
)

// RandomImageFactory generates random images with the specified layer size and
// count parameters.
type RandomImageFactory struct {
	LayerSizeKB uint
	LayerCount  uint
	Seed        int64
	Tags        []string

	dockerClient      *client.Client
	allGeneratedFiles []string
}

// GenerateImage generates unique files filled with random bytes then uses
// those files to build a docker image with layers filled using the
// randomly-generated files according to the random layer count and layer size
// parameters specified in RandomImageFactory
func (f *RandomImageFactory) GenerateImage() error {
	err := f.generateRandomFilePool()
	if err != nil {
		return err
	}

	f.shuffleGeneratedFilePaths()

	dockerfilePath, err := f.generateDockerfile(f.allGeneratedFiles)
	if err != nil {
		return err
	}

	tarFile := path.Join(os.Getenv("PWD"), "context.tar")
	files := append(f.allGeneratedFiles, dockerfilePath)
	err = f.createArchive(tarFile, files)
	if err != nil {
		return err
	}

	cli, err := f.getDockerClient()
	if err != nil {
		return err
	}

	buildContext, err := os.Open(tarFile)
	if err != nil {
		return fmt.Errorf("failed to open tarball %s: %w", tarFile, err)
	}

	options := types.ImageBuildOptions{
		Dockerfile: dockerfilePath,
		Tags:       f.Tags,
	}
	resp, err := cli.ImageBuild(context.Background(), buildContext, options)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	fmt.Println(buf.String())

	return nil
}

func (f *RandomImageFactory) createArchive(name string, filePaths []string) error {
	file, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("failed to create tar archive %q: %w", name, err)
	}

	tw := tar.NewWriter(file)
	defer func() {
		tw.Close()
	}()

	for _, filePath := range filePaths {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file info %q: %w", err)
		}

		bs, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %q: %w", err)
		}

		hdr := &tar.Header{
			Name: filePath,
			Mode: 0600,
			Size: fileInfo.Size(),
		}

		err = tw.WriteHeader(hdr)
		if err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		_, err = tw.Write(bs)
		if err != nil {
			return fmt.Errorf("failed to write tar body: %w", err)
		}
	}

	return nil
}

func (f *RandomImageFactory) shuffleGeneratedFilePaths() {
	for i := range f.allGeneratedFiles {
		j := rand.Intn(i + 1)
		f.allGeneratedFiles[i], f.allGeneratedFiles[j] = f.allGeneratedFiles[j], f.allGeneratedFiles[i]
	}
}

func (f *RandomImageFactory) generateRandomFilePool() error {
	err := os.MkdirAll(imageDir, 0700)
	if err != nil {
		return fmt.Errorf("failed creating directory %q: %w", imageDir, err)
	}

	for i := uint(0); i < f.LayerCount; i++ {
		filePath := path.Join(imageDir, fmt.Sprintf("random_%dKB_%d.txt", f.LayerSizeKB, i))
		f.allGeneratedFiles = append(f.allGeneratedFiles, filePath)

		_, err = os.Stat(filePath)
		if err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("error checking if file exists: %w", err)
		}

		err = ioutil.WriteFile(filePath, f.randBytes(1024*int(f.LayerSizeKB)), 0644)
		if err != nil {
			return fmt.Errorf("error writing random bytes to file: %w", err)
		}
	}

	return nil
}

func (f *RandomImageFactory) generateDockerfile(filePaths []string) (string, error) {
	filename := "./dockerfile.generated"
	dockerFile := `FROM scratch
`
	for _, path := range filePaths {
		dockerFile += fmt.Sprintf("ADD %s /opt\n", path)
	}

	err := ioutil.WriteFile(filename, []byte(dockerFile), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write to %q: %w", filename, err)
	}

	return filename, nil
}

func (f *RandomImageFactory) getDockerClient() (*client.Client, error) {
	if f.dockerClient != nil {
		return f.dockerClient, nil
	}

	cli, err := client.NewClientWithOpts(
		client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to initialized docker client: %w", err)
	}

	f.dockerClient = cli

	return cli, nil
}

// randBytes is largely copied from the "Mimicing strings.Builder with package
// unsafe" solution in https://stackoverflow.com/a/31832326, but adjusted to
// produce a byte array instead of a string
func (f *RandomImageFactory) randBytes(n int) []byte {
	b := make([]byte, n)
	src := rand.NewSource(f.Seed)

	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return b
}
