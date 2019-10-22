package cmdutil

import (
	"fmt"
	halkyon "halkyon.io/api"
	capability "halkyon.io/api/capability/v1beta1"
	component "halkyon.io/api/component/v1beta1"
	link "halkyon.io/api/link/v1beta1"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
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
}

func newHalkyonDescriptor(size int) *HalkyonDescriptor {
	types := KnownResourceTypes()
	hd := &HalkyonDescriptor{entitiesByType: make(map[ResourceType]entitiesRegistry, len(types))}
	for _, t := range types {
		hd.entitiesByType[t] = make(entitiesRegistry, size)
	}
	return hd
}

func (hd *HalkyonDescriptor) add(object runtime.Object, path string) {
	switch t := object.(type) {
	case *capability.Capability:
		hd.addNewEntity(t, t.Name, path)
	case *component.Component:
		hd.addNewEntity(t, t.Name, path)
	case *link.Link:
		hd.addNewEntity(t, t.Name, path)
	default:
		panic(fmt.Errorf("unknown object %T", t))
	}
}

func (hd *HalkyonDescriptor) addNewEntity(object runtime.Object, name, path string) {
	rt, err := ResourceTypeFor(object)
	if err != nil {
		panic(err)
	}
	hdMap := hd.entitiesByType[rt]
	if e, ok := hdMap[name]; ok {
		panic(fmt.Errorf("attempted to register a %s named %s from %s but another one already exist in %s",
			object.GetObjectKind().GroupVersionKind().Kind, name, path, e.Path))
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
	hd.addEntitiesFromDir(path)

	children, err := ioutil.ReadDir(path)
	if err != nil {
		panic(err)
	}
	for _, child := range children {
		if child.IsDir() {
			hd.addEntitiesFromDir(filepath.Join(path, child.Name()))
		}
	}

	return hd
}

func (hd *HalkyonDescriptor) addEntitiesFromDir(path string) {
	extensions := []string{"yaml", "yml"}
	for _, extension := range extensions {
		hdPath := filepath.Join(path, descriptorName(extension))
		d, _ := LoadHalkyonDescriptor(hdPath)
		hd.mergeWith(d)

		hdPath = halkyonDescriptorFrom(path, extension)
		fromDekorate, _ := LoadHalkyonDescriptor(hdPath)
		hd.mergeWith(fromDekorate)
	}
}

func LoadHalkyonDescriptor(descriptor string) (*HalkyonDescriptor, error) {
	// look for the component name in the halkyon descriptor
	file, err := os.Open(descriptor)
	if err != nil {
		return newHalkyonDescriptor(0), err
	}
	decoder := k8yml.NewYAMLToJSONDecoder(file)
	list := &v1.List{}
	err = decoder.Decode(list)
	if err != nil {
		return newHalkyonDescriptor(0), err
	}
	hd := newHalkyonDescriptor(len(list.Items))
	for _, value := range list.Items {
		object := value.Object
		if object == nil {
			object, _, err = deserializer.Decode(value.Raw, nil, nil)
			if err != nil {
				return nil, err
			}
		}

		hd.add(object, descriptor)
	}

	return hd, nil
}

func halkyonDescriptorFrom(path, extension string) string {
	return filepath.Join(path, "target", "classes", "META-INF", "dekorate", descriptorName(extension))
}

func descriptorName(extension string) string {
	return fmt.Sprintf("halkyon.%s", extension)
}
