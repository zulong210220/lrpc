### panic

```
goroutine 21 [running]:
fmt.(*buffer).writeString(...)
	/usr/local/go/src/fmt/print.go:82
fmt.(*fmt).padString(0xc00008c1e0, 0x0, 0x25)
	/usr/local/go/src/fmt/format.go:110 +0x8c
fmt.(*fmt).fmtS(0xc00008c1e0, 0x0, 0x25)
	/usr/local/go/src/fmt/format.go:359 +0x61
fmt.(*pp).fmtString(0xc00008c1a0, 0x0, 0x25, 0x76)
	/usr/local/go/src/fmt/print.go:447 +0x131
fmt.(*pp).printValue(0xc00008c1a0, 0x16e6840, 0xc00062e258, 0x198, 0x76, 0x2)
	/usr/local/go/src/fmt/print.go:761 +0x2153
fmt.(*pp).printValue(0xc00008c1a0, 0x17730e0, 0xc00062e240, 0x199, 0xc000000076, 0x1)
	/usr/local/go/src/fmt/print.go:810 +0x1ab9
fmt.(*pp).printValue(0xc00008c1a0, 0x16c1c20, 0xc00062e240, 0x16, 0xc000000076, 0x0)
	/usr/local/go/src/fmt/print.go:880 +0x18be
fmt.(*pp).printArg(0xc00008c1a0, 0x16c1c20, 0xc00062e240, 0x76)
	/usr/local/go/src/fmt/print.go:716 +0x292
fmt.(*pp).doPrint(0xc00008c1a0, 0xc000623950, 0x5, 0x5)
	/usr/local/go/src/fmt/print.go:1161 +0xf8
fmt.Fprint(0x18fc6e0, 0xc0002b8240, 0xc000623950, 0x5, 0x5, 0xd3, 0x0, 0x0)
	/usr/local/go/src/fmt/print.go:232 +0x58
github.com/zulong210220/lrpc/log.(*Logger).write(0xc0002b6a10, 0xc00087dda0)
	/Users/lijinya/gowork/src/github.com/zulong210220/lrpc/log/logs.go:214 +0x915
github.com/zulong210220/lrpc/log.(*Logger).start(0xc0002b6a10)
	/Users/lijinya/gowork/src/github.com/zulong210220/lrpc/log/logs.go:153 +0x70
created by github.com/zulong210220/lrpc/log.(*Logger).run
	/Users/lijinya/gowork/src/github.com/zulong210220/lrpc/log/logs.go:162 +0x71
exit status 2
```


```
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x18 pc=0x168003e]

goroutine 1 [running]:
github.com/zulong210220/lrpc/client.(*Client).send(0x0, 0xc000694a50)
	/Users/lijinya/gowork/src/github.com/zulong210220/lrpc/client/client.go:211 +0x4e
github.com/zulong210220/lrpc/client.(*Client).Do(0x0, 0x180e0be, 0x7, 0x16c9760, 0xc000125f70, 0x16d01e0, 0xc000125f58, 0xc0007c4060, 0x1)
	/Users/lijinya/gowork/src/github.com/zulong210220/lrpc/client/client.go:251 +0x1dd
github.com/zulong210220/lrpc/client.(*Client).Call(0x0, 0x1919980, 0xc000124000, 0x180e0be, 0x7, 0x16c9760, 0xc000125f70, 0x16d01e0, 0xc000125f58, 0xc0005bb680, ...)
	/Users/lijinya/gowork/src/github.com/zulong210220/lrpc/client/client.go:257 +0x192
github.com/zulong210220/lrpc/xclient.(*XClient).call(0xc000300690, 0xc0005bb680, 0x18, 0x1919980, 0xc000124000, 0x180e0be, 0x7, 0x16c9760, 0xc000125f70, 0x16d01e0, ...)
	/Users/lijinya/gowork/src/github.com/zulong210220/lrpc/xclient/xclient.go:86 +0xe2
github.com/zulong210220/lrpc/xclient.(*XClient).Call(0xc000300690, 0x1919980, 0xc000124000, 0x180c858, 0x5, 0x180e0be, 0x7, 0x16c9760, 0xc000125f70, 0x16d01e0, ...)
	/Users/lijinya/gowork/src/github.com/zulong210220/lrpc/xclient/xclient.go:98 +0x356
main.main()
	/Users/lijinya/gowork/src/github.com/zulong210220/lrpc/examples/demo1/client.go:41 +0x3fd
exit status 2
```

