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
``
