# cmd 包使用说明

> oneshow 项目自写的命令行参数解析库（`github.com/milin2436/oneshow/cmd`）。
> **不用** Go 标准库 `flag`，参数通过预先定义（`ParamDef`）的方式使用。
> 本文档供开发时（包括 Claude 后续使用）快速上手，避免踩坑。

## 1. 核心类型

| 类型 | 作用 |
|---|---|
| `Context` | 全局上下文。`CmdMap`(已注册命令)、`ParamGroupMap`(本次解析出的参数)、`Single`(单命令模式)、`Name`/`Version`(帮助头部显示，默认取自构建注入的 `CmdName`/`VERSION`)、`Debug`(解析调试开关)。 |
| `Program` | 一个子命令。`Name`(命令名)、`Desc`、`Usage`、`Cmd`(执行函数)、`ParamDefMap`(参数定义表)、`Target`(最后一个位置参数)、`Args`(**全部**位置参数，按顺序)。 |
| `ParamDef` | 一个参数的定义。`Name`(短名，用作 map key，可以是多字符如 `dn`)、`LongName`(长名，`--xxx` 形式按它匹配)、`NeedValue`(是否需要值)、`Desc`。 |
| `ParamGroup` | 解析后的一个参数实例，`Name` / `Value` / `NeedValue`，存放在 `ct.ParamGroupMap[Name]`。 |

命令执行函数签名：`type CMD func(*Program)`。

## 2. 基本流程（多命令模式，oneshow 的实际用法）

```go
ct := cmd.NewContext()      // 1. 创建上下文
registerCommands(ct)        // 2. 自己往 ct.CmdMap 注册命令（见下）
ct.Run()                    // 3. 按 os.Args[1] 找子命令执行；找不到则打印帮助
```

注册命令示例（照抄 `main/oneshow.go` 里 `setFuns()` 的模式）：

```go
func registerCommands(ct *cmd.Context) {
    ct.CmdMap = map[string]*cmd.Program{}

    pro := new(cmd.Program)
    pro.Name = "ls"
    pro.Desc = "list onedrive directory contents"
    pro.Usage = "usage: " + pro.Name + " [OPTION] path"
    pro.ParamDefMap = map[string]*cmd.ParamDef{}

    pro.ParamDefMap["h"] = &cmd.ParamDef{Name: "h", LongName: "help", NeedValue: false, Desc: "print help"}
    pro.ParamDefMap["l"] = &cmd.ParamDef{Name: "l", LongName: "list", NeedValue: false, Desc: "list files detail"}
    pro.ParamDefMap["d"] = &cmd.ParamDef{Name: "d", LongName: "dir", NeedValue: true, Desc: "set download dir"}

    pro.Cmd = func(pro *cmd.Program) {
        if ct.ParamGroupMap["h"] != nil {   // 惯例：每个命令先处理 -h
            cmd.PrintCmdHelp(pro)
            return
        }
        dir := pro.Target                  // 位置参数（最后一个）
        for _, a := range pro.Args { ... } // 位置参数（全部）
        if p := ct.ParamGroupMap["d"]; p != nil {   // 带值参数
            _ = p.Value
        }
    }
    ct.CmdMap[pro.Name] = pro
}
```

## 3. 解析规则（实际语义，务必记住）

调用链：`Run()` 多命令模式调 `ParseArgs(os.Args[2:], fn)`，单命令模式调 `ParseArgs(os.Args[1:], app)`。**`ParseArgs` 只解析"命令名之后"的参数，从 index 0 开始，不再跳过任何元素。**

支持的参数形式（`ParseArgs` 一个函数全部处理）：

| 输入 | 效果 |
|---|---|
| `-h` / `-dn` | 布尔开关，短名形式（按 `Name` 匹配，`dn` 这种多字符短名也行） |
| `--help` / `--list` | 布尔开关，长名形式（按 `LongName` 匹配；`--h` 这种与短名相同的也能回退到 `Name`） |
| `-d <dir>` | 带值参数，值取下一个非 `-` 开头的参数 |
| `-d=<dir>` | 带值参数，`=` 内嵌值 |
| `--dir=<dir>` | 长名 + 内嵌值 |
| `--` | 其后所有参数一律视为位置参数（不再解析为选项） |
| `-` | 单独一个 `-` 按位置参数处理 |
| `""` | 空参数直接跳过，不会 panic |

规则：

