package app

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
	"io"
	"os"
)

func Run(f func(req *pluginpb.CodeGeneratorRequest) (*pluginpb.CodeGeneratorResponse, error)) error {
	if len(os.Args) > 1 {
		return fmt.Errorf("unknown argument %q (this program should be run by protoc, not directly)", os.Args[1])
	}

	inData, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	req := &pluginpb.CodeGeneratorRequest{}
	if err := proto.Unmarshal(inData, req); err != nil {
		return err
	}

	rsp, err := f(req)
	if err != nil {
		return err
	}

	outData, err := proto.Marshal(rsp)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(outData)
	return err
}
