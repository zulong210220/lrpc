## lrpc

自制rpc框架

- 编码解码


fix 
- 数据竞争panic (打印header指针)
- read阻塞,或者全部读取解析出错 设置定长buf 字节流前面写入其长度
- p2c 新加server不连接


粘包 导致阻塞

```
' {"MagicNumber":3927900,"CodecType":"application/json","ConnectTimeout":3000000000,"HandleTimeout":0}Foo.Sum{"Num1":2,"Num2":4} ' 146 <nil> 1024
```
