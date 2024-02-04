package app

import (
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"os"
	"strings"
)

type CSharpConfig struct {
	Namespace       string `toml:"namespace"`
	ClassName       string `toml:"class"`
	Filename        string `toml:"filename"`
	IsMsgDisposable bool   `toml:"disposable_message,omitempty"`
}

type GoConfig struct {
	ImportPath string `toml:"import_path"`
	Package    string `toml:"package"`
	Filename   string `toml:"filename"`
}

type OutConfig struct {
	CSharp *CSharpConfig `toml:"csharp"`
	Go     *GoConfig     `toml:"go"`
}

type MsgGroupConfig struct {
	Name string `toml:"name"`
	Min  uint16 `toml:"min"`
	Max  uint16 `toml:"max"`
}

type GenConfig struct {
	Out       *OutConfig        `toml:"out"`
	MsgGroups []*MsgGroupConfig `toml:"groups"`
}

func ReadConfigFromFile(file string) (*GenConfig, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var config GenConfig
	if err = toml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

type GenContext struct {
	Config       *GenConfig
	nextMsgIdMap map[string]uint16
	maxMsgIdMap  map[string]uint16
}

func NewGenContext(c *GenConfig) *GenContext {
	ctx := &GenContext{
		Config:       c,
		nextMsgIdMap: make(map[string]uint16),
		maxMsgIdMap:  make(map[string]uint16),
	}
	for _, group := range c.MsgGroups {
		ctx.nextMsgIdMap[group.Name] = group.Min
		ctx.maxMsgIdMap[group.Name] = group.Max
	}
	return ctx
}

func NewGenContextFromConfigFile(configFile string) (*GenContext, error) {
	c, err := ReadConfigFromFile(configFile)
	if err != nil {
		return nil, err
	}
	return NewGenContext(c), nil
}

func (ctx *GenContext) AllocMsgId(magicComments string) (uint16, bool, error) {
	for _, line := range strings.Split(magicComments, "\n") {
		groupName, found := strings.CutPrefix(strings.TrimSpace(line), "@group=\"")
		if !found || len(groupName) < 2 || groupName[len(groupName)-1] != '"' {
			continue
		}

		groupName = groupName[:len(groupName)-1]
		if id, ok := ctx.nextMsgIdMap[groupName]; ok && id <= ctx.maxMsgIdMap[groupName] {
			ctx.nextMsgIdMap[groupName]++
			return id, true, nil
		} else if !ok {
			return 0, false, fmt.Errorf("group %s does not exists", groupName)
		} else {
			return 0, false, fmt.Errorf("group %s has no more available message id", groupName)
		}
	}
	return 0, false, nil
}
