package app

import (
	"flag"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
	"slices"
)

type GoGenerator struct {
	version string

	compilerVersion string
	ctx             *GenContext
}

func NewGoGenerator(version string) *GoGenerator {
	return &GoGenerator{version: version}
}

func (g *GoGenerator) initGen(req *pluginpb.CodeGeneratorRequest) (*protogen.Plugin, error) {
	var configFile string
	flags := &flag.FlagSet{}
	flags.StringVar(&configFile, "config", "link.toml", "")

	// 创建 Plugin 并读取命令行参数
	plugin, err := protogen.Options{ParamFunc: flags.Set}.New(req)
	if err != nil {
		return nil, err
	}

	g.compilerVersion = GetCompilerVersion(req)
	g.ctx, err = NewGenContextFromConfigFile(configFile)
	if err != nil {
		return nil, err
	}
	return plugin, nil
}

func (g *GoGenerator) Execute(req *pluginpb.CodeGeneratorRequest) (*pluginpb.CodeGeneratorResponse, error) {
	plugin, err := g.initGen(req)
	if err != nil {
		return nil, err
	}

	for _, f := range plugin.Files {
		if !f.Generate {
			continue
		}

		fileName := f.GeneratedFilenamePrefix + "_link.pb.go"
		genFile := plugin.NewGeneratedFile(fileName, f.GoImportPath)
		g.writeFileHeader(f, genFile)

		msgNames, err := g.writeFlatMsg(f.Messages, genFile)
		if err != nil {
			return nil, err
		}
		if len(msgNames) <= 0 {
			genFile.Skip()
			continue
		}

		g.writeInitFunc(msgNames, genFile)
	}

	g.genLinkFile(plugin)
	return plugin.Response(), nil
}

func (g *GoGenerator) writeFileHeader(srcFile *protogen.File, genFile *protogen.GeneratedFile) {
	genFile.P("// Code generated by protoc-gen-golink. DO NOT EDIT.")
	genFile.P("// versions:")
	genFile.P("// \tprotoc-gen-golink v", g.version)

	if srcFile != nil {
		genFile.P("// \tprotoc            v", g.compilerVersion)
		genFile.P("// source: ", srcFile.Desc.Path())
		genFile.P()
		genFile.P("package ", srcFile.GoPackageName)
	}

	genFile.P()
}

func (g *GoGenerator) writeFlatMsg(messages []*protogen.Message, genFile *protogen.GeneratedFile) ([]string, error) {
	allMsgTypeNames := make([]string, 0)
	stack := slices.Clone(messages)
	slices.Reverse(stack)

	// DFS
	for len(stack) > 0 {
		msg := stack[len(stack)-1]
		stack = append(stack[:len(stack)-1], msg.Messages...)
		slices.Reverse(stack[len(stack)-len(msg.Messages):])

		msgId, ok, err := g.ctx.AllocMsgId(string(msg.Comments.Leading))
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		msgName := msg.GoIdent.GoName
		allMsgTypeNames = append(allMsgTypeNames, msgName)

		genFile.P("const (")
		genFile.P("    ", msgName, "_MsgId uint16 = ", msgId)
		genFile.P("    ", msgName, "_MsgName string = \"", msgName, "\"")
		genFile.P(")")
		genFile.P()
		genFile.P("func (*", msgName, ") MsgId() uint16 { return ", msgName, "_MsgId }")
		genFile.P("func (*", msgName, ") MsgName() string { return ", msgName, "_MsgName }")
		genFile.P()
	}
	return allMsgTypeNames, nil
}

func (g *GoGenerator) writeInitFunc(msgNames []string, genFile *protogen.GeneratedFile) {
	pkg := protogen.GoImportPath(g.ctx.Config.Out.Go.ImportPath)
	registerFunc := pkg.Ident("registerNetMessage")
	interfaceName := pkg.Ident("NetMessage")

	genFile.P("func init() {")
	for _, name := range msgNames {
		genFile.P("    ", registerFunc, "(func() ", interfaceName, " { return new(", name, ") })")
	}
	genFile.P("}")
}

func (g *GoGenerator) genLinkFile(plugin *protogen.Plugin) {
	pkgFmt := protogen.GoImportPath("fmt")
	pkgPb := protogen.GoImportPath("google.golang.org/protobuf/proto")

	genFile := plugin.NewGeneratedFile(
		g.ctx.Config.Out.Go.Filename,
		protogen.GoImportPath(g.ctx.Config.Out.Go.ImportPath),
	)
	g.writeFileHeader(nil, genFile)
	genFile.P("package ", g.ctx.Config.Out.Go.Package)
	genFile.P()
	genFile.P("const (")

	for _, group := range g.ctx.Config.MsgGroups {
		groupNameIdent := UnderscoresToCamelCase(group.Name, true, false)
		genFile.P("    MsgGroupMin_", groupNameIdent, " = ", group.Min)
		genFile.P("    MsgGroupMax_", groupNameIdent, " = ", group.Max)
	}

	genFile.P(")")
	genFile.P()
	genFile.P("type NetMessage interface {")
	genFile.P("    ", pkgPb.Ident("Message"))
	genFile.P("    MsgId() uint16")
	genFile.P("    MsgName() string")
	genFile.P("}")
	genFile.P()
	genFile.P("type msgInfo struct {")
	genFile.P("    name    string")
	genFile.P("    factory func() NetMessage")
	genFile.P("}")
	genFile.P()
	genFile.P("var msgInfoMap = make(map[uint16]*msgInfo)")
	genFile.P()
	genFile.P("func registerNetMessage(factory func() NetMessage) {")
	genFile.P("    msg := factory()")
	genFile.P("    msgInfoMap[msg.MsgId()] = &msgInfo{")
	genFile.P("        name:    msg.MsgName(),")
	genFile.P("        factory: factory,")
	genFile.P("    }")
	genFile.P("}")
	genFile.P()
	genFile.P("func NewMsgById(msgId uint16) (NetMessage, error) {")
	genFile.P("    info, ok := msgInfoMap[msgId]")
	genFile.P("    if !ok {")
	genFile.P("        return nil, ", pkgFmt.Ident("Errorf"), "(\"unknown message id: %d\", msgId)")
	genFile.P("    }")
	genFile.P("    return info.factory(), nil")
	genFile.P("}")
	genFile.P()
	genFile.P("func MsgName(msgId uint16) (string, error) {")
	genFile.P("    info, ok := msgInfoMap[msgId]")
	genFile.P("    if !ok {")
	genFile.P("        return \"\", ", pkgFmt.Ident("Errorf"), "(\"unknown message id: %d\", msgId)")
	genFile.P("    }")
	genFile.P("    return info.name, nil")
	genFile.P("}")
}
