# protolinker

Protobuf 消息分组 + 自动编号，支持 C# 和 Go。

该插件会额外生成一个 Message 汇总文件，方便根据 id 创建 Message 对象。

## 安装

```
go install github.com/stalomeow/protolinker/...@latest
```

## 配置

TOML 配置文件，可以起名为 `link.toml`。

``` toml
# 配置 C# 的 Message 汇总文件
[out.csharp]
namespace = "Examples"
class = "NetMessageStore"
filename = "NetMessageStore.cs"
disposable_message = false # 是否让 Message 实现 IDisposable 接口

# 配置 Go 的 Message 汇总文件
[out.go]
import_path = "github.com/stalomeow/protolinker/examples"
package = "examples"

# Go 的文件名配置比较特殊，请参考下面文档中 output directory 相关内容
# https://protobuf.dev/reference/go/go-generated/#invocation
filename = "github.com/stalomeow/protolinker/examples/net_message.go"

# 第一组消息
[[groups]]
name = "group/one"
min = 100 # 该组消息的 uint16 id 的最小值（包含）
max = 300 # 该组消息的 uint16 id 的最大值（包含）

# 第二组消息
[[groups]]
name = "group/two"
min = 400
max = 900

# ...
```

## 分组

分组信息写在 Message 的 Leading Comments 里，独占一行，格式为 `@group="GROUP_NAME"`。`=` 左右不要有空格。

``` protobuf
syntax = "proto3";

option csharp_namespace = "Examples";
option go_package = "github.com/stalomeow/protolinker/examples";

// @group="group/one"
message MessageA {
    int32 i = 1;
}

// @group="group/one"
message MessageB {
    int32 i = 1;
}

// @group="group/two"
message MessageC {
    int32 i = 1;
}

// @group="group/two"
message MessageD {
    int32 i = 1;
}
```

## 生成

id 的分配顺序取决于：

1. protoc 参数中 `.proto` 文件的顺序。
2. `.proto` 文件中 Message 的定义顺序。

如果分开调用 protoc，先生成 C# 再生成 Go 的话，需要保证上面几个顺序一致。

### C#

``` make
protoc ./*.proto -I=.        \
  --csharp_out=$(CSHARP_OUT) \
  --csharp_opt=$(CSHARP_OPT) \
  --cslink_out=$(CSHARP_OUT) \
  --cslink_opt=config=link.toml
```

- `--cslink_out`：指定输出目录。
- `--cslink_opt`：

    - `config`：配置文件的路径。
    - `base_namespace`：同 `--csharp_opt` 里的 `base_namespace`。

### Go

``` make
protoc ./*.proto -I=.        \
  --go_out=$(GOLANG_OUT)     \
  --go_opt=$(GOLANG_OPT)     \
  --golink_out=$(GOLANG_OUT) \
  --golink_opt=$(GOLANG_OPT),config=link.toml
```

- `--golink_out`：指定输出目录。
- `--golink_opt`：

    - `config`：配置文件的路径。
    - 所有 `--go_opt` 的参数。

## 使用

[examples 文件夹](examples) 中有生成的样例代码。

### C#

``` csharp
var store = new Examples.NetMessageStore();

// 从某处获得 msgId、msgSize 以及 data

// 过滤消息
if (msgId < store.MsgGroupMin_GroupOne || msgId > store.MsgGroupMax_GroupOne)
{
    return;
}

// 解析消息
MessageParser parser = store.GetMsgParserById(msgId);
var span = new ReadOnlySpan<byte>(data, msgSize);
IMessage msg = parser.ParseFrom(span);

// 处理消息
switch (msgId)
{
    case Examples.MessageA.MsgId:
        // ...
        break;

    case Examples.MessageB.MsgId:
        // ...
        break;

    // ...
}
```

``` csharp
// INetMessage 是插件生成的接口
// 它派生自 IMessage，包含了 MsgId 和 MsgName 信息
INetMessage msg = new Examples.MessageA();

// 消息长度
int msgSize = msg.CalculateSize();

// 消息 id
ushort msgId = msg.MsgId;

// 写入消息
msg.WriteTo(span);
```

### Go

``` go
// 从某处获得 msgId、msgSize 以及 data

// 过滤消息
if msgId < examples.MsgGroupMin_GroupOne || msgId > examples.MsgGroupMax_GroupOne {
    return
}

// 解析消息
msg, err := examples.NewMsgById(msgId)
if err != nil {
    panic(err)
}
err = pb.Unmarshal(data[:msgSize], msg)
if err != nil {
    panic(err)
}

// 处理消息
switch msgId {
case examples.MessageA_MsgId:
    // ...

case examples.MessageB_MsgId:
    // ...

// ...
}
```

``` go
// NetMessage 是插件生成的接口，同 C#
msg := new(examples.MessageA).(examples.NetMessage)

// 消息 id
msgId := msg.MsgId()
```
