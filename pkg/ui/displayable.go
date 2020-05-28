package ui

import (
	"sort"
	"strings"
)

type Displayable interface {
	Help() string
	Display() string
	Name() string
	GetUnderlying() interface{}
}

type DisplayableMap struct {
	byName    map[string]Displayable
	byIndex   []Displayable
	byDisplay map[string]Displayable
}

var Empty = DisplayableMap{}

func NewDisplayableMap(size int) DisplayableMap {
	return DisplayableMap{
		byName:    make(map[string]Displayable, size),
		byIndex:   make([]Displayable, 0, size),
		byDisplay: make(map[string]Displayable, size),
	}
}

func (d *DisplayableMap) Add(displayable Displayable) {
	d.byName[displayable.Name()] = displayable
	d.byIndex = append(d.byIndex, displayable)
	d.byDisplay[displayable.Display()] = displayable
}

func (d DisplayableMap) Len() int {
	return len(d.byIndex)
}

func (d DisplayableMap) Less(i, j int) bool {
	return strings.Compare(d.byIndex[i].Display(), d.byIndex[j].Display()) == -1
}

func (d DisplayableMap) Swap(i, j int) {
	d.byIndex[i], d.byIndex[j] = d.byIndex[j], d.byIndex[i]
}

func (d DisplayableMap) asDisplayableOptions() []string {
	result := make([]string, 0, d.Len())
	for _, displayable := range d.byIndex {
		result = append(result, displayable.Display())
	}
	sort.Strings(result)
	return result
}

func (d DisplayableMap) GetByIndex(i int) Displayable {
	return d.byIndex[i]
}

func (d DisplayableMap) GetByName(name string) (Displayable, bool) {
	displayable, ok := d.byName[name]
	return displayable, ok
}

func (d DisplayableMap) GetByDisplay(display string) (Displayable, bool) {
	displayable, ok := d.byDisplay[display]
	return displayable, ok
}
