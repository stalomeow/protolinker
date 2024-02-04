package app

import (
	"bytes"
	"flag"
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/pluginpb"
	"strings"
)

type GenFile struct {
	buf         bytes.Buffer
	indentCount int
}

func (g *GenFile) P(v ...interface{}) {
	if len(v) > 0 {
		for i := 0; i < g.indentCount; i++ {
			fmt.Fprint(&g.buf, "    ")
		}
		for _, x := range v {
			fmt.Fprint(&g.buf, x)
		}
	}
	fmt.Fprintln(&g.buf)
}

func (g *GenFile) Indent(count int) {
	g.indentCount += count

	if g.indentCount < 0 {
		g.indentCount = 0
	}
}

func (g *GenFile) AppendToRsp(rsp *pluginpb.CodeGeneratorResponse, filename string) {
	rsp.File = append(rsp.File, &pluginpb.CodeGeneratorResponse_File{
		Name:    proto.String(filename),
		Content: proto.String(g.buf.String()),
	})
}

func UnderscoresToCamelCase(input string, capNextLetter, preservePeriod bool) string {
	var result strings.Builder

	for i := 0; i < len(input); i++ {
		if 'a' <= input[i] && input[i] <= 'z' {
			if capNextLetter {
				result.WriteByte(input[i] + 'A' - 'a')
			} else {
				result.WriteByte(input[i])
			}
			capNextLetter = false
		} else if 'A' <= input[i] && input[i] <= 'Z' {
			if i == 0 && !capNextLetter {
				result.WriteByte(input[i] + ('a' - 'A'))
			} else {
				result.WriteByte(input[i])
			}
			capNextLetter = false
		} else if '0' <= input[i] && input[i] <= '9' {
			result.WriteByte(input[i])
			capNextLetter = true
		} else {
			capNextLetter = true
			if input[i] == '.' && preservePeriod {
				result.WriteByte('.')
			}
		}
	}

	if len(input) > 0 && input[len(input)-1] == '#' {
		result.WriteByte('_')
	}

	if result.Len() > 0 && ('0' <= result.String()[0] && result.String()[0] <= '9') &&
		len(input) > 0 && input[0] == '_' {
		result.WriteString("_")
	}

	return result.String()
}

func SetCommandLineFlags(req *pluginpb.CodeGeneratorRequest, flags *flag.FlagSet) (map[string]string, error) {
	args := make(map[string]string)

	for _, param := range strings.Split(req.GetParameter(), ",") {
		var value string
		if i := strings.Index(param, "="); i >= 0 {
			value = param[i+1:]
			param = param[0:i]
		}

		if param != "" {
			args[param] = value
			if err := flags.Set(param, value); err != nil {
				return nil, err
			}
		}
	}
	return args, nil
}

func GetFilesToGenerate(req *pluginpb.CodeGeneratorRequest) ([]protoreflect.FileDescriptor, error) {
	genFileMap := make(map[string]interface{})
	for _, fileName := range req.FileToGenerate {
		genFileMap[fileName] = nil
	}

	fileReg := new(protoregistry.Files)
	results := make([]protoreflect.FileDescriptor, 0)
	for _, f := range req.ProtoFile {
		// 所有文件都要注册
		desc, err := protodesc.NewFile(f, fileReg)
		if err != nil {
			return nil, fmt.Errorf("invalid FileDescriptorProto %q: %v", f.GetName(), err)
		}
		if err := fileReg.RegisterFile(desc); err != nil {
			return nil, fmt.Errorf("cannot register descriptor %q: %v", f.GetName(), err)
		}

		// 只有需要生成的文件才会被返回
		if _, ok := genFileMap[f.GetName()]; ok {
			results = append(results, desc)
		}
	}
	return results, nil
}

func GetCompilerVersion(req *pluginpb.CodeGeneratorRequest) string {
	ver := req.GetCompilerVersion()
	return fmt.Sprintf("%v.%v.%v", ver.GetMajor(), ver.GetMinor(), ver.GetPatch())
}

func GetLeadingComments(desc protoreflect.MessageDescriptor) string {
	loc := desc.ParentFile().SourceLocations().ByDescriptor(desc)
	return loc.LeadingComments
}