- **未定义的参数直接报错**：`unknown option: -x`。拼错参数名会立刻暴露（不再静默忽略）。
- 带值参数缺值（下一个是 `-x` 或已到末尾）→ 报错 `can not get value of <name>`。
- 位置参数：**全部**存入 `pro.Args`，**最后一个**再写一份到 `pro.Target`（保持旧代码只读 `Target` 也能用）。
- 每次 `ParseArgs` 前会**自动清空**上次的 `ParamGroupMap` / `Target` / `Args`，多次解析不会残留脏数据。
- 匹配顺序：短名 `Name` 优先，查不到再按 `LongName`。

参数读取惯用方式：

```go
// 布尔开关：-l 是否出现过
if ct.Has("l") { ... }                       // 等价于 ct.ParamGroupMap["l"] != nil
// 带值参数（带默认值）
dir := ct.Get("d", ".")                      // -d 给了取 -d 的值，否则 "."（空值会原样返回，不替换成默认）
// 组合校验（返回 bool，配合 if 判断）
ct.Need("f", "g")                            // 全部出现才为 true
ct.Any("a", "b", "c")                        // 至少一个出现
// 帮助
if ct.Has("h") { cmd.PrintCmdHelp(pro); return }
```

辅助方法定义在 `Context` 上：`Has(name)` / `Need(names...)` / `Any(names...)` / `Get(name, def)`。`name` 都是 `ParamDef` 的 key（短名，如 `l`）。注意这些只做"是否存在"判断——**必填/二选一/条件依赖这类规则在 handler 里用普通代码表达**，不给 `ParamDef` 加声明式校验字段（见 §6 第 4 条）。

## 4. 单命令模式

```go
ct := cmd.NewContext()
app := &cmd.Program{ ... }   // 填好 ParamDefMap、Cmd
ct.Single = app
ct.Run()                      // 内部解析 os.Args[1:]
```

## 5. 帮助输出

- `ct.ShowHelp()`：列出 `CmdMap` 所有命令名 + 描述，**按名称排序**；命令列自动对齐。
- `cmd.PrintCmdHelp(pro)`：打印命令名、Desc、Usage、Options 表，**按参数 key 排序**；每个选项显示 `-短名, --长名`，带值参数追加 ` <value>`。
- 渲染效果示例（`ls --help`）：
  ```
  ls
  list onedrive directory contents

  Usage:
    ls [OPTION] path

  Options:
    -d, --direct_url  list files direct url
    -h, --help        print help
  ```
  注意 `Usage` 字段里的 `usage:` 前缀会被自动去掉，避免和 "Usage:" 标签重复。
- **颜色**：终端（TTY）下标题加粗、标签青色、命令/参数名绿色；输出被重定向/管道时不加颜色，不影响脚本解析。实现基于 `os.Stdout.Stat()` 的 `ModeCharDevice` 判断。
- 头部显示 `ct.Name` / `ct.Version`：`NewContext()` 默认取包变量 `CmdName`/`VERSION`（构建时用 `-ldflags -X` 注入，见 `main/Makefile`）；也可手动覆盖 `ct.Name`。
- 惯例：每个命令都注册 `h` 参数，在 `Cmd` 开头先判断 `ct.ParamGroupMap["h"] != nil` 就打印帮助并返回。由于 `--help` 现在也能解析，`-h` 和 `--help` 都能触达同一分支。

## 6. 已知限制

1. **不支持组合短选项**（如 `-la`）：会整体当成参数名 `la` 查，查不到就报 `unknown option: -la`。多字符短名（如 `dn`、`ss`）是设计支持的，与组合选项区分不开，习惯用空格分开写。
2. **带值参数的值不能以 `-` 开头**：`-d -foo` 会被当作缺值报错（沿用历史行为）。
3. 参数名只匹配一次（`Name` 优先，再 `LongName`），没有长短名混用下的多义性校验——设计上避免出现一个短名等于另一个长名的情况。
4. **没有声明式"必填参数"字段**（`ParamDef` 不加 `Required`）。"是否必需"是调用时刻的领域规则，不是参数定义属性——`-d` 在 `d` 命令里可选、别处可能必填。用 `ct.Need(...)` / `ct.Any(...)` 在 handler 顶部做判断，组合规则（二选一、条件依赖、互斥）用普通代码表达即可。

## 7. 现有命令速查（oneshow 已注册）

见 `main/oneshow.go` 的 `setFuns()`：

`ls` `rm` `info` `d`(下载) `auth` `u`(上传) `web` `webdav` `users` `su` `saveUser` `who` `search` `mv`

给项目新增子命令时，照 `setFuns()` 里的模式添加即可。参数值若需转 int，参考 `u` 命令里 `strconv.Atoi(p.Value)` 的写法。