```
&{328 rpc/lserver.go  3  [Server.handleRequest  :  0xc0007e0060  :  {0x17593e0 0xc0007daa70 409}]} [Server.handleRequest  :  0xc0007e0060  :  <rpc.Args Value>]
false
Server.handleRequest
false
 :
false
&{Foo.Sum 2141 rpc server: request handle timeout 0s}
false
 :
false
{4292 18421264}
&{328 rpc/lserver.go  3  [Server.handleRequest  :  0xc0007e00c0  :  {0x17593e0 0xc0008408a0 409}]} [Server.handleRequest  :  0xc0007e00c0  :  <rpc.Args Value>]
false
Server.handleRequest
false
 :
false
goroutine 8 [running]:
runtime/debug.Stack(0xc0000bb478, 0x1727020, 0x1e0fab0)
	/usr/local/go/src/runtime/debug/stack.go:24 +0x9d
github.com/zulong210220/lrpc/log.(*Logger).start.func1()
	/Users/lijinya/gowork/src/github.com/zulong210220/lrpc/log/logs.go:135 +0x65
panic(0x1727020, 0x1e0fab0)
	/usr/local/go/src/runtime/panic.go:967 +0x15d
fmt.(*buffer).writeString(...)
	/usr/local/go/src/fmt/print.go:82
fmt.(*fmt).padString(0xc0000c8860, 0x0, 0x25)
	/usr/local/go/src/fmt/format.go:110 +0x8c
fmt.(*fmt).fmtS(0xc0000c8860, 0x0, 0x25)
	/usr/local/go/src/fmt/format.go:359 +0x61
fmt.(*pp).fmtString(0xc0000c8820, 0x0, 0x25, 0x76)
	/usr/local/go/src/fmt/print.go:447 +0x131
fmt.(*pp).printValue(0xc0000c8820, 0x16e6f00, 0xc0007e00d8, 0x198, 0x76, 0x2)
	/usr/local/go/src/fmt/print.go:761 +0x2153
fmt.(*pp).printValue(0xc0000c8820, 0x17737a0, 0xc0007e00c0, 0x199, 0xc000000076, 0x1)
	/usr/local/go/src/fmt/print.go:810 +0x1ab9
fmt.(*pp).printValue(0xc0000c8820, 0x16c22e0, 0xc0007e00c0, 0x16, 0x76, 0x0)
	/usr/local/go/src/fmt/print.go:880 +0x18be
fmt.(*pp).printArg(0xc0000c8820, 0x16c22e0, 0xc0007e00c0, 0x76)
	/usr/local/go/src/fmt/print.go:716 +0x292
fmt.(*pp).doPrintln(0xc0000c8820, 0xc0000bbf20, 0x1, 0x1)
	/usr/local/go/src/fmt/print.go:1173 +0xb1
fmt.Fprintln(0x18fe360, 0xc000010018, 0xc0000bbf20, 0x1, 0x1, 0x6, 0x0, 0x0)
	/usr/local/go/src/fmt/print.go:264 +0x58
fmt.Println(...)
	/usr/local/go/src/fmt/print.go:274
github.com/zulong210220/lrpc/log.(*Logger).write(0xc00021c9a0, 0xc000837e00)
	/Users/lijinya/gowork/src/github.com/zulong210220/lrpc/log/logs.go:232 +0xa36
github.com/zulong210220/lrpc/log.(*Logger).start(0xc00021c9a0)
	/Users/lijinya/gowork/src/github.com/zulong210220/lrpc/log/logs.go:169 +0x92
created by github.com/zulong210220/lrpc/log.(*Logger).run
	/Users/lijinya/gowork/src/github.com/zulong210220/lrpc/log/logs.go:178 +0x71

runtime error: invalid memory address or nil pointer dereference
1 /usr/local/go/src/runtime/panic.go 967 true
2 /usr/local/go/src/runtime/panic.go 212 true
3 /usr/local/go/src/runtime/signal_unix.go 687 true
4 /usr/local/go/src/runtime/memmove_amd64.s 187 true
5 /usr/local/go/src/fmt/print.go 82 true
6 /usr/local/go/src/fmt/format.go 110 true
7 /usr/local/go/src/fmt/format.go 359 true
8 /usr/local/go/src/fmt/print.go 447 true
9 /usr/local/go/src/fmt/print.go 761 true
10 /usr/local/go/src/fmt/print.go 810 true
11 /usr/local/go/src/fmt/print.go 880 true
12 /usr/local/go/src/fmt/print.go 716 true
13 /usr/local/go/src/fmt/print.go 1173 true
14 /usr/local/go/src/fmt/print.go 264 true
15 /usr/local/go/src/fmt/print.go 274 true
16 /Users/lijinya/gowork/src/github.com/zulong210220/lrpc/log/logs.go 232 true
17 /Users/lijinya/gowork/src/github.com/zulong210220/lrpc/log/logs.go 169 true
18 /usr/local/go/src/runtime/asm_amd64.s 1373 true
19  0 false
exit status 1
```
