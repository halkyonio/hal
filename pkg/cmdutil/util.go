package cmdutil

import (
	"fmt"
	"github.com/spf13/cobra"
	halkyon "halkyon.io/api"
	"halkyon.io/api/component/v1beta1"
	"halkyon.io/hal/pkg/validation"
	"io/ioutil"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8yml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"path/filepath"
)

var deserializer runtime.Decoder

func init() {
	s := scheme.Scheme
	if err := halkyon.AddToScheme(s); err != nil {
		panic(err)
	}

	deserializer = scheme.Codecs.UniversalDeserializer()
}

func CommandName(name, fullParentName string) string {
	return fullParentName + " " + name
}

// FlagValueIfSet retrieves the value of the specified flag if it is set for the given command
func FlagValueIfSet(cmd *cobra.Command, flagName string) string {
	flag, _ := cmd.Flags().GetString(flagName)
	return flag
}

func IsInteractive(cmd *cobra.Command) bool {
	return cmd.Flags().NFlag() <= 2 // heuristics to determine whether we're running in interactive mode
}

func halkyonDescriptorFrom(path string) string {
	return filepath.Join(path, "target", "classes", "META-INF", "dekorate", "halkyon.yml")
}

type ComponentInfo struct {
	component  *v1beta1.Component
	descriptor string
}

func GetComponentsFrom(path string) ([]*v1beta1.Component, error) {
	name := filepath.Base(path)
	if "halkyon.yaml" != name && "halkyon.yml" != name {
		return []*v1beta1.Component{}, fmt.Errorf("%s is not an Halkyon descriptor (must be named halkyon.yml or halkyon.yaml)", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return []*v1beta1.Component{}, err
	}
	components := make([]*v1beta1.Component, 0, 7)
	decoder := k8yml.NewYAMLToJSONDecoder(file)
	list := &v1.List{}
	err = decoder.Decode(list)
	for _, value := range list.Items {
		object := value.Object
		if object == nil {
			object, _, err = deserializer.Decode(value.Raw, nil, nil)
			if err != nil {
				return []*v1beta1.Component{}, err
			}
		}
		// look for a component descriptor in the halkyon list
		if c, ok := object.(*v1beta1.Component); ok {
			components = append(components, c)
		}
	}
	return components, nil
}

func GetComponents() (map[string]ComponentInfo, error) {
	// look for halkyon descriptors starting from the current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	infos := make(map[string]ComponentInfo, 7)

	// look for dekorate descriptor
	infos, err = addComponentsFromDir(currentDir, infos)
	if err != nil {
		return nil, err
	}

	// look for dekorate and top-level halkyon descriptors in child dirs
	children, err := ioutil.ReadDir(currentDir)
	if err != nil {
		return nil, err
	}
	for _, child := range children {
		if child.IsDir() {
			match := filepath.Join(currentDir, child.Name())
			infos, err = addComponentsFromDir(match, infos)
			if err != nil {
				return nil, err
			}
		}
	}
	return infos, nil
}

func addComponentsFromDir(currentDir string, infos map[string]ComponentInfo) (map[string]ComponentInfo, error) {
	// look for dekorate descriptor
	dekorate := halkyonDescriptorFrom(currentDir)
	if err := addComponentsFrom(dekorate, infos); err != nil {
		return nil, err
	}

	// look for halkyon descriptor at the root of the current dir
	if err := addComponentsFrom(filepath.Join(currentDir, "halkyon.yml"), infos); err != nil {
		return nil, err
	}
	if err := addComponentsFrom(filepath.Join(currentDir, "halkyon.yaml"), infos); err != nil {
		return nil, err
	}
	return infos, nil
}

func addComponentsFrom(descriptor string, infos map[string]ComponentInfo) error {
	if validation.CheckFileExist(descriptor) {
		components, err := GetComponentsFrom(descriptor)
		if err != nil {
			return err
		}
		for _, component := range components {
			infos[component.Name] = ComponentInfo{
				component:  component,
				descriptor: descriptor,
			}
		}
	}
	return nil
}
