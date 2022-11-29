package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/mt-sre/devkube/dev"
	appsv1 "k8s.io/api/apps/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
)

// Replaces `container`'s image. If it is the packager operator manager,
//the `PKO_IMAGE` environment variable is also replaced with that image.
func patchDeployment(deployment *appsv1.Deployment, name string, container string) {
	image := getImageName(name)

	if name == "package-operator-manager" {
		replaceImageAndEnvVar(deployment, image, container, "PKO_IMAGE")
	} else {
		replaceImage(deployment, image, container)
	}
}

func replaceImage(deployment *appsv1.Deployment, image string, container string) {
	for i := range deployment.Spec.Template.Spec.Containers {
		containerObj := &deployment.Spec.Template.Spec.Containers[i]

		if containerObj.Name == container {
			containerObj.Image = image
			break
		}
	}
}

func replaceImageAndEnvVar(deployment *appsv1.Deployment, image string, container string, envVar string) {
	for i := range deployment.Spec.Template.Spec.Containers {
		containerObj := &deployment.Spec.Template.Spec.Containers[i]

		if containerObj.Name == container {
			containerObj.Image = image
			for j := range containerObj.Env {
				env := &containerObj.Env[j]
				if env.Name == envVar {
					env.Value = image
				}
			}
			break
		}
	}
}

func (b *builder) imageURL(name string) string {
	return b.internalImageURL(name, false)
}

func (b *builder) imageURLWithDigest(name string) string {
	return b.internalImageURL(name, true)
}

func digestFile(imageName string) string {
	return path.Join(cacheDir, imageName+".digest")
}

func (b *builder) internalImageURL(name string, useDigest bool) string {
	envvar := strings.ReplaceAll(strings.ToUpper(name), "-", "_") + "_IMAGE"
	if url := os.Getenv(envvar); len(url) != 0 {
		return url
	}

	if !useDigest {
		image := b.imageOrg + "/" + name + ":" + b.version
		return image
	}

	digest, err := os.ReadFile(digestFile(name))
	if err != nil {
		panic(err)
	}

	image := b.imageOrg + "/" + name + "@" + string(digest)
	return image
}

// dependency for all targets requiring a container runtime
func determineContainerRuntime() {
	containerRuntime = os.Getenv("CONTAINER_RUNTIME")
	if len(containerRuntime) == 0 || containerRuntime == "auto" {
		cr, err := dev.DetectContainerRuntime()
		if err != nil {
			panic(err)
		}
		containerRuntime = string(cr)
		logger.Info("detected container-runtime", "container-runtime", containerRuntime)
	}
}

func loadIntoObject(scheme *k8sruntime.Scheme, filePath string, out interface{}) error {
	objs, err := dev.LoadKubernetesObjectsFromFile(filePath)
	if err != nil {
		return fmt.Errorf("loading object from file: %w", err)
	}
	if err := scheme.Convert(&objs[0], out, nil); err != nil {
		return fmt.Errorf("converting: %w", err)
	}
	return nil
}

// dumpManifestsFromFolder dumps all kubernetes manifests from all files
// in the given folder into the output file. It does not recurse into subfolders.
// It dumps the manifests in lexical order based on file name.
func dumpManifestsFromFolder(folderPath string, outputPath string) error {
	folder, err := os.Open(folderPath)
	if err != nil {
		return fmt.Errorf("open %q: %w", folderPath, err)
	}
	defer folder.Close()

	files, err := folder.Readdir(-1)
	if err != nil {
		return fmt.Errorf("reading directory: %w", err)
	}
	sort.Sort(fileInfosByName(files))

	if _, err = os.Stat(outputPath); err == nil {
		err = os.Remove(outputPath)
		if err != nil {
			return fmt.Errorf("removing old file: %s", err)
		}
	}

	outputFile, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed opening file: %s", err)
	}
	defer outputFile.Close()
	for i, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := path.Join(folderPath, file.Name())
		fileYaml, err := ioutil.ReadFile(filePath)
		cleanFileYaml := bytes.Trim(fileYaml, "-\n")
		if err != nil {
			return fmt.Errorf("reading %s: %w", filePath, err)
		}

		_, err = outputFile.Write(cleanFileYaml)
		if err != nil {
			return fmt.Errorf("failed appending manifest from file %s to output file: %s", file, err)
		}
		if i != len(files)-1 {
			_, err = outputFile.WriteString("\n---\n")
			if err != nil {
				return fmt.Errorf("failed appending --- %s to output file: %s", file, err)
			}
		} else {
			_, err = outputFile.WriteString("\n")
			if err != nil {
				return fmt.Errorf("failed appending new line %s to output file: %s", file, err)
			}
		}
	}
	return nil
}

// Sorts fs.FileInfo objects by basename.
type fileInfosByName []fs.FileInfo

func (x fileInfosByName) Len() int { return len(x) }

func (x fileInfosByName) Less(i, j int) bool {
	iName := path.Base(x[i].Name())
	jName := path.Base(x[j].Name())
	return iName < jName
}

func (x fileInfosByName) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
