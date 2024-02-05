CSHARP_OUT := "./gen_cs"
CSHARP_OPT := file_extension=.g.cs,serializable
GOLANG_OUT := "./gen_go"
GOLANG_OPT := module=github.com/stalomeow/protolinker/examples

all: csharp golang
.PHONY: all

csharp:
	protoc ./*.proto -I=.        \
	  --csharp_out=$(CSHARP_OUT) \
	  --csharp_opt=$(CSHARP_OPT) \
	  --cslink_out=$(CSHARP_OUT) \
	  --cslink_opt=config=link.toml
.PHONY: csharp

golang:
	protoc ./*.proto -I=.        \
	  --go_out=$(GOLANG_OUT)     \
	  --go_opt=$(GOLANG_OPT)     \
	  --golink_out=$(GOLANG_OUT) \
	  --golink_opt=$(GOLANG_OPT),config=link.toml
.PHONY: golang
