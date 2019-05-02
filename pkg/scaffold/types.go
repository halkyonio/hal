package scaffold

import "sort"

type Project struct {
	GroupId     string
	ArtifactId  string
	Version     string
	PackageName string
	OutDir      string
	Template    string `yaml:"template"  json:"template"`

	SnowdropBomVersion string
	SpringBootVersion  string
	Modules            []string

	UrlService   string
	UseAp4k      bool
	UseSupported bool
}

type Config struct {
	Templates []Template `yaml:"templates"    json:"templates"`
	Boms      []Bom      `yaml:"bomversions"  json:"bomversions"`
	Modules   []Module   `yaml:"modules"      json:"modules"`
}

func (c *Config) GetTemplatesMap() map[string]Template {
	result := make(map[string]Template, len(c.Templates))

	for _, value := range c.Templates {
		result[value.Name] = value
	}

	return result
}

func (c *Config) GetTemplateNames() []string {
	result := make([]string, len(c.Templates))
	for i, value := range c.Templates {
		result[i] = value.Name
	}
	sort.Strings(result)
	return result
}

func (c *Config) GetModuleNames() []string {
	return GetModuleNamesFor(c.Modules)
}

func GetModuleNamesFor(modules []Module) []string {
	result := make([]string, len(modules))
	for i, v := range modules {
		result[i] = v.Name
	}
	sort.Strings(result)
	return result
}

func (c *Config) GetBOMMap() (map[string]Bom, string) {
	var defaultVersion string
	result := make(map[string]Bom, len(c.Boms))
	for _, v := range c.Boms {
		result[v.Community] = v
		if v.Default {
			defaultVersion = v.Community
		}
	}
	return result, defaultVersion
}

func (c *Config) GetSpringBootVersions() []string {
	boms, _ := c.GetBOMMap()
	return GetSpringBootVersions(boms)
}

func GetSpringBootVersions(boms map[string]Bom) []string {
	result := make([]string, 0, len(boms))
	for k := range boms {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

func (c *Config) GetSupportedVersionFor(springBootVersion string) string {
	for _, v := range c.Boms {
		if v.Community == springBootVersion {
			return v.Supported
		}
	}
	return ""
}

type Template struct {
	Name        string `yaml:"name"                     json:"name"`
	Description string `yaml:"description"              json:"description"`
}

type Bom struct {
	Community string `yaml:"community" json:"community"`
	Snowdrop  string `yaml:"snowdrop"  json:"snowdrop"`
	Supported string `yaml:"supported"  json:"supported"`
	Default   bool   `yaml:"default"  json:"default"`
}

type Module struct {
	Name         string       `yaml:"name"             json:"name"`
	Description  string       `yaml:"description"      json:"description"`
	Guide        string       `yaml:"guide_ref"        json:"guide_ref"`
	Dependencies []Dependency `yaml:"dependencies"     json:"dependencies"`
	tags         []string     `yaml:"tags"             json:"tags"`
}

type Dependency struct {
	GroupId    string `yaml:"groupid"           json:"groupid"`
	ArtifactId string `yaml:"artifactid"        json:"artifactid"`
	Scope      string `yaml:"scope"             json:"scope"`
	Version    string `yaml:"version,omitempty" json:"version,omitempty"`
}
