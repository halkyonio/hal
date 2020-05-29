package cmdutil

import (
	"fmt"
	halkyon "halkyon.io/api"
	capability "halkyon.io/api/capability/v1beta1"
	component "halkyon.io/api/component/v1beta1"
	"halkyon.io/hal/pkg/ui"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8yml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

var deserializer runtime.Decoder

func init() {
	s := scheme.Scheme
	if err := halkyon.AddToScheme(s); err != nil {
		panic(err)
	}

	deserializer = scheme.Codecs.UniversalDeserializer()
}

type HalkyonDescriptorEntity struct {
	Name   string
	Path   string
	Entity runtime.Object
}

func newHalkyonDescriptorEntity(object runtime.Object, name, path string) HalkyonDescriptorEntity {
	return HalkyonDescriptorEntity{Path: path, Entity: object, Name: name}
}

type entitiesRegistry map[string]HalkyonDescriptorEntity

type HalkyonDescriptor struct {
	entitiesByType map[ResourceType]entitiesRegistry
	path           string
}

func newHalkyonDescriptor(size int) *HalkyonDescriptor {
	types := KnownResourceTypes()
	hd := &HalkyonDescriptor{entitiesByType: make(map[ResourceType]entitiesRegistry, len(types))}
	for _, t := range types {
		hd.entitiesByType[t] = make(entitiesRegistry, size)
	}
	return hd
}

func (hd *HalkyonDescriptor) Add(object runtime.Object) {
	hd.add(object, hd.path)
}

func (hd *HalkyonDescriptor) add(object runtime.Object, path string) {
	switch t := object.(type) {
	case *capability.Capability:
		hd.addNewEntity(t, t.Name, path, Capability)
	case *component.Component:
		hd.addNewEntity(t, t.Name, path, Component)
	default:
		panic(fmt.Errorf("unknown object %T", t))
	}
}

func (hd *HalkyonDescriptor) addNewEntity(object runtime.Object, name, path string, rt ResourceType) {
	hdMap := hd.entitiesByType[rt]
	if e, ok := hdMap[name]; ok {
		if path != e.Path {
			panic(fmt.Errorf("attempted to register a %s named %s from %s but another one already exist in %s",
				object.GetObjectKind().GroupVersionKind().Kind, name, path, e.Path))
		}
	}
	hdMap[name] = newHalkyonDescriptorEntity(object, name, path)
}

func (hd *HalkyonDescriptor) mergeWith(descriptor *HalkyonDescriptor) {
	for _, registry := range descriptor.entitiesByType {
		for _, entity := range registry {
			hd.add(entity.Entity, entity.Path)
		}
	}
}

func (hd *HalkyonDescriptor) IsEmpty() bool {
	return hd.Size() == 0
}

func (hd *HalkyonDescriptor) Size() int {
	size := 0
	for _, registry := range hd.entitiesByType {
		size += len(registry)
	}
	return size
}

func (hd *HalkyonDescriptor) GetDefinedEntitiesWith(t ResourceType) entitiesRegistry {
	return hd.entitiesByType[t]
}

func LoadAvailableHalkyonEntities(path string) *HalkyonDescriptor {
	hd := newHalkyonDescriptor(10)
	hd.path = path
	hd.addEntitiesFromDir(path)

	children, err := ioutil.ReadDir(path)
	if err != nil {
		panic(err)
	}
	for _, child := range children {
		name := child.Name()
		if !strings.HasPrefix(name, ".") && child.IsDir() {
			hd.addEntitiesFromDir(filepath.Join(path, name))
		}
	}

	return hd
}

func (hd *HalkyonDescriptor) addEntitiesFromDir(path string) {
	extensions := []string{"yaml", "yml"}
	for _, extension := range extensions {
		hdPath := filepath.Join(path, descriptorName(extension))
		hd.loadDescriptorAt(hdPath)

		hdPath = halkyonDescriptorFrom(path, extension)
		hd.loadDescriptorAt(hdPath)
	}
}

func (hd *HalkyonDescriptor) loadDescriptorAt(hdPath string) {
	fromDekorate, e := LoadHalkyonDescriptor(hdPath)
	if e != nil && !os.IsNotExist(e) {
		ui.OutputError(fmt.Sprintf("Ignoring %s due to error: %v", hdPath, e))
	}
	hd.mergeWith(fromDekorate)
}

func LoadHalkyonDescriptor(descriptor string) (*HalkyonDescriptor, error) {
	return LoadHalkyonDescriptorCreatingIfNeeded(descriptor, false)
}

func LoadHalkyonDescriptorCreatingIfNeeded(descriptor string, create bool) (*HalkyonDescriptor, error) {
	// look for the component name in the halkyon descriptor
	if filepath.Base(descriptor) != "halkyon.yml" && filepath.Base(descriptor) != "halkyon.yaml" {
		descriptor = filepath.Join(descriptor, "halkyon.yml")
	}
	file, err := os.Open(descriptor)
	if err != nil {
		if !create {
			return newHalkyonDescriptor(0), err
		} else {
			hd := newHalkyonDescriptor(7)
			hd.path = descriptor
			return hd, nil
		}
	}
	decoder := k8yml.NewYAMLToJSONDecoder(file)
	list := &v1.List{}
	err = decoder.Decode(list)
	if err != nil {
		return newHalkyonDescriptor(0), err
	}
	hd := newHalkyonDescriptor(len(list.Items))
	hd.path = descriptor
	for _, value := range list.Items {
		object := value.Object
		if object == nil {
			object, _, err = deserializer.Decode(value.Raw, nil, nil)
			if err != nil {
				return newHalkyonDescriptor(0), err
			}
		}

		hd.add(object, descriptor)
	}

	return hd, nil
}

func (hd *HalkyonDescriptor) OutputAt(path ...string) error {
	list := v1.List{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		Items: make([]runtime.RawExtension, 0, 7),
	}
	for _, registry := range hd.entitiesByType {
		for _, entity := range registry {
			list.Items = append(list.Items, runtime.RawExtension{Object: entity.Entity})
		}
	}

	bytes, err := yaml.Marshal(list)
	if err != nil {
		return err
	}
	p := hd.path
	if len(hd.path) == 0 || len(path) == 1 {
		p = path[0]
	}
	return ioutil.WriteFile(p, bytes, 0644)
}

func halkyonDescriptorFrom(path, extension string) string {
	return filepath.Join(path, "target", "classes", "META-INF", "dekorate", descriptorName(extension))
}

func descriptorName(extension string) string {
	return fmt.Sprintf("halkyon.%s", extension)
}

func CreateOrUpdateHalkyonDescriptorWith(object runtime.Object, path string) error {
	descriptor, err := LoadHalkyonDescriptorCreatingIfNeeded(path, true)
	if err != nil {
		return err
	}
	descriptor.Add(object)
	return descriptor.OutputAt()
}
